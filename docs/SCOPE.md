# Microservices Architecture & API Specification

This document details the functional scope, API contracts, inter-service gRPC interfaces, database schemas, and SRE observability hooks for the **SRE Hands-On** platform.

---

## 1. System Overview & Domain Model

The platform simulates a high-throughput **FlashSale & Order Processing System** composed of:
- **2 HTTP Gateways**: Public storefront client API + Backoffice Admin & Chaos API.
- **3 Go Microservices**: gRPC-based domain services for Auth, Catalog/Orders, and Payments/Notifications.
- **1 HA PostgreSQL Database**: Managed via CloudNativePG with synchronous streaming replication.
- **Observability Layer**: OTel Tracing (Tempo), Prometheus Metrics (Golden Signals), Elasticsearch Logs.

```
                      ┌──────────────────────────────────────────────┐
                      │              NGINX Ingress (:80)             │
                      └──────────────┬──────────────┬────────────────┘
                                     │              │
                   /api/v1/store/*   │              │  /api/v1/admin/*
                                     ▼              ▼
                    ┌─────────────────────┐    ┌─────────────────────┐
                    │ storefront-gateway  │    │    admin-gateway    │
                    │   (HTTP Port 8080)  │    │   (HTTP Port 8081)  │
                    └──────────┬──────────┘    └──────────┬──────────┘
                               │                          │
                 ┌─────────────┼─────────────┐            │
                 │ gRPC        │ gRPC        │ gRPC       │ gRPC
                 ▼             ▼             │            ▼
      ┌────────────────┐ ┌────────────────┐  │  ┌────────────────────┐
      │  auth-service  │ │ order-service  │  │  │  payment-service   │
      │  (Port :50051) │ │ (Port :50052)  │  │  │  (Port :50053)     │
      └───────┬────────┘ └───────┬────────┘  │  └─────────┬──────────┘
              │                  │           └───────────▶│
              │ SQL              │ SQL                    │ SQL
              ▼                  ▼                        ▼
      ┌──────────────────────────────────────────────────────────────┐
      │             CloudNativePG PostgreSQL HA Cluster              │
      │             (pg-cluster-rw / pg-cluster-ro :5432)            │
      └──────────────────────────────────────────────────────────────┘
```

---

## 2. Gateway Specifications

### 2.1 Storefront Gateway (`storefront-gateway` — Port `8080`)
Exposes external RESTful endpoints to browser clients and load testing tools.

| Method | Endpoint | Auth | Description |
| :--- | :--- | :---: | :--- |
| `POST` | `/api/v1/store/auth/register` | Public | Registers a new customer account. |
| `POST` | `/api/v1/store/auth/login` | Public | Authenticates credentials, returns signed JWT. |
| `GET` | `/api/v1/store/products` | Public | Lists active catalog products with current stock. |
| `GET` | `/api/v1/store/products/{id}` | Public | Fetches product details. |
| `POST` | `/api/v1/store/orders` | **JWT** | Submits order; reserves stock and executes payment. |
| `GET` | `/api/v1/store/orders` | **JWT** | Lists order history for the authenticated user. |
| `GET` | `/api/v1/store/orders/{id}` | **JWT** | Fetches detailed order and payment receipt. |
| `GET` | `/healthz` | Public | Liveness/readiness health check probe. |
| `GET` | `/metrics` | Public | Prometheus scrape endpoint. |

#### Request & Response Schemas

##### `POST /api/v1/store/auth/register`
- **Request Body**:
  ```json
  {
    "username": "johndoe",
    "email": "john@example.com",
    "password": "SecurePassword123!"
  }
  ```
- **Response `201 Created`**:
  ```json
  {
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "username": "johndoe",
    "email": "john@example.com"
  }
  ```

##### `POST /api/v1/store/auth/login`
- **Request Body**:
  ```json
  {
    "username": "johndoe",
    "password": "SecurePassword123!"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6...",
    "expires_in": 86400,
    "user": {
      "id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
      "username": "johndoe",
      "role": "customer"
    }
  }
  ```

