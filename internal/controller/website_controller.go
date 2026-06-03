/*
Package controller holds the Website reconciler.

Reconcile loop (level-triggered, idempotent):
  1. Fetch the Website. If gone, return (owned objects are garbage-collected via
     owner references).
  2. Ensure a Deployment matching the spec exists; create or update it.
  3. Ensure a Service fronting it exists.
  4. Read back the Deployment's readiness and write it to Website .status.

Because every managed object carries an owner reference to the Website,
deleting the Website cascades, and re-running reconcile always converges to the
same result regardless of how many times it fires.
*/
package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webv1alpha1 "github.com/ChenhaoZhang01/website-operator/api/v1alpha1"
)

// WebsiteReconciler reconciles a Website object.
type WebsiteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=web.example.com,resources=websites,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=web.example.com,resources=websites/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=web.example.com,resources=websites/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *WebsiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	var site webv1alpha1.Website
	if err := r.Get(ctx, req.NamespacedName, &site); err != nil {
		// Not found => deleted; owned objects are cleaned up by GC.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if site.Spec.Replicas < 0 {
		return ctrl.Result{}, fmt.Errorf("replicas must be >= 0")
	}

	if err := r.reconcileDeployment(ctx, &site); err != nil {
		l.Error(err, "reconciling deployment")
		return ctrl.Result{}, err
	}
	if err := r.reconcileService(ctx, &site); err != nil {
		l.Error(err, "reconciling service")
		return ctrl.Result{}, err
	}

	// Refresh status from the managed Deployment.
	var dep appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: site.Name, Namespace: site.Namespace}, &dep); err == nil {
		site.Status.ReadyReplicas = dep.Status.ReadyReplicas
		if dep.Status.ReadyReplicas >= site.Spec.Replicas && site.Spec.Replicas > 0 {
			site.Status.Phase = "Ready"
		} else {
			site.Status.Phase = "Progressing"
		}
		meta := metav1.Condition{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			Reason:             "Progressing",
			Message:            "waiting for pods",
			LastTransitionTime: metav1.Now(),
		}
		if site.Status.Phase == "Ready" {
			meta.Status = metav1.ConditionTrue
			meta.Reason = "AllReplicasReady"
			meta.Message = "all replicas are ready"
		}
		site.Status.Conditions = []metav1.Condition{meta}
		if err := r.Status().Update(ctx, &site); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func labelsFor(name string) map[string]string {
	return map[string]string{"app.kubernetes.io/name": name, "app.kubernetes.io/managed-by": "website-operator"}
}

func (r *WebsiteReconciler) reconcileDeployment(ctx context.Context, site *webv1alpha1.Website) error {
	port := site.Spec.ContainerPort
	if port == 0 {
		port = 8080
	}
	replicas := site.Spec.Replicas

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: site.Name, Namespace: site.Namespace},
	}
	// CreateOrUpdate fetches, applies the mutate fn, then creates or patches.
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		dep.Labels = labelsFor(site.Name)
		dep.Spec.Replicas = &replicas
		dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsFor(site.Name)}
		dep.Spec.Template.ObjectMeta.Labels = labelsFor(site.Name)
		dep.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:  "web",
			Image: site.Spec.Image,
			Ports: []corev1.ContainerPort{{ContainerPort: port}},
		}}
		// Tie lifecycle to the Website so deletion cascades.
		return controllerutil.SetControllerReference(site, dep, r.Scheme)
	})
	return err
}

func (r *WebsiteReconciler) reconcileService(ctx context.Context, site *webv1alpha1.Website) error {
	port := site.Spec.ContainerPort
	if port == 0 {
		port = 8080
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: site.Name, Namespace: site.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = labelsFor(site.Name)
		svc.Spec.Selector = labelsFor(site.Name)
		svc.Spec.Ports = []corev1.ServicePort{{
			Name:       "http",
			Port:       80,
			TargetPort: intstr.FromInt(int(port)),
		}}
		return controllerutil.SetControllerReference(site, svc, r.Scheme)
	})
	return err
}

// SetupWithManager wires the controller to watch Websites and the objects it
// owns, so changes to a managed Deployment/Service re-trigger reconciliation.
func (r *WebsiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1alpha1.Website{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
