# User Service

The **User Service** manages user accounts, authentication, and authorization. It exposes a gRPC interface consumed exclusively by the Gateway Service, and issues JWTs used across the system.

**Language:** Go | **Port:** `40041` | **Protocol:** gRPC

---

## Responsibilities

- User registration and login
- Password hashing and verification
- JWT access token issuance
- Profile retrieval and update
- Role management (`user`, `manager`, `admin`)
- VIP account management
- Bulk user deletion (admin)

---

## Project Structure

```
user-service/
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

| Package                            | Purpose                   |
| ---------------------------------- | ------------------------- |
| `google.golang.org/grpc`           | gRPC server               |
| `gorm.io/gorm` + `driver/postgres` | ORM and PostgreSQL driver |
| `dgrijalva/jwt-go`                 | JWT token generation      |
| `sirupsen/logrus`                  | Structured logging        |

---

## Configuration

| Variable                                                  | Description                      |
| --------------------------------------------------------- | -------------------------------- |
| `SERVER_HOST` / `SERVER_PORT`                             | gRPC bind address                |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection            |
| `DB_SSL_MODE`                                             | SSL mode (`disable` / `require`) |
| `JWT_SECRET`                                              | Secret for signing JWTs          |
| `JWT_ALGORITHM`                                           | Algorithm (e.g., `HS256`)        |
| `JWT_EXP_MINS`                                            | Token expiry duration            |

---

## Running Locally

```bash
docker-compose up -d postgres-user

cd services/user-service
go run ./cmd
```

---

## See Also

- [services/](../) — All services overview
- [proto/user/](../../proto/user/) — Protobuf definitions for this service
