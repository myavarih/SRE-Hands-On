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
	@export PATH=$$PATH:$$(go env GOPATH)/bin && \
	mkdir -p services/gen/go && \
	protoc -I api/proto \
		--go_out=. --go_opt=module=github.com/myavarih/sre-hands-on \
		--go-grpc_out=. --go-grpc_opt=module=github.com/myavarih/sre-hands-on \
		api/proto/auth/v1/auth.proto \
		api/proto/order/v1/order.proto \
		api/proto/payment/v1/payment.proto

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
			docker build -t $$svc:latest -f services/$$svc/Dockerfile .; \
		elif [ -f frontend/Dockerfile ] && [ "$$svc" = "frontend" ]; then \
			echo "==> Building Docker image [frontend:latest]..."; \
			docker build -t frontend:latest -f frontend/Dockerfile frontend; \
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
# Data Layer & Storage (CloudNativePG HA PostgreSQL)
# ==============================================================================

.PHONY: deploy-db
deploy-db: ## Apply PostgreSQL user secret and HA cluster manifests
	@echo "==> Applying PostgreSQL Secret & Cluster in sre-data..."
	@kubectl apply -f deploy/k8s/base/data/postgres-secret.yaml
	@kubectl apply -f deploy/k8s/base/data/postgres-cluster.yaml
	@echo "==> Waiting for PostgreSQL HA Cluster [pg-cluster] to become ready..."
	@kubectl wait --namespace sre-data --for=condition=Ready cluster/pg-cluster --timeout=180s || true

.PHONY: db-status
db-status: ## Show CloudNativePG cluster health, pods, PVCs, and endpoints
	@echo "==> CloudNativePG Cluster Status:"
	@kubectl get cluster -n sre-data -o wide
	@echo ""
	@echo "==> PostgreSQL Pods:"
	@kubectl get pods -n sre-data -o wide
	@echo ""
	@echo "==> Persistent Volume Claims:"
	@kubectl get pvc -n sre-data

.PHONY: db-shell
db-shell: ## Open an interactive psql session to the primary PostgreSQL instance
	@kubectl exec -it -n sre-data pg-cluster-1 -- bash -c 'PGPASSWORD=app_password psql -h localhost -U app_user -d app_db'

# ==============================================================================
# Observability & Monitoring (Prometheus, Alertmanager, Grafana)
# ==============================================================================

.PHONY: deploy-monitoring
deploy-monitoring: ## Deploy/Upgrade kube-prometheus-stack, monitors, alerts, and dashboards
	@echo "==> Deploying kube-prometheus-stack in sre-monitoring..."
	@helm upgrade --install k8s-monitoring prometheus-community/kube-prometheus-stack \
		--namespace sre-monitoring \
		-f deploy/k8s/base/monitoring/values/values-prometheus.yaml \
		-f deploy/k8s/base/monitoring/values/values-grafana.yaml \
		-f deploy/k8s/base/monitoring/values/values-alertmanager.yaml
	@echo "==> Applying Scrapers, Alert Rules, Dashboards, and Tracing..."
	@kubectl apply -f deploy/k8s/base/monitoring/monitors/
	@kubectl apply -f deploy/k8s/base/monitoring/alerts/
	@kubectl apply -f deploy/k8s/base/monitoring/dashboards/
	@kubectl apply -f deploy/k8s/base/monitoring/tracing/

.PHONY: monitoring-status
monitoring-status: ## Check status of all monitoring components in sre-monitoring
	@echo "==> Monitoring & Tracing Pods:"
	@kubectl get pods -n sre-monitoring -o wide
	@echo ""
	@echo "==> PrometheusRules:"
	@kubectl get prometheusrules -n sre-monitoring
	@echo ""
	@echo "==> ServiceMonitors & PodMonitors:"
	@kubectl get servicemonitors,podmonitors -n sre-monitoring

.PHONY: grafana-port-forward
grafana-port-forward: ## Forward Grafana UI to localhost:3000 (admin / prom-operator)
	@echo "==> Forwarding Grafana to http://localhost:3000 (admin / prom-operator)..."
	@kubectl port-forward -n sre-monitoring svc/k8s-monitoring-grafana 3000:80

.PHONY: prom-port-forward
prom-port-forward: ## Forward Prometheus UI to localhost:9090
	@echo "==> Forwarding Prometheus to http://localhost:9090..."
	@kubectl port-forward -n sre-monitoring svc/k8s-monitoring-prometheus 9090:9090

