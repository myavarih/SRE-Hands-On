# SRE Hands-On: Cloud-Native Microservices & Observability Lab

[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.36-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![gRPC](https://img.shields.io/badge/gRPC-v1.65-244c5a?logo=grpc&logoColor=white)](https://grpc.io/)
[![PostgreSQL](https://img.shields.io/badge/CloudNativePG-HA_PostgreSQL_17-336791?logo=postgresql&logoColor=white)](https://cloudnative-pg.io/)
[![Prometheus](https://img.shields.io/badge/Prometheus-v3.14-E6522C?logo=prometheus&logoColor=white)](https://prometheus.io/)
[![Grafana](https://img.shields.io/badge/Grafana-v11.6-F46800?logo=grafana&logoColor=white)](https://grafana.com/)
[![Elasticsearch](https://img.shields.io/badge/Elasticsearch-v8.14-005571?logo=elasticsearch&logoColor=white)](https://www.elastic.co/)
[![ArgoCD](https://img.shields.io/badge/GitOps-ArgoCD_v2.12-EF7B4D?logo=argo&logoColor=white)](https://argo-cd.readthedocs.io/)

A production-grade, cloud-native Site Reliability Engineering (SRE) laboratory demonstrating poly-service Clean Architecture in Go, high-availability database replication, declarative GitOps deployment, and unified 3-pillar observability (Metrics, Distributed Traces, and Structured Logs).

---

## 1. System Architecture

The platform models an e-commerce ecosystem partitioned into presentation edge gateways, domain microservices communicating via gRPC, a synchronous high-availability database cluster, and a dedicated observability fabric.

```
                                [ Public Traffic ]
                                        │
                                        ▼
                        ┌───────────────────────────────┐
                        │   NGINX Ingress Controller    │
                        │    (hostPort 80 / 443)        │
                        └───────────────┬───────────────┘
                                        │
                ┌───────────────────────┴───────────────────────┐
                ▼                                               ▼
    ┌───────────────────────┐                       ┌───────────────────────┐
    │  storefront-gateway   │ (HTTP :8080)          │     admin-gateway     │ (HTTP :8081)
    │  Public REST API      │                       │  Admin & Chaos API    │
    └───────────┬───────────┘                       └───────────┬───────────┘
                │                                               │
                │             gRPC (:50051, :50052, :50053)     │
                ├───────────────────────┬───────────────────────┤
                ▼                       ▼                       ▼
    ┌───────────────────────┐ ┌───────────────────┐ ┌───────────────────────┐
    │     auth-service      │ │   order-service   │ │    payment-service    │
    │  gRPC :50051          │ │   gRPC :50052     │ │  gRPC :50053          │
    │  JWT & User Identity  │ │   Catalog & UoW   │ │  Transactions & Chaos │
    └───────────┬───────────┘ └─────────┬─────────┘ └───────────┬───────────┘
                │                       │                       │
                └───────────────────────┼───────────────────────┘
                                        │
                         SQL (Read/Write Split via Pooler)
                                        │
                ┌───────────────────────┴───────────────────────┐
                ▼                                               ▼
    ┌───────────────────────┐                       ┌───────────────────────┐
    │     pg-cluster-rw     │                       │     pg-cluster-ro     │
    │  Primary Instance     │ ══════ Sync Rep ════> │  Standby Instance     │
    │  (sre-lab-worker)     │      (RPO = 0)        │  (sre-lab-worker2)    │
    └───────────────────────┘                       └───────────────────────┘
```

### Microservices & Gateway Topology

| Service | Protocol / Port | Role & Responsibility | Key Dependencies |
| :--- | :--- | :--- | :--- |
| **`storefront-gateway`** | HTTP `:8080` | Customer-facing API gateway; handles user registration, catalog, and checkout. | Gin, OpenTelemetry, gRPC clients |
| **`admin-gateway`** | HTTP `:8081` | Internal operations gateway; exposes system inventory and chaos fault-injection endpoints. | Gin, OpenTelemetry, Chaos Hook |
| **`auth-service`** | gRPC `:50051` | Issues signed JWT tokens, verifies credentials, and manages user accounts. | bcrypt, JWT-Go, PostgreSQL |
| **`order-service`** | gRPC `:50052` | Orchestrates order placement, stock verification, and Unit-of-Work transactions. | GORM, Protobuf v1, OpenTelemetry |
| **`payment-service`** | gRPC `:50053` | Processes payments and features an SRE chaos hook for dynamic latency injection. | GORM, Chaos Interceptor, OTel |
| **`frontend`** | HTTP `:80` | Vanilla HTML5/JS dashboard for interactive order creation and chaos control. | NGINX, Storefront REST API |

---

## 2. Full-Stack Observability & SRE Pipeline

The cluster implements the **Three Pillars of Observability** with correlation across metrics, traces, and logs:

```
                            ┌─────────────────────────────────┐
                            │      Grafana (Port :3000)       │
                            │  Unified Visualization & RED    │
                            └────┬───────────────┬────────────┘
                                 │               │
                 PromQL / Metrics│               │LogQL / Traces
                                 ▼               ▼
    ┌──────────────────────────────┐   ┌──────────────────────────────┐
    │     Prometheus Operator      │   │        Grafana Tempo         │
    │  Scrapes :9090 /metrics      │   │  OTLP gRPC Distributed Trace │
    └──────────────▲───────────────┘   └──────────────▲───────────────┘
                   │                                  │
    ┌──────────────┴───────────────┐   ┌──────────────┴───────────────┐
    │      Fluent Bit DaemonSet    │   │    OpenTelemetry Collector   │
    │  Tail /var/log/containers    │   │  Pipelines & Span Batching   │
    └──────────────┬───────────────┘   └──────────────▲───────────────┘
                   │                                  │
                   ▼                                  │
    ┌──────────────────────────────┐                  │
    │    Elasticsearch & Kibana    │                  │
    │  Indices: fluent-bit-*       │                  │
    └──────────────────────────────┘                  │
                   ▲                                  │
                   └──────────────────────────────────┘
                              Application Pods
                    (slog JSON Logs + OTel Trace Context)
```

1. **Metrics (Prometheus & Alertmanager)**:
   - Scrapes all application pods and gateways via declarative `ServiceMonitor` definitions.
   - Custom Grafana Dashboard: **SRE Lab: Microservices Golden Signals (RED)** tracking Rate (RPS), Errors (5xx ratios), and Latency (P50, P90, P99).
2. **Distributed Tracing (OpenTelemetry & Tempo)**:
   - B3 / W3C TraceContext propagation across HTTP gateways and downstream gRPC services.
   - Spans exported via OTLP (`otel-collector:4317`) into Grafana Tempo.
3. **Log Aggregation (Fluent Bit, Elasticsearch & Kibana)**:
   - High-throughput Fluent Bit DaemonSet parsing CRI container log format.
   - Daily indices (`fluent-bit-YYYY.MM.DD`) in Elasticsearch with automated Kibana Data Views and Log Stream views.
   - Unified `message` runtime mapping for both Go `slog` JSON (`msg`) and raw container stdout (`log`).

---

## 3. End-to-End System Verification (Live Logs)

### A. High-Availability Database Layer (`sre-data`)

The database runs as an automated 2-instance HA cluster managed by CloudNativePG with synchronous replication ($RPO=0$):

```console
$ make db-status
==> CloudNativePG Cluster Status:
NAME         AGE   INSTANCES   READY   STATUS                     PRIMARY
pg-cluster   24h   2           2       Cluster in healthy state   pg-cluster-1

==> PostgreSQL Pods:
NAME           READY   STATUS    IP            NODE              ROLE
pg-cluster-1   1/1     Running   10.244.1.9    sre-lab-worker    Primary (Read/Write)
pg-cluster-2   1/1     Running   10.244.2.12   sre-lab-worker2   Standby (Read-Only)

==> Services & Split Routing:
pg-cluster-rw   10.96.30.227   5432/TCP   Targets Primary for write transactions
pg-cluster-ro   10.96.71.64    5432/TCP   Targets Standby for read offloading
```

### B. Correlated Order Execution Across Microservices

Dispatching a synthetic order via the Public Storefront Gateway triggers an authenticated, traced transaction spanning three microservices:

#### 1. Public Order Dispatch (`HTTP POST /api/v1/store/orders`):
```json
{
  "statusCode": 201,
  "message": "Order created and payment processed",
  "data": {
    "order_id": "de2ae5fe-b4d3-46c8-8756-ce46c7f44562",
    "status": "COMPLETED",
    "total_price_cents": 18900,
    "transaction_id": "txn_38f139e1-472",
    "user_id": "6397cff8-36d0-4f6c-8b29-0426e446da09",
    "items": [
      {
        "product_id": "d5e4dd9f-9579-4451-bfb7-419bf3100c1f",
        "product_name": "Ergonomic Split Mechanical Keyboard",
        "quantity": 1,
        "unit_price_cents": 18900
      }
    ]
  }
}
```

#### 2. Correlated Structured Logs (Captured from Live Workloads):

* **`storefront-gateway` (HTTP Request & Trace Propagation)**:
```json
{"time":"2026-10-16T09:43:38.616Z","level":"INFO","msg":"HTTP Request Processed","service":"storefront-gateway","status":201,"method":"POST","path":"/api/v1/store/orders","ip":"10.244.0.1","latency_ms":71,"trace_id":"d460760e801e577649a01ae1cfafb6c1","span_id":"4b16517fe55fed72"}
```

* **`order-service` (Inventory Reservation & Unit-of-Work Commit)**:
```json
{"time":"2026-10-16T09:43:38.616Z","level":"INFO","msg":"Order confirmed successfully","service":"order-service","order_id":"de2ae5fe-b4d3-46c8-8756-ce46c7f44562","total_cents":18900}
```

* **`payment-service` (Payment Gateway Execution)**:
```json
{"time":"2026-10-16T09:43:38.609Z","level":"INFO","msg":"Payment processed successfully","service":"payment-service","payment_id":"9b2e98bf-8709-4da9-95ad-ef651aab5546","order_id":"de2ae5fe-b4d3-46c8-8756-ce46c7f44562","amount_cents":18900}
```

* **Elasticsearch Search Verification**:
```json
{
  "_index": "fluent-bit-2026.10.16",
  "_source": {
    "service": "order-service",
    "msg": "Order confirmed successfully",
    "order_id": "de2ae5fe-b4d3-46c8-8756-ce46c7f44562",
    "kubernetes": {
      "pod_name": "order-service-679c9cdbc7-tzqbg",
      "namespace_name": "sre-app"
    }
  },
  "fields": {
    "message": ["Order confirmed successfully"]
  }
}
```

---

## 4. Quickstart & Operational Lifecycle

### Prerequisites

Ensure the following tooling is installed locally:
- **Docker Engine** $\ge$ 24.0
- **Kind** $\ge$ v0.23.0
- **kubectl** $\ge$ v1.30.0
- **Helm 3** $\ge$ v3.14.0
- **Go** $\ge$ 1.24
- **make**, **curl**, and **jq**

---

### Step 1: Provision Local Kubernetes Cluster & Ingress

```bash
make cluster-up
```
*Creates a 3-node Kind cluster (`sre-lab-control-plane`, `sre-lab-worker`, `sre-lab-worker2`), deploys the NGINX Ingress Controller with hostPort mappings (`80:80`, `443:443`), and provisions all namespaces.*

---

### Step 2: Deploy CloudNativePG HA PostgreSQL Cluster

```bash
# 1. Install CloudNativePG Operator CRDs and Controller
kubectl apply --server-side --force-conflicts -f deploy/k8s/base/data/cnpg-1.25.0.yaml
kubectl wait --namespace cnpg-system --for=condition=ready pod --selector=app.kubernetes.io/name=cloudnative-pg --timeout=120s

# 2. Deploy PostgreSQL Secrets & HA Cluster
make deploy-db

# 3. Check Database Health
make db-status
```

---

### Step 3: Deploy Observability & Logging Stack

```bash
# 1. Deploy Prometheus, Alertmanager, Grafana, OpenTelemetry & Tempo
make deploy-monitoring

# 2. Deploy Elasticsearch, Fluent Bit DaemonSet, and Kibana
make deploy-logging

# 3. Verify Health
make monitoring-status
make logging-status
```

---

### Step 4: Build, Sideload & Deploy Microservices

```bash
# 1. Compile Protobufs & Run Unit Tests
make proto
make test

# 2. Build Multi-Service Container Images
make docker-build

# 3. Sideload Local Images into Kind Nodes
make kind-load

# 4. Deploy Microservices, Frontend UI, and Ingress Rules
make deploy-apps

# 5. Check Application Pod Readiness
make apps-status
```

---

### Step 5: Traffic Generation, Chaos Engineering & Port Forwards

#### Access Endpoints
- **Storefront & Admin Web UI**: `http://localhost/` (Direct via Ingress)
- **Grafana**: `make grafana-port-forward` $\rightarrow$ `http://localhost:3000` (`admin` / `prom-operator`)
- **Kibana**: `make kibana-port-forward` $\rightarrow$ `http://localhost:5601`
- **Prometheus**: `make prom-port-forward` $\rightarrow$ `http://localhost:9090`
- **Alertmanager**: `make alertmanager-port-forward` $\rightarrow$ `http://localhost:9093`

#### Traffic Simulation & SRE Chaos Drills
```bash
# Generate synthetic user orders through Ingress
make traffic-gen

# Inject 1500ms latency into payment-service (Chaos Drill)
make chaos-inject

# Generate traffic and observe latency breach in Grafana / Tempo
make traffic-gen

# Reset chaos injection back to nominal
make chaos-reset
```

---

### Step 6: Cluster Teardown

```bash
make cluster-down
```
*Deletes the Kind cluster and releases all allocated Docker network bridges and container volumes.*

---

## 5. Repository Topology

```
.
├── api/proto/                 # Protobuf contracts (auth, order, payment v1)
├── services/                  # Microservices & API Gateways
│   ├── auth-service/          # gRPC Auth service (port :50051, bcrypt, JWT)
│   ├── order-service/         # gRPC Order service (port :50052, catalog, inventory, UoW)
│   ├── payment-service/       # gRPC Payment service (port :50053, SRE chaos hook)
│   ├── storefront-gateway/    # Public REST API Gateway (Gin, port :8080)
│   ├── admin-gateway/         # Internal Admin & Chaos REST Gateway (Gin, port :8081)
│   └── gen/go/                # Generated Go Protobuf & gRPC stubs
├── pkg/                       # Shared Kernel (Hexagonal Ports & Core Utilities)
│   ├── entities/              # Domain models (User, Order, Product, Payment)
│   ├── ports/                 # Abstract repository and Unit-of-Work interfaces
│   ├── persistence/           # GORM adapters with PostgreSQL Read/Write split support
│   ├── exceptions/            # Uniform domain error constructs & gRPC error mapping
│   └── telemetry/             # OpenTelemetry tracer, slog logging, Prometheus metrics
├── frontend/                  # Lightweight static HTML5/JS dashboard & SRE control panel
├── deploy/                    # Declarative Kubernetes & Infrastructure Manifests
│   ├── kind/                  # Kind cluster config (1 control-plane, 2 worker nodes)
│   ├── ingress/               # NGINX Ingress controller for Kind
│   └── k8s/                   # Base & overlay manifests
│       ├── base/data/         # CloudNativePG operator & HA PostgreSQL Cluster
│       ├── base/monitoring/   # kube-prometheus-stack, Tempo, OTel Collector
│       ├── base/logging/      # Elasticsearch, Kibana, Fluent Bit, Kibana Setup Job
│       ├── base/apps/         # Deployments & Services for microservices
│       ├── base/ingress/      # Ingress hostPort routing rules
│       ├── overlays/          # Kustomize environment overlays (local-kind, production)
│       └── argocd/            # ArgoCD GitOps Root Application & App-of-Apps
└── Makefile                   # Developer orchestration lifecycle targets
```