##### `POST /api/v1/store/orders`
- **Headers**: `Authorization: Bearer <JWT_TOKEN>`
- **Request Body**:
  ```json
  {
    "items": [
      {
        "product_id": "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22",
        "quantity": 2
      }
    ],
    "payment_method": "credit_card"
  }
  ```
- **Response `201 Created`**:
  ```json
  {
    "order_id": "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380c33",
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "total_price_cents": 5998,
    "status": "COMPLETED",
    "payment": {
      "transaction_id": "tx_99281a8b",
      "status": "SUCCESS"
    },
    "created_at": "2026-08-24T18:00:00Z"
  }
  ```

---

### 2.2 Admin Gateway (`admin-gateway` — Port `8081`)
Exposes operational, inventory management, and SRE chaos injection endpoints.

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/v1/admin/stats` | System overview (active users, total orders, revenue). |
| `POST` | `/api/v1/admin/products` | Create catalog product or refill inventory stock. |
| `POST` | `/api/v1/admin/chaos/fault-injection` | **SRE Chaos Hook**: Injects latency/errors into downstream services. |
| `GET` | `/healthz` | Liveness/readiness probe. |
| `GET` | `/metrics` | Prometheus metrics endpoint. |

#### SRE Chaos Fault Injection Payload
- **`POST /api/v1/admin/chaos/fault-injection`**:
  ```json
  {
    "target_service": "payment-service",
    "latency_ms": 1500,
    "error_rate": 0.25,
    "enabled": true
  }
  ```
  *(Simulates a 25% failure rate and 1.5s delay in the payment service to test circuit breaking and Alertmanager alerts).*

---

## 3. Microservice gRPC Contracts & Protobuf Definitions

### 3.1 Auth Service (`auth.proto` — Port `50051`)
```protobuf
syntax = "proto3";

package auth.v1;
option go_package = "services/gen/go/auth/v1;authv1";

service AuthService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}

message RegisterRequest {
  string username = 1;
  string email = 2;
  string password = 3;
}

message RegisterResponse {
  string user_id = 1;
  string username = 2;
  string email = 3;
}

message LoginRequest {
  string username = 1;
  string password = 2;
}

message LoginResponse {
  string token = 1;
  int64 expires_in = 2;
  string user_id = 3;
  string username = 4;
}

message ValidateTokenRequest {
  string token = 1;
}

message ValidateTokenResponse {
  bool valid = 1;
  string user_id = 2;
  string username = 3;
  string role = 4;
}

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  string user_id = 1;
  string username = 2;
  string email = 3;
  string role = 4;
  string created_at = 5;
}
```

---

### 3.2 Order & Catalog Service (`order.proto` — Port `50052`)
```protobuf
syntax = "proto3";

package order.v1;
option go_package = "services/gen/go/order/v1;orderv1";

service OrderService {
  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse);
  rpc GetProduct(GetProductRequest) returns (GetProductResponse);
  rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse);
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc ListUserOrders(ListUserOrdersRequest) returns (ListUserOrdersResponse);
}

message ProductItem {
  string id = 1;
  string name = 2;
  string description = 3;
  int64 price_cents = 4;
  int32 stock_quantity = 5;
}

message ListProductsRequest {}

message ListProductsResponse {
  repeated ProductItem products = 1;
}

message GetProductRequest {
  string product_id = 1;
}

message GetProductResponse {
  ProductItem product = 1;
}

message CreateProductRequest {
  string name = 1;
  string description = 2;
  int64 price_cents = 3;
  int32 initial_stock = 4;
}

message CreateProductResponse {
  ProductItem product = 1;
}

message OrderItemInput {
  string product_id = 1;
  int32 quantity = 2;
}

message CreateOrderRequest {
  string user_id = 1;
  repeated OrderItemInput items = 2;
  string payment_method = 3;
}

message OrderItemDetail {
  string product_id = 1;
  string product_name = 2;
  int32 quantity = 3;
  int64 unit_price_cents = 4;
}

message CreateOrderResponse {
  string order_id = 1;
  string user_id = 2;
  int64 total_price_cents = 3;
  string status = 4;
  string transaction_id = 5;
  string created_at = 6;
  repeated OrderItemDetail items = 7;
}

