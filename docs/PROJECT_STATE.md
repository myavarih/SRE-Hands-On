# Project State, Architecture & Session Continuity Guide

This document captures the project's exact architectural state, conventions, user preferences, and implementation roadmap so that any future AI agent or human engineer can seamlessly continue development.

---

## 1. Engineering Principles & User Working Preferences

- **Collaboration Style**: Peer software engineer dynamic. Provide deep technical explanations, root causes, and systems architecture.
- **Execution Protocol**:
  - **Consult & Discuss First**: Before creating files or executing commands, explain the proposed design and wait for explicit confirmation.
  - **Command Trigger**: Only execute steps when the user explicitly instructs (e.g. `"do step X.X"` or `"proceed with step X.X"`).
- **Backend Architecture Standards**:
  - Strict **Clean Architecture** (Hexagonal / Ports & Adapters) following the structure of `Projects/Backend`.
  - **Shared Kernel (`pkg/`)**: Entities, interfaces (ports), domain exceptions, GORM PostgreSQL persistence, and telemetry live centrally in `pkg/`.
  - **Application Layer (`internal/application/usecase/`)**: Contains 100% of the pure business logic and unit-of-work transactions. Zero transport dependencies (no Gin, gRPC, or GORM imports).
  - **Presentation Layer (`internal/presentation/controllers/`)**: Thin transport adapters (`AuthGrpcController`, `OrderGrpcController`, `Receive[T]`, `Respond()`).
  - **Gateways**: Built with **Gin** router, comprehensive middleware pipeline (Auth, CORS, Logging with TraceID, Prometheus, Rate Limiting, Recovery).
  - **Internal Services**: Inter-service communication via **gRPC + Protobuf**.

---

## 2. Completed Milestones (Phases 1 & 2)

