# Comprehensive Project Specification & Developer Preferences

This master document captures all architectural decisions, project requirements, engineering standards, user collaboration rules, and implementation progress for the **SRE Hands-On** platform. Any future AI agent or human contributor must adhere strictly to the guidelines defined below.

---

## 1. Collaboration & Execution Rules (User Preferences)

1. **Peer Software Engineer Dynamic**:
   - Treat the user as a fellow senior software engineer.
   - Communicate with deep technical accuracy, root causes, system mechanics, and architecture rather than high-level over-simplifications.
2. **Consult & Explain Before Doing (Strict Rule)**:
   - Before writing any code, creating any file, or running any terminal command, explain what you are doing in technical detail.
   - Discuss architectural options, trade-offs, and get alignment.
3. **Execution Trigger Rule (Non-Negotiable)**:
   - **NEVER execute any step (no file creation, no command running) unless the user explicitly says `"do step X.X"` or `"proceed with step X.X"`.**
   - Until that exact phrase is provided, all responses must remain purely explanatory, consultative, and discussion-focused.
4. **Actionable Commands**:
   - Provide exact, copy-pasteable bash commands with concise explanations of what each command does.
5. **Explain the "Why"**:
   - Always pinpoint the underlying mechanism, failure mode, and rationale behind architectural choices.

---

## 2. System Architecture & Tech Stack Decisions

### 2.1 Core Infrastructure
- **Host OS & Hardware**: Arch Linux x86_64, 16 GB Host RAM.
- **Local Cluster**: `kind` (Kubernetes in Docker) running a **3-node cluster**:
  - `sre-lab-control-plane`: Binds host ports `80` (HTTP) and `443` (HTTPS) via `extraPortMappings`. Hosts the Ingress Controller (`nodeSelector: ingress-ready: "true"`).
  - `sre-lab-worker` (Zone A): Primary application workloads & PostgreSQL Primary node.
  - `sre-lab-worker2` (Zone B): Secondary workloads, PostgreSQL Standby replica, and Observability stack.
- **Reverse Proxy / Ingress**: NGINX Ingress Controller (using `hostPort: 80/443` on Kind control-plane).
- **Namespaces**:
  - `sre-app`: Go microservices, gateways, and frontend web UI.
  - `sre-data`: CloudNativePG operator and PostgreSQL HA cluster.
  - `sre-monitoring`: Prometheus, Alertmanager, Grafana, Tempo, OpenTelemetry Collector.
  - `sre-logging`: Elasticsearch, Kibana, Fluent Bit DaemonSet.
  - `argocd`: ArgoCD GitOps server and controllers.
  - `ingress-nginx`: NGINX Ingress Controller.

### 2.2 Microservices & Backend Architecture (Clean Architecture)
- **Language**: Go (Golang).
- **Architecture Pattern**: Strict **Clean Architecture (Hexagonal / Ports & Adapters)** adapted from `Projects/Backend`:
  - **Shared Kernel (`pkg/`)**:
    - `pkg/entities`: Pure GORM and domain model definitions (`User`, `Product`, `Order`, `Payment`, `Notification`).
    - `pkg/ports`: Repository interfaces, UnitOfWork contract, and external client interfaces.
    - `pkg/exceptions`: Typed domain errors (`NotFoundError`, `ConflictError`, `ValidationError`, etc.) + bidirectional gRPC status converters (`ToGRPCError`, `FromGRPCError`).
    - `pkg/persistence`: GORM PostgreSQL implementations, connection pool tuning (`MaxOpen: 25`, `MaxIdle: 10`), auto-migrations, and catalog seeder.
    - `pkg/telemetry`: OpenTelemetry SDK initialization (OTLP gRPC exporter), structured `slog` with automatic `trace_id` and `span_id` extraction from context, and Prometheus metrics.
  - **Microservices (`services/auth-service`, `order-service`, `payment-service`)**:
    - `bootstrap/`: `environment.go` (typed env loader) and `app.go` (Composition Root).
    - `internal/application/`:
      - `dto/`: Transport-agnostic Data Transfer Objects.
      - `usecase/`: **100% pure business logic and transaction orchestration.** Zero imports of Gin, gRPC, or GORM.
    - `internal/presentation/controllers/`: Thin gRPC adapters mapping Protobuf messages to DTOs and calling use cases.
  - **Gateways (`services/storefront-gateway`, `admin-gateway`)**:
    - Built using the **Gin** HTTP framework.
    - `internal/presentation/controllers/`:
      - `receive.go`: Generic `Receive[T](ctx)` for JSON, query, and URI binding + validation.
      - `respond.go`: Standard response envelope `{ statusCode, message, data }`.
      - `v1/` controllers: Validates input, calls internal gRPC services with `context.WithTimeout`, and responds.
    - `internal/presentation/middleware/`:
      - `AuthMiddleware`: Local JWT signature verification with gRPC fallback.
      - `CORSMiddleware`: Cross-origin headers.
      - `LoggingMiddleware`: Structured slog with TraceID & SpanID.
      - `PrometheusMiddleware`: Latency histogram (`http_request_duration_seconds`) and request counter (`http_requests_total`).
      - `RateLimitMiddleware`: Token bucket rate limiter with rejection tracking (`gateway_rate_limit_rejections_total`).
      - `RecoveryMiddleware`: Catches panics and returns clean 500 JSON.
    - `internal/presentation/routes/`: `http.go` router assembly and versioned sub-routes (`v1/auth`, `v1/product`, `v1/order`, `v1/admin`).

