.PHONY: help proto-all proto-go proto-python proto-clean \
	test-all test-unit-all test-integration test-e2e test-performance test-performance-smoke test-performance-load test-performance-stress \
	docker-run-test docker-build-test docker-down-test \
	docker-run docker-build docker-down

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  help                        Show this help message"
	@echo "  proto-all                   Generate code from protobuf definitions for all languages"
	@echo "  proto-go                    Generate Go code from protobuf definitions"
	@echo "  proto-python                Generate Python code from protobuf definitions"
	@echo "  proto-clean                 Clean generated protobuf code"
	@echo "  test-all                    Run all tests (unit, integration, e2e, performance)"
	@echo "  test-unit-all               Run all unit tests"
	@echo "  test-integration            Run integration tests (requires running test containers)"
	@echo "  test-e2e                    Run end-to-end tests"
	@echo "  test-performance            Run all performance tests (smoke, load, stress)"
	@echo "  test-performance-smoke      Run smoke performance tests"
	@echo "  test-performance-load       Run load performance tests"
	@echo "  test-performance-stress     Run stress performance tests"
	@echo "  docker-run-test             Start test containers (docker-compose.test.yaml)"
	@echo "  docker-build-test           Build Docker images for testing"
	@echo "  docker-down-test            Stop and remove test containers"
	@echo "  docker-run                  Run the application in Docker containers"
	@echo "  docker-build                Build Docker images for the application"
	@echo "  docker-down                 Stop and remove application containers"

# PROTO
proto-all:
	$(MAKE) -C proto/ all
proto-go:
	$(MAKE) -C proto/ go
proto-python:
	$(MAKE) -C proto/ python
proto-clean:
	$(MAKE) -C proto/ clean

# TEST
test-all:
	$(MAKE) -C test/ test-all
test-unit-all:
	$(MAKE) -C test/ test-unit-all
test-integration:
	$(MAKE) -C test/ test-integration
test-e2e:
	$(MAKE) -C test/ test-e2e
test-performance:
	$(MAKE) -C test/ test-performance
test-performance-smoke:
	$(MAKE) -C test/ test-performance-smoke
test-performance-load:
	$(MAKE) -C test/ test-performance-load
test-performance-stress:
	$(MAKE) -C test/ test-performance-stress

# DOCKER (TEST)
docker-run-test:
	docker-compose -f test/docker-compose.test.yaml up --build -d
docker-build-test:
	docker-compose -f test/docker-compose.test.yaml build
docker-down-test:
	docker-compose -f test/docker-compose.test.yaml down

# DOCKER (APP)
docker-run:
	docker-compose -f docker-compose.yaml up --build -d
docker-build:
	docker-compose -f docker-compose.yaml build
docker-down:
	docker-compose -f docker-compose.yaml down

