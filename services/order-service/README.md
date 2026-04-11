# Order Service

The **Order Service** manages the full lifecycle of book orders — from creation and status tracking to cancellation. It communicates with the User Service and Book Service via gRPC to validate requests and update inventory, and publishes domain events to RabbitMQ for asynchronous notifications.

**Language:** Go | **Port:** `40043` | **Protocol:** gRPC + RabbitMQ (publisher)

---

## Responsibilities

- Order creation, retrieval, listing, cancellation, and status updates
- Validate user identity via User Service (gRPC)
- Reserve / release book inventory via Book Service (gRPC)
- Publish order events to RabbitMQ (`order-events` exchange)

---

## Order States

```
PENDING ──► APPROVED ──► COMPLETED
    │
    └──► CANCELLED
```

---

## Project Structure

```
order-service/
├── cmd/                    # Main entrypoint
├── internal/
│   ├── applications/       # Use-case / business logic layer
│   ├── broker/             # RabbitMQ publisher
│   ├── config/             # Environment configuration loader
│   ├── handlers/           # gRPC handler implementations
│   ├── models/             # Domain models
│   └── repository/         # Database access layer (GORM)
├── pkg/                    # Shared utilities (logger, etc.)
├── Dockerfile
└── go.mod
```

---

## Key Dependencies

| Package                            | Purpose                   |
| ---------------------------------- | ------------------------- |
| `google.golang.org/grpc`           | gRPC server and client    |
| `gorm.io/gorm` + `driver/postgres` | ORM and PostgreSQL driver |
| `sirupsen/logrus`                  | Structured logging        |

---

## Configuration

| Variable                                                  | Description                     |
| --------------------------------------------------------- | ------------------------------- |
| `SERVER_HOST` / `SERVER_PORT`                             | gRPC bind address               |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection           |
| `USER_SERVICE_ADDR`                                       | gRPC address of User Service    |
| `BOOK_SERVICE_ADDR`                                       | gRPC address of Book Service    |
| `RABBITMQ_URL`                                            | RabbitMQ AMQP connection string |
| `RABBITMQ_EXCHANGE`                                       | Exchange name for order events  |

---

## Running Locally

```bash
docker-compose up -d postgres-order rabbitmq user-service book-service

cd services/order-service
go run ./cmd
```

---

## See Also

- [services/](../) — All services overview
- [services/notification-service/](../notification-service/) — Consumes order events
- [proto/order/](../../proto/order/) — Protobuf definitions for this service
