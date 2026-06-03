IMG ?= ghcr.io/ChenhaoZhang01/website-operator:latest

.PHONY: tidy build test fmt vet run docker-build install deploy undeploy generate manifests

tidy:
	go mod tidy

build: fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

run: fmt vet ## Run the controller against your current kubeconfig.
	go run ./cmd/main.go

test: ## Run unit tests (envtest recommended for full integration).
	go test ./... -v

fmt:
	go fmt ./...

vet:
	go vet ./...

docker-build: ## Build the controller image.
	docker build -t ${IMG} .

install: ## Install the CRD into the cluster.
	kubectl apply -f config/crd/

deploy: install ## Deploy RBAC + manager.
	kubectl apply -f config/rbac/role.yaml
	kubectl apply -f config/manager/manager.yaml

undeploy:
	kubectl delete -f config/manager/manager.yaml --ignore-not-found
	kubectl delete -f config/crd/ --ignore-not-found

# These require the kubebuilder toolchain (controller-gen). The committed
# zz_generated.deepcopy.go and config/crd are hand-maintained equivalents so the
# project builds without them.
generate:
	controller-gen object:headerFile= paths=./api/...

manifests:
	controller-gen crd rbac:roleName=website-operator-manager-role paths=./... output:crd:dir=config/crd
