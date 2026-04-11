# Tests

This directory contains all automated tests for the Library Management System, organized by test type.

---

## Test Categories

| Category    | Location                         | Tool             | Description                                 |
| ----------- | -------------------------------- | ---------------- | ------------------------------------------- |
| Unit        | [`unit/`](./unit/)               | Go test / pytest | Per-service isolated unit tests             |
| Integration | [`integration/`](./integration/) | Go test / pytest | Cross-service tests against real containers |
| End-to-End  | [`e2e/`](./e2e/)                 | Shell (curl)     | Full workflow scenarios via the REST API    |
| Performance | [`performance/`](./performance/) | k6               | Smoke, load, and stress tests               |

---

## Running Tests

All test commands are available from the **repository root** via `make`:

```bash
# Run all test suites
make test-all

# Unit tests only
make test-unit-all

# Integration tests (requires running test containers)
make test-integration

# End-to-end tests
make test-e2e

# All performance tests
make test-performance

# Individual performance suites
make test-performance-smoke
make test-performance-load
make test-performance-stress
```

---

## Integration Tests

Integration tests require running containers. Start them with:

```bash
make docker-run-test
# or
docker-compose -f test/docker-compose.test.yaml up -d
```

Then run:

```bash
make test-integration
```

Teardown:

```bash
make docker-down-test
```

### Integration Test Coverage

| File                              | Scenario                            |
| --------------------------------- | ----------------------------------- |
| `auth_flow_test.go`               | Registration and login flows        |
| `token_lifecycle_test.go`         | JWT expiry and refresh              |
| `rbac_access_test.go`             | Role-based access control           |
| `book_inventory_test.go`          | Book creation and inventory updates |
| `order_flow_test.go`              | Full order lifecycle                |
| `concurrency_order_test.go`       | Concurrent order stress             |
| `notification_flow_test.go`       | Order event → email notification    |
| `notification_order_flow_test.py` | Python-side notification validation |
| `rate_limit_test.go`              | Rate limiting enforcement           |
| `service_outage_test.go`          | Behaviour under partial outage      |

---

## Performance Tests

Performance tests use [k6](https://k6.io/). Install k6 first:

```bash
# macOS
brew install k6

# Windows (Chocolatey)
choco install k6

# Linux
sudo apt install k6
```

| Script           | Purpose                                       |
| ---------------- | --------------------------------------------- |
| `smoke_test.js`  | Validates the system works under minimal load |
| `load_test.js`   | Sustained load over several minutes           |
| `stress_test.js` | Ramps load beyond expected capacity           |

---

## See Also

- [services/](../services/) — Service source code
