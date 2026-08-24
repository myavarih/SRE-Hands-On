# SRE Hands-On: Cloud-Native Microservices & Observability Lab

A practical, production-grade hands-on laboratory for Kubernetes, GitOps, Site Reliability Engineering (SRE), and full-stack observability.

## Architecture Overview

- **Edge & Ingress**: NGINX Reverse Proxy + Kubernetes Ingress Controller
- **Gateways (HTTP/REST)**:
  - `storefront-gateway`: Public customer-facing REST API
  - `admin-gateway`: Internal/Ops administration REST API
- **Core Microservices (Go + gRPC)**:
  - `auth-service`: Authentication, RBAC, and user state
  - `order-service`: Catalog, inventory management, and order orchestration
  - `payment-service`: Payment processing and notification dispatch
- **Data Layer**: PostgreSQL with Primary/Standby synchronous replication via CloudNativePG
- **Frontend**: Lightweight vanilla HTML5/JS dashboard

## Observability & SRE Stack

- **Metrics**: Prometheus Operator + Alertmanager + cAdvisor + Node Exporter
- **Traces**: OpenTelemetry (Go OTel SDK + OTel Collector) + Grafana Tempo
- **Logs**: Elasticsearch + Kibana with Fluent Bit log aggregation
- **Visualization**: Grafana (unified Metrics $\leftrightarrow$ Traces $\leftrightarrow$ Logs)
- **GitOps**: ArgoCD with Kustomize overlays
- **Load Testing & Chaos**: k6 load generation & fault injection scenarios