### 2.3 Protocol Separation
- **External Traffic**: HTTP/REST (JSON) from Browser / Clients $\rightarrow$ NGINX Ingress $\rightarrow$ Gateways.
- **Internal Service-to-Service Traffic**: **gRPC (HTTP/2 + Protobuf binary serialization)** between Gateways and Services, and between `order-service` and `payment-service`.

### 2.4 Data Layer: PostgreSQL HA via CloudNativePG (CNPG)
- **Shared-Nothing Storage**: Each PostgreSQL instance has its own independent PVC / disk.
- **Streaming Replication**: Synchronous WAL streaming (`minSyncReplicas: 1`) from Primary to Standby replica.
- **Routing**:
  - `pg-cluster-rw:5432` $\rightarrow$ Routes writes & strong-consistency reads to the active Primary.
  - `pg-cluster-ro:5432` $\rightarrow$ Load balances read-heavy queries across Standby replicas.
- **Automated Failover**: Self-healing leader promotion upon Primary node failure with RPO = 0.

### 2.5 Observability Stack (3 Pillars + SRE Alerting)
- **Metrics**: Prometheus Operator + Alertmanager + cAdvisor + Node Exporter + Grafana.
- **Traces**: Go OpenTelemetry SDK $\rightarrow$ OTel Collector (OTLP gRPC `:4317`) $\rightarrow$ Grafana Tempo backend.
- **Logs**: Structured JSON stdout $\rightarrow$ Fluent Bit DaemonSet $\rightarrow$ Elasticsearch $\rightarrow$ Kibana & Grafana.
- **Correlation**: TraceID injected in logs; Prometheus Exemplars link metric latency spikes directly to Tempo traces.
- **SRE Fault Injection**: Dynamic chaos endpoint (`POST /api/v1/admin/chaos/fault-injection`) to simulate latency and error spikes.

### 2.6 Container Runtime & Tooling
- **Container Runtime**: Docker / containerd inside Kind.
- **k6 Load Testing**: Run on-demand via Docker or Kubernetes `Job` (no local host package install needed).
- **GitOps**: ArgoCD with declarative Kustomize overlays.

---

## 3. Current Implementation Status

