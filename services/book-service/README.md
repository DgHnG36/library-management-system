# Book Service

The **Book Service** manages the library's book catalog and inventory. It exposes a gRPC interface consumed by both the Gateway Service and the Order Service.

**Language:** Go | **Port:** `40042` | **Protocol:** gRPC

---

## Responsibilities

- Book catalog management (create, update, delete, list, get)
- Inventory and quantity tracking
- Book availability checks
- Stock reservation and release for orders

---

## Project Structure

```
book-service/
├── cmd/                    # Main entrypoint
├── internal/
│   ├── applications/       # Use-case / business logic layer
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

| Package                            | Purpose                      |
| ---------------------------------- | ---------------------------- |
| `google.golang.org/grpc`           | gRPC server                  |
| `gorm.io/gorm` + `driver/postgres` | ORM and PostgreSQL driver    |
| `google/uuid`                      | UUID generation for book IDs |
| `sirupsen/logrus`                  | Structured logging           |

---

## Configuration

| Variable                                                  | Description                      |
| --------------------------------------------------------- | -------------------------------- |
| `SERVER_HOST` / `SERVER_PORT`                             | gRPC bind address                |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection            |
| `DB_SSL_MODE`                                             | SSL mode (`disable` / `require`) |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS`                 | Connection pool sizing           |
| `DB_CONN_MAX_LIFETIME`                                    | Max connection lifetime          |

---

## Running Locally

```bash
docker-compose up -d postgres-book

cd services/book-service
go run ./cmd
```

---

## See Also

- [services/](../) — All services overview
- [proto/book/](../../proto/book/) — Protobuf definitions for this service
