# Proto

This directory contains the canonical Protobuf (`.proto`) service and message definitions for the Library Management System. Generated stubs are committed to [`shared/`](../shared/) and should **not** be edited by hand.

---

## Definitions

| File                                       | Service        | Description                               |
| ------------------------------------------ | -------------- | ----------------------------------------- |
| [`user/user.proto`](./user/user.proto)     | `UserService`  | User registration, login, profile, roles  |
| [`book/book.proto`](./book/book.proto)     | `BookService`  | Book catalog and inventory management     |
| [`order/order.proto`](./order/order.proto) | `OrderService` | Order lifecycle management                |
| [`common.proto`](./common.proto)           | —              | Shared message types used across services |

---

## Code Generation

Generated stubs live in [`shared/go/`](../shared/go/) (Go) and [`shared/python/`](../shared/python/) (Python).

Use the root `Makefile` to regenerate:

```bash
# Generate stubs for all languages
make proto-all

# Generate Go stubs only
make proto-go

# Generate Python stubs only
make proto-python

# Remove all generated files
make proto-clean
```

> **Prerequisites:** `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, and `grpcio-tools` must be installed and available on `$PATH`.

---

## Adding a New RPC

1. Edit or create the relevant `.proto` file in this directory.
2. Run `make proto-all` to regenerate stubs.
3. Implement the new handler in the corresponding service under `services/`.

---

## See Also

- [shared/go/](../shared/go/) — Generated Go stubs
- [shared/python/](../shared/python/) — Generated Python stubs
- [services/](../services/) — Services that implement / consume these definitions