message GetOrderRequest {
  string order_id = 1;
}

message GetOrderResponse {
  string order_id = 1;
  string user_id = 2;
  int64 total_price_cents = 3;
  string status = 4;
  string transaction_id = 5;
  string created_at = 6;
  repeated OrderItemDetail items = 7;
}

message ListUserOrdersRequest {
  string user_id = 1;
}

message ListUserOrdersResponse {
  repeated GetOrderResponse orders = 1;
}
```

---

### 3.3 Payment & Notification Service (`payment.proto` — Port `50053`)
```protobuf
syntax = "proto3";

package payment.v1;
option go_package = "services/gen/go/payment/v1;paymentv1";

service PaymentService {
  rpc ProcessPayment(ProcessPaymentRequest) returns (ProcessPaymentResponse);
  rpc GetPaymentStatus(GetPaymentStatusRequest) returns (GetPaymentStatusResponse);
  rpc SendNotification(SendNotificationRequest) returns (SendNotificationResponse);
  rpc SetChaosConfig(SetChaosConfigRequest) returns (SetChaosConfigResponse);
}

message ProcessPaymentRequest {
  string order_id = 1;
  string user_id = 2;
  int64 amount_cents = 3;
  string payment_method = 4;
}

message ProcessPaymentResponse {
  string payment_id = 1;
  string order_id = 2;
  string transaction_ref = 3;
  string status = 4; // SUCCESS or FAILED
  string message = 5;
  string processed_at = 6;
}

message GetPaymentStatusRequest {
  string payment_id = 1;
}

message GetPaymentStatusResponse {
  string payment_id = 1;
  string order_id = 2;
  int64 amount_cents = 3;
  string status = 4;
  string transaction_ref = 5;
}

message SendNotificationRequest {
  string user_id = 1;
  string recipient = 2;
  string channel = 3; // EMAIL or SMS
  string message = 4;
}

message SendNotificationResponse {
  string notification_id = 1;
  string status = 2; // SENT or FAILED
}

message SetChaosConfigRequest {
  int32 latency_ms = 1;
  double error_rate = 2; // 0.0 to 1.0
  bool enabled = 3;
}

message SetChaosConfigResponse {
  bool success = 1;
  string current_status = 2;
}
```

---

## 4. PostgreSQL Database Schema (GORM Models)

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(64) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'customer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Products table
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    stock_quantity INT NOT NULL CHECK (stock_quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Orders table
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    total_price_cents BIGINT NOT NULL CHECK (total_price_cents >= 0),
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    transaction_id VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Order items table
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Payments table
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    user_id UUID NOT NULL REFERENCES users(id),
    amount_cents BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL,
    transaction_ref VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    recipient VARCHAR(255) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_payments_order_id ON payments(order_id);
```

---

## 5. SRE Observability & Golden Signals

| Signal | Metric Name | Metric Type | Labels / Context |
| :--- | :--- | :--- | :--- |
| **Latency** | `http_request_duration_seconds` | Histogram | `handler`, `method`, `status` + OTel Trace Exemplars |
| **Latency** | `grpc_server_handling_seconds` | Histogram | `grpc_service`, `grpc_method`, `grpc_code` |
| **Traffic** | `http_requests_total` | Counter | `handler`, `method`, `status` |
| **Traffic** | `grpc_server_handled_total` | Counter | `grpc_service`, `grpc_method`, `grpc_code` |
| **Errors** | `http_requests_total{status=~"5.."}` | Counter | 5xx server error rate |
| **Errors** | `grpc_server_handled_total{grpc_code!="OK"}`| Counter | Failed RPC error rate |
| **Saturation**| `gorm_pool_in_use_connections` | Gauge | DB pool saturation |
| **Saturation**| `container_cpu_usage_seconds_total` | Counter | Pod CPU utilization |
| **Saturation**| `container_memory_working_set_bytes` | Gauge | Pod memory pressure |
| **Replication**| `pg_stat_replication_bytes_lag` | Gauge | Postgres Primary-to-Standby lag |
