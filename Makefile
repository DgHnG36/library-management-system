.PHONY: help proto-all proto-go proto-python proto-clean \
	test-all test-unit test-integration test-e2e test-performance test-performance-load test-performance-stress \
	docker-run-test docker-build-test docker-down-test docker-run docker-build docker-down \ 

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  help                     Show this help message"
	@echo "  proto-all                Generate code from protobuf definitions for all languages"
	@echo "  proto-go                 Generate Go code from protobuf definitions"
	@echo "  proto-python             Generate Python code from protobuf definitions"
	@echo "  proto-clean              Clean generated protobuf code"
	@echo "  test-all                 Run all tests (unit, integration, e2e, performance)"
	@echo "  test-unit                Run unit tests"
	@echo "  test-integration         Run integration tests"
	@echo "  test-e2e                 Run end-to-end tests"
	@echo "  test-performance         Run all performance tests (load and stress)"
	@echo "  test-performance-load    Run load performance tests"
	@echo "  test-performance-stress   Run stress performance tests"
	@echo "  docker-run-test          Run tests in Docker containers"
	@echo "  docker-build-test        Build Docker images for testing"
	@echo "  docker-down-test         Stop and remove Docker containers used for testing"
	@echo "  docker-run               Run the application in Docker containers"
	@echo "  docker-build             Build Docker images for the application"
	@echo "  docker-down              Stop and remove Docker containers for the application"

.proto-all: proto-go proto-python
.proto-go:
	protoc --go_out=. --go-grpc_out=. proto/*.proto
.proto-python:
	protoc --python_out=. proto/*.proto
.proto-clean:
	rm -rf shared/*
.test-all: test-unit test-integration test-e2e test-performance
.test-unit:
	go test -v ./... -tags=unit
.test-integration:
	go test -v ./... -tags=integration
.test-e2e:
	go test -v ./... -tags=e2e
.test-performance: test-performance-load test-performance-stress
.test-performance-load:
	go test -v ./... -tags=performance_load
.test-performance-stress:
	go test -v ./... -tags=performance_stress
.docker-run-test:
	docker-compose -f docker-compose.local.yaml up --build --abort-on-container-exit -d
.docker-build-test:
	docker-compose -f docker-compose.local.yaml build
.docker-down-test:
	docker-compose -f docker-compose.local.yaml down
.docker-run:
	docker-compose -f docker-compose.prod.yaml up --build -d
.docker-build:
	docker-compose -f docker-compose.prod.yaml build
.docker-down:
	docker-compose -f docker-compose.prod.yaml down