# ==============================================================================
# Logging Layer (Elasticsearch, Fluent Bit, Kibana)
# ==============================================================================

.PHONY: deploy-logging
deploy-logging: ## Deploy Elasticsearch, Fluent Bit DaemonSet, and Kibana in sre-logging
	@echo "==> Deploying Logging Stack in sre-logging..."
	@kubectl apply -f deploy/k8s/base/logging/

.PHONY: logging-status
logging-status: ## Check status of all logging components in sre-logging
	@echo "==> Logging Pods & Services:"
	@kubectl get pods,svc -n sre-logging -o wide
	@echo ""
	@echo "==> Elasticsearch Cluster Health & Indices:"
	@kubectl exec -n sre-logging elasticsearch-0 -- curl -s http://localhost:9200/_cat/indices?v || true

.PHONY: kibana-port-forward
kibana-port-forward: ## Forward Kibana UI to localhost:5601
	@echo "==> Forwarding Kibana to http://localhost:5601..."
	@kubectl port-forward -n sre-logging svc/kibana 5601:5601

.PHONY: es-port-forward
es-port-forward: ## Forward Elasticsearch API to localhost:9200
	@echo "==> Forwarding Elasticsearch to http://localhost:9200..."
	@kubectl port-forward -n sre-logging svc/elasticsearch 9200:9200

# ==============================================================================
# Microservices & Frontend Application Layer (sre-app)
# ==============================================================================

.PHONY: deploy-apps
deploy-apps: ## Deploy microservices, frontend UI, and Ingress into sre-app
	@echo "==> Deploying Microservices & Ingress into sre-app..."
	@kubectl apply -f deploy/k8s/base/apps/
	@kubectl apply -f deploy/k8s/base/ingress/

.PHONY: apps-status
apps-status: ## Check status of all microservices, frontend, and Ingress in sre-app
	@echo "==> Application Pods & Services in sre-app:"
	@kubectl get pods,svc,ingress -n sre-app -o wide

.PHONY: traffic-gen
traffic-gen: ## Send synthetic orders through Ingress to generate metrics, traces, and logs
	@echo "==> Generating synthetic order traffic..."
	@TOKEN=$$(curl -s -X POST http://localhost/api/v1/store/auth/login -H "Content-Type: application/json" -d '{"username":"sre_lead","password":"sre-password-123"}' | jq -r .data.token 2>/dev/null || true); \
	if [ -z "$$TOKEN" ] || [ "$$TOKEN" = "null" ]; then \
		echo "Registering user sre_lead..."; \
		curl -s -X POST http://localhost/api/v1/store/auth/register -H "Content-Type: application/json" -d '{"username":"sre_lead","email":"engineer@sre.lab","password":"sre-password-123"}' > /dev/null; \
		TOKEN=$$(curl -s -X POST http://localhost/api/v1/store/auth/login -H "Content-Type: application/json" -d '{"username":"sre_lead","password":"sre-password-123"}' | jq -r .data.token); \
	fi; \
	PROD_IDS=($$(curl -s http://localhost/api/v1/store/products | jq -r '.data[].id')); \
	for i in {1..5}; do \
		PID=$${PROD_IDS[$$((RANDOM % $${#PROD_IDS[@]}))]}; \
		curl -s -X POST http://localhost/api/v1/store/orders -H "Content-Type: application/json" -H "Authorization: Bearer $$TOKEN" -d "{\"items\":[{\"product_id\":\"$$PID\",\"quantity\":1}],\"payment_method\":\"credit_card\"}" | jq -c '{order_id: .data.order_id, status: .data.status, total: .data.total_price_cents}'; \
		sleep 0.3; \
	done

.PHONY: chaos-inject
chaos-inject: ## Inject 1500ms latency delay into payment-service via Admin Gateway
	@echo "==> Injecting 1500ms latency into payment-service..."
	@curl -s -X POST http://localhost/api/v1/admin/chaos/fault-injection -H "Content-Type: application/json" -d '{"latency_ms":1500,"error_rate":0.0,"enabled":true}' | jq .

.PHONY: chaos-reset
chaos-reset: ## Reset chaos faults in payment-service to nominal
	@echo "==> Resetting chaos faults..."
	@curl -s -X POST http://localhost/api/v1/admin/chaos/fault-injection -H "Content-Type: application/json" -d '{"latency_ms":0,"error_rate":0.0,"enabled":false}' | jq .

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean
clean: ## Remove build binaries and test coverage profiles
	@echo "==> Cleaning build artifacts..."
	@rm -rf bin/ dist/ *.out *.coverprofile