| Milestone | Status | Details |
| :--- | :---: | :--- |
| **Kind 3-Node Cluster** | **Done** | `sre-lab` cluster running with port mappings 80/443. |
| **NGINX Ingress** | **Done** | Deployed with hostPort binding; responds on `localhost:80`. |
| **Namespaces** | **Done** | `sre-app`, `sre-data`, `sre-monitoring`, `sre-logging`, `argocd`. |
| **Developer Makefile** | **Done** | Targets: `cluster-up`, `cluster-down`, `proto`, `test`, `docker-build`, `kind-load`, `status`. |
| **Protobuf Definitions** | **Done** | `auth.proto`, `order.proto`, `payment.proto` compiled to Go stubs. |
| **Shared Kernel (`pkg/`)** | **Done** | `entities`, `ports`, `exceptions`, `persistence` (GORM Postgres), `telemetry` (OTel/Prom/slog). |
| **`auth-service`** | **Done** | Clean Architecture with `AuthUseCase`, `AuthGrpcController`, bcrypt, JWT, and GORM. |
| **`order-service`** | **Done** | `OrderUseCase`, `OrderGrpcController`, atomic stock deduction + payment client coordination. |
| **`payment-service`** | **Done** | `PaymentUseCase`, `PaymentGrpcController`, and SRE Chaos Fault Injection hook. |
| **`storefront-gateway`** | **Done** | Gin REST Gateway with `v1/auth`, `v1/product`, `v1/order`, and full middleware pipeline. |
| **`admin-gateway`** | **Done** | Gin REST Gateway with `v1/admin`, `v1/chaos`, and metrics. |
| **Dockerfiles** | **Done** | Multi-stage Alpine container build files for all 5 services. |
| **CNPG HA PostgreSQL** | **Done** | CloudNativePG v1.25.0, 2 instances (Primary on worker, Standby on worker2) with synchronous WAL streaming, `app_db`, `app_user`, and `pg-cluster-rw` / `pg-cluster-ro` services. |
| **`kube-prometheus-stack`** | **Done** | Prometheus Operator, Prometheus Server (Exemplar storage + 2Gi PVC), Alertmanager (1Gi PVC), Grafana (with 3-pillar datasources & sidecar dashboards), Node Exporters, `cnpg-cluster-metrics` PodMonitor, `microservices-monitor` ServiceMonitor, and `sre-slo-alerts` / `postgres-alerts` PrometheusRules. |
| **Tempo & OTel Collector** | **Done** | Grafana Tempo v2.6.1 distributed tracing storage, OpenTelemetry Collector Contrib v0.114.0 (OTLP gRPC `:4317` receiver, batch processor, memory limiter, Tempo exporter, Prometheus exporter). |
| **Logging Stack (EFK)** | **Done** | Elasticsearch v8.14.3 with 512MB JVM heap & 2Gi PVC, Fluent Bit v3.1.9 DaemonSet on all 3 nodes (CRI log parsing + Kubernetes metadata enrichment), Kibana v8.14.3 UI on port 5601. |
| **Microservices Deployments** | **Done** | Declarative Deployment & Service manifests for `auth-service`, `order-service`, `payment-service`, `storefront-gateway`, `admin-gateway` in `sre-app`. |
| **Frontend UI & Ingress** | **Done** | Vanilla HTML5/JS dashboard frontend on port 80, NGINX Ingress routes for `/`, `/api/v1/store/`, `/api/v1/admin/`. |
| **3-Pillar Correlation** | **Done** | Verified: Prometheus metrics, Tempo distributed traces (`storefront` $\to$ `order` $\to$ `payment`), Elasticsearch logs enriched with `trace_id` & `span_id`, SRE Chaos fault injection. |
| **GitOps with ArgoCD** | **Done** | Kustomize base & local-kind overlay hierarchy, ArgoCD v2.12.3 in `argocd`, declarative App-of-Apps root application with automated reconciliation & self-healing. |

---

## 4. Next Steps Roadmap (Phases 3 to 6)

### Phase 3: Data Layer & Observability Infrastructure
1. Deploy **CloudNativePG Operator** & apply HA PostgreSQL cluster manifest in `sre-data`.
2. Deploy **`kube-prometheus-stack`** in `sre-monitoring`.
3. Deploy **Grafana Tempo** & **OpenTelemetry Collector** in `sre-monitoring`.
4. Deploy **Elasticsearch + Kibana + Fluent Bit** in `sre-logging`.

### Phase 4: Frontend & End-to-End Ingress Integration
1. Build static Vanilla HTML5/JS dashboard frontend (`frontend/index.html`).
2. Write Kubernetes Ingress routing rules mapping `http://localhost/` to frontend, `/api/v1/store/*` to Storefront Gateway, and `/api/v1/admin/*` to Admin Gateway.

### Phase 5: GitOps Delivery (ArgoCD)
1. Deploy ArgoCD in `argocd` namespace.
2. Structure Kustomize manifests (`deploy/k8s/base` and `deploy/k8s/overlays/local-kind`).
3. Connect ArgoCD Application to `https://github.com/myavarih/SRE-Hands-On.git`.

### Phase 6: SRE Hands-on Testing & Chaos Drills
1. Write `k6` load testing scripts to generate synthetic high-concurrency traffic.
2. Configure Horizontal Pod Autoscaler (HPA) to scale pods under CPU / traffic surges.
3. Configure Alertmanager alerts for P99 latency SLO breaches and 5xx error budgets.
4. Execute chaos scenarios (database primary failover, latency injection, worker node shutdown).