### Phase 1: Local Multi-Node Kubernetes & Ingress (COMPLETE)
- [x] Multi-node Kind cluster configuration ([`deploy/kind/kind-cluster.yaml`](file:///home/emmwhy/Projects/SRE_hands_on/deploy/kind/kind-cluster.yaml)) with 1 control-plane (host ports 80/443 mapped) + 2 worker nodes (zone-a, zone-b).
- [x] NGINX Ingress Controller ([`deploy/ingress/nginx-ingress-kind.yaml`](file:///home/emmwhy/Projects/SRE_hands_on/deploy/ingress/nginx-ingress-kind.yaml)) bound to host port 80/443.
- [x] Declarative Namespaces ([`deploy/k8s/base/namespaces.yaml`](file:///home/emmwhy/Projects/SRE_hands_on/deploy/k8s/base/namespaces.yaml)): `sre-app`, `sre-data`, `sre-monitoring`, `sre-logging`, `argocd`.
- [x] Developer Makefile ([`Makefile`](file:///home/emmwhy/Projects/SRE_hands_on/Makefile)) with `cluster-up`, `cluster-down`, `proto`, `test`, `docker-build`, `kind-load`, `status`.

### Phase 2: Domain Modeling, Shared Kernel & Microservices (COMPLETE)
- [x] Protobuf contracts ([`api/proto/auth/v1/auth.proto`](file:///home/emmwhy/Projects/SRE_hands_on/api/proto/auth/v1/auth.proto), [`order.proto`](file:///home/emmwhy/Projects/SRE_hands_on/api/proto/order/v1/order.proto), [`payment.proto`](file:///home/emmwhy/Projects/SRE_hands_on/api/proto/payment/v1/payment.proto)) and compiled Go stubs (`services/gen/go/`).
- [x] Shared Kernel ([`pkg/entities`](file:///home/emmwhy/Projects/SRE_hands_on/pkg/entities), [`pkg/ports`](file:///home/emmwhy/Projects/SRE_hands_on/pkg/ports), [`pkg/exceptions`](file:///home/emmwhy/Projects/SRE_hands_on/pkg/exceptions), [`pkg/persistence`](file:///home/emmwhy/Projects/SRE_hands_on/pkg/persistence), [`pkg/telemetry`](file:///home/emmwhy/Projects/SRE_hands_on/pkg/telemetry)).
- [x] **`auth-service`** (`:50051`): Clean Architecture with `AuthUseCase`, `AuthGrpcController`, `bcrypt`, JWT, and GORM.
- [x] **`payment-service`** (`:50053`): `PaymentUseCase`, `PaymentGrpcController`, and **SRE Chaos Fault Injection hook**.
- [x] **`order-service`** (`:50052`): `OrderUseCase`, `OrderGrpcController`, atomic stock deduction transaction, and payment client coordination.
- [x] **`storefront-gateway`** (`:8080`): Public Gin REST gateway with controllers (`v1/auth`, `v1/product`, `v1/order`), middleware (`auth`, `cors`, `logging`, `prometheus`, `rate_limit`, `recovery`), and router.
- [x] **`admin-gateway`** (`:8081`): Admin Gin REST gateway with `admin_controller`, `chaos_controller`, and metrics.
- [x] Multi-stage Dockerfiles for all 5 services with minimal Alpine runtime footprint.

---

## 3. Remaining Roadmap (Phases 3 to 6)

### Phase 3: Data Layer & Observability Infrastructure
- [x] **CloudNativePG (CNPG) Operator & HA PostgreSQL Cluster** (`deploy/k8s/base/data/`): 2 instances (1 Primary on `sre-lab-worker` + 1 Synchronous Standby replica on `sre-lab-worker2`) in `sre-data` with `minSyncReplicas: 1` ($RPO = 0$), `app_db`, `app_user`, and services `pg-cluster-rw` & `pg-cluster-ro`.
- [x] **`kube-prometheus-stack`** (`deploy/k8s/base/monitoring/`): Prometheus Operator, Prometheus Server (with Exemplars & 2Gi PVC), Alertmanager (1Gi PVC), Grafana (with pre-wired 3-pillar datasources & sidecar dashboard auto-discovery), Node Exporters, `cnpg-cluster-metrics` PodMonitor, `microservices-monitor` ServiceMonitor, and `sre-slo-alerts` / `postgres-alerts` PrometheusRules.
- [ ] Deploy **Grafana Tempo** in `sre-monitoring` for distributed trace storage.
- [ ] Deploy **OpenTelemetry Collector** in `sre-monitoring` (routes OTLP traces $\rightarrow$ Tempo, metrics $\rightarrow$ Prometheus).
- [ ] Deploy **Elasticsearch + Kibana + Fluent Bit** DaemonSet in `sre-logging`.

### Phase 4: Observability Correlation & Frontend UI
- Verify 3-Pillar Correlation in Grafana:
  - Metric latency spikes $\rightarrow$ Click Exemplar $\rightarrow$ Jumps to Tempo Trace $\rightarrow$ Jumps to Elasticsearch Pod Logs.
- Create static Vanilla HTML5/JS Frontend dashboard and NGINX Ingress routes.

### Phase 5: GitOps Delivery with ArgoCD
- Deploy **ArgoCD** in `argocd` namespace.
- Create declarative **Kustomize** base and overlays (`deploy/k8s/base` & `deploy/k8s/overlays/local-kind`).
- Register ArgoCD root application pointing to `https://github.com/myavarih/SRE-Hands-On.git` for auto-reconciliation.

### Phase 6: SRE Scenarios, Load Testing & Chaos Drills
- Run **k6** load testing via Docker / Kubernetes Jobs to simulate traffic surges.
- Configure Horizontal Pod Autoscaler (HPA) for gateways and services.
- Define Prometheus alerting rules (P99 Latency breach, 5xx Error budget burn rate, Postgres replication lag).
- Execute SRE drills:
  1. Database Primary failover drill.
  2. Chaos latency/error injection via `POST /api/v1/admin/chaos/fault-injection`.
  3. Worker node drain/kill simulation (`docker stop kind-worker`).
