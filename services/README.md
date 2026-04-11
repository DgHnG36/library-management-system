# Services

This directory contains all microservices that make up the Library Management System. Each service is independently deployable and communicates via gRPC internally, while the Gateway Service exposes a REST API to external clients.

---

## Service Overview

| Service                                         | Language | Port    | Protocol  | Description                                                |
| ----------------------------------------------- | -------- | ------- | --------- | ---------------------------------------------------------- |
| [gateway-service](./gateway-service/)           | Go       | `8080`  | REST/HTTP | API Gateway — routes requests, handles auth, rate limiting |
| [user-service](./user-service/)                 | Go       | `40041` | gRPC      | User management, authentication, JWT issuance              |
| [book-service](./book-service/)                 | Go       | `40042` | gRPC      | Book catalog, inventory, and availability                  |
| [order-service](./order-service/)               | Go       | `40043` | gRPC      | Order lifecycle management                                 |
| [notification-service](./notification-service/) | Python   | —       | RabbitMQ  | Event-driven email notifications                           |

---

## Communication Overview

```
Client
  │
  ▼
Gateway Service (REST :8080)
  ├──gRPC──► User Service (:40041)
  ├──gRPC──► Book Service (:40042)
  └──gRPC──► Order Service (:40043)
                   │
                RabbitMQ
                   │
                   ▼
         Notification Service
```

---

## Quick Reference

- Each service has its own `go.mod` / `requirements.txt` and **Dockerfile**.
- Shared Protobuf-generated stubs live in [`shared/`](../shared/).
- Proto source definitions live in [`proto/`](../proto/).

## See Also

- [k8s/](../k8s/) — Kubernetes manifests
- [test/](../test/) — Unit, integration, e2e, and performance tests
- [monitoring/](../monitoring/) — Prometheus & Grafana stack
