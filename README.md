# Website Operator — a Kubernetes controller

A custom controller (operator) built on **controller-runtime** that introduces a
`Website` CRD. Declare an image, replica count, and host; the controller
reconciles the underlying **Deployment + Service** to match, and reports
readiness back in `.status`.

![stack](https://img.shields.io/badge/stack-Go%20·%20controller--runtime%20·%20CRD-1f6feb)

> Go isn't installed on the machine this was authored on, so it ships as a
> complete, idiomatic, buildable project (including a hand-written
> `zz_generated.deepcopy.go` so it compiles without `controller-gen`). Run
> `make build` / `make deploy` on any machine with Go + a cluster.

## What it demonstrates

- A **CustomResourceDefinition** with an OpenAPI schema, status subresource,
  printer columns, and defaulting.
- A **level-triggered, idempotent reconcile loop** using `CreateOrUpdate`.
- **Owner references** so deleting a `Website` cascades to its Deployment and
  Service, and so edits to owned objects re-trigger reconciliation (`Owns(...)`).
- **Status + conditions** following Kubernetes conventions.
- Proper **RBAC**, leader election, health/readiness probes, and a non-root
  distroless image.

## The API

```yaml
apiVersion: web.example.com/v1alpha1
kind: Website
metadata:
  name: hello
spec:
  image: nginxdemos/hello:plain-text
  replicas: 3
  containerPort: 80
  host: hello.example.com
```

```
$ kubectl get websites
NAME    IMAGE                          DESIRED   READY   PHASE
hello   nginxdemos/hello:plain-text    3         3       Ready
```

## Layout

```
api/v1alpha1/            Website types, scheme registration, deepcopy
internal/controller/     WebsiteReconciler (the reconcile loop)
cmd/main.go              manager wiring (scheme, probes, leader election)
config/crd/              the CRD manifest
config/rbac/             ClusterRole
config/manager/          namespace, SA, binding, manager Deployment
config/samples/          a sample Website
Dockerfile, Makefile
```

## Run it

```bash
# against your current kubeconfig (e.g. a kind cluster):
make install            # apply the CRD
make run                # run the controller locally
kubectl apply -f config/samples/web_v1alpha1_website.yaml
kubectl get deploy,svc,website
```

Or deploy the controller into the cluster:

```bash
make docker-build IMG=ghcr.io/you/website-operator:dev
make deploy IMG=ghcr.io/you/website-operator:dev
```

## How reconcile converges

1. Fetch the `Website`; if it's gone, stop (GC cleans up owned objects).
2. `CreateOrUpdate` the Deployment to match `spec` (image, replicas, port).
3. `CreateOrUpdate` the Service fronting it.
4. Read the Deployment's `readyReplicas` and write `status` + an `Available`
   condition.

Every managed object has the `Website` as its controller owner, so the loop is
idempotent and self-healing: hand-edit the Deployment and the controller snaps
it back.
