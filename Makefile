SHELL        := /usr/bin/env bash
CLUSTER_NAME := sre-lab
SERVICES     := storefront-gateway admin-gateway auth-service order-service payment-service frontend

.PHONY: help
help: ## Show this help message
	@echo -e "\033[1;36mSRE Hands-On Lab — Developer Makefile\033[0m"
	@echo -e "\033[0;33mUsage:\033[0m make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[32m%-18s\033[0m %s\n", $$1, $$2}'

# ==============================================================================
# Cluster Lifecycle Targets
# ==============================================================================

.PHONY: cluster-up
cluster-up: ## Bootstrap local 3-node Kind cluster, Ingress, and Namespaces
	@echo "==> Creating Kind cluster [$(CLUSTER_NAME)]..."
	@kind get clusters | grep -q "^$(CLUSTER_NAME)$$" || kind create cluster --config deploy/kind/kind-cluster.yaml
	@echo "==> Applying NGINX Ingress Controller..."
	@kubectl apply -f deploy/ingress/nginx-ingress-kind.yaml
	@echo "==> Waiting for Ingress Controller to become ready..."
	@kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
	@echo "==> Applying Namespaces..."
	@kubectl apply -f deploy/k8s/base/namespaces.yaml
	@echo "==> Cluster is ready!"

.PHONY: cluster-down
cluster-down: ## Tear down the local Kind cluster and free all resources
	@echo "==> Deleting Kind cluster [$(CLUSTER_NAME)]..."
	@kind delete cluster --name $(CLUSTER_NAME)

.PHONY: status
status: ## Display cluster nodes, pods across all namespaces, and endpoints
	@echo "==> Nodes:"
	@kubectl get nodes -o wide
	@echo ""
	@echo "==> Pods (All Namespaces):"
	@kubectl get pods -A -o wide

# ==============================================================================
# Code Generation & Testing
# ==============================================================================

.PHONY: proto
proto: ## Compile all Protocol Buffer (.proto) service definitions into Go code
	@echo "==> Generating Protobuf & gRPC Go stubs..."
	@mkdir -p services/gen/go
	@protoc -I api/proto \
		--go_out=services/gen/go --go_opt=paths=source_relative \
		--go-grpc_out=services/gen/go --go-grpc_opt=paths=source_relative \
		api/proto/**/*.proto 2>/dev/null || echo "Note: Add .proto files under api/proto/ to generate stubs."

.PHONY: test
test: ## Run unit tests with race condition detector and coverage
	@echo "==> Running tests..."
	@go test -v -race -cover ./...

# ==============================================================================
# Docker & Image Sideloading
# ==============================================================================

.PHONY: docker-build
docker-build: ## Build Docker images for all microservices locally
	@for svc in $(SERVICES); do \
		if [ -f services/$$svc/Dockerfile ]; then \
			echo "==> Building Docker image [$$svc:latest]..."; \
			docker build -t $$svc:latest -f services/$$svc/Dockerfile services/$$svc; \
		fi \
	done

.PHONY: kind-load
kind-load: ## Sideload built local Docker images into Kind cluster nodes
	@for svc in $(SERVICES); do \
		if docker image inspect $$svc:latest &>/dev/null; then \
			echo "==> Loading $$svc:latest into Kind cluster..."; \
			kind load docker-image $$svc:latest --name $(CLUSTER_NAME); \
		fi \
	done

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean
clean: ## Remove build binaries and test coverage profiles
	@echo "==> Cleaning build artifacts..."
	@rm -rf bin/ dist/ *.out *.coverprofile
