# Gateway Service

The **Gateway Service** is the single entry point for all external client requests. It acts as a reverse proxy, routing REST HTTP calls to the appropriate internal gRPC microservices. It also handles cross-cutting concerns such as authentication, rate limiting, CORS, and observability.

**Language:** Go | **Port:** `8080`

---

## Responsibilities

- JWT-based authentication and authorization middleware
- Role-based access control (user / manager / admin)
- Per-IP rate limiting (backed by Redis)
- Request ID injection and structured request logging
- Prometheus metrics exposition (`/metrics`)
- Health (`/healthy`) and readiness (`/ready`) probes
- CORS handling

---

## API Routes

### System

| Method | Path       | Auth | Description                                         |
| ------ | ---------- | ---- | --------------------------------------------------- |
| GET    | `/healthy` | —    | Liveness probe                                      |
| GET    | `/ready`   | —    | Readiness probe (checks Redis + all gRPC upstreams) |
| GET    | `/metrics` | —    | Prometheus metrics                                  |

### Auth (`/api/v1/auth`)

| Method | Path                    | Auth | Description             |
| ------ | ----------------------- | ---- | ----------------------- |
| POST   | `/api/v1/auth/register` | —    | Register a new user     |
| POST   | `/api/v1/auth/login`    | —    | Login and receive a JWT |

### Books — Public (`/api/v1/books`)

| Method | Path                | Auth | Description      |
| ------ | ------------------- | ---- | ---------------- |
| GET    | `/api/v1/books`     | —    | List all books   |
| GET    | `/api/v1/books/:id` | —    | Get a book by ID |

### User — Protected (`/api/v1/user`)

| Method | Path                   | Auth | Description        |
| ------ | ---------------------- | ---- | ------------------ |
| GET    | `/api/v1/user/profile` | JWT  | Get own profile    |
| PATCH  | `/api/v1/user/profile` | JWT  | Update own profile |

### Orders — Protected (`/api/v1/orders`)

| Method | Path                        | Auth | Description        |
| ------ | --------------------------- | ---- | ------------------ |
| POST   | `/api/v1/orders`            | JWT  | Create a new order |
| GET    | `/api/v1/orders`            | JWT  | List own orders    |
| GET    | `/api/v1/orders/:id`        | JWT  | Get order by ID    |
| POST   | `/api/v1/orders/:id/cancel` | JWT  | Cancel an order    |

### Management — Manager/Admin (`/api/v1/management`)

| Method | Path                                        | Auth     | Description          |
| ------ | ------------------------------------------- | -------- | -------------------- |
| GET    | `/api/v1/management/users`                  | Manager+ | List all users       |
| POST   | `/api/v1/management/books`                  | Manager+ | Create books         |
| DELETE | `/api/v1/management/books/:id`              | Manager+ | Delete a book        |
| PUT    | `/api/v1/management/books/:id`              | Manager+ | Update a book        |
| PATCH  | `/api/v1/management/books/:id/quantity`     | Manager+ | Update book quantity |
| GET    | `/api/v1/management/books/:id/availability` | Manager+ | Check availability   |
| GET    | `/api/v1/management/orders`                 | Manager+ | List all orders      |
| PATCH  | `/api/v1/management/orders/:id/status`      | Manager+ | Update order status  |

### Admin (`/api/v1/admin`)

| Method | Path                          | Auth  | Description         |
| ------ | ----------------------------- | ----- | ------------------- |
| PATCH  | `/api/v1/admin/users/:id/vip` | Admin | Upgrade user to VIP |
| DELETE | `/api/v1/admin/users`         | Admin | Delete users        |

---

## Project Structure

```
gateway-service/
├── cmd/                  # Main entrypoint
├── internal/
│   ├── clients/          # gRPC client wrappers for upstream services
│   ├── config/           # Environment configuration loader
│   ├── delivery/
│   │   └── http/v1/      # HTTP handlers (user, book, order)
│   ├── dto/              # Request/response data transfer objects
│   ├── middleware/        # Auth, CORS, rate-limit middleware
│   └── router/           # Gin router setup
├── pkg/
│   └── logger/           # Structured logger
├── Dockerfile
└── go.mod
```

---

## Key Dependencies

| Package                    | Purpose                              |
| -------------------------- | ------------------------------------ |
| `gin-gonic/gin`            | HTTP router and middleware framework |
| `redis/go-redis/v9`        | Rate limiting backend                |
| `dgrijalva/jwt-go`         | JWT parsing and validation           |
| `prometheus/client_golang` | Metrics instrumentation              |
| `google.golang.org/grpc`   | gRPC client connections              |

---

## Configuration

The service is configured entirely via environment variables:

| Variable                      | Description                                |
| ----------------------------- | ------------------------------------------ |
| `SERVER_HOST` / `SERVER_PORT` | HTTP bind address (default `0.0.0.0:8080`) |
| `JWT_SECRET`                  | Secret for JWT signing                     |
| `JWT_EXP_MINS`                | Token expiry duration                      |
| `REDIS_ADDR`                  | Redis address for rate limiting            |
| `USER_SERVICE_ADDR`           | gRPC address of User Service               |
| `BOOK_SERVICE_ADDR`           | gRPC address of Book Service               |
| `ORDER_SERVICE_ADDR`          | gRPC address of Order Service              |

---

## Running Locally

```bash
# From the repo root — start all dependencies first
docker-compose up -d redis postgres-user postgres-book postgres-order user-service book-service order-service

# Then run the gateway
cd services/gateway-service
go run ./cmd
```

---

## See Also

- [services/](../) — All services overview
- [proto/](../../proto/) — Protobuf definitions
- [docs/curl-tests.md](../../docs/curl-tests.md) — curl test examples
