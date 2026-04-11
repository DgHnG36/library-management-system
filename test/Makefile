.PHONY: test-all test-unit-all test-unit-gateway test-unit-user test-unit-book test-unit-order test-unit-notification \
	test-integration test-integration-all \
	test-e2e test-e2e-all \
	test-performance test-performance-all test-performance-smoke test-performance-load test-performance-stress

TEST_TYPES := unit integration e2e performance

test-all:
	@echo "%%%%%% RUNNING ALL TESTS %%%%%%"
	@for test in $(TEST_TYPES); do \
		echo "RUNNING $$(echo $$test | tr '[:lower:]' '[:upper:]') TESTS..."; \
		$(MAKE) test-$$test-all; \
	done
	@echo "%%%%%% ALL TESTS COMPLETED %%%%%%"

UNIT_TEST_TYPES := gateway user book order notification
test-unit-all:
	@echo "%%%%%% RUNNING ALL UNIT TESTS %%%%%%"
	@for test in $(UNIT_TEST_TYPES); do \
		echo "RUNNING UNIT TESTS FOR $$(echo $$test | tr '[:lower:]' '[:upper:]')..."; \
		$(MAKE) test-unit-$$test; \
	done
	@echo "%%%%%% ALL UNIT TESTS COMPLETED %%%%%%"
test-unit-gateway:
	@go test -v ./unit/test-gateway-service/...
test-unit-user:
	@go test -v ./unit/test-user-service/...
test-unit-book:
	@go test -v ./unit/test-book-service/...
test-unit-order:
	@go test -v ./unit/test-order-service/...
test-unit-notification:
	@pytest -v ./unit/test-notification-service/...	

test-integration:
	@if [ $$(docker-compose -f docker-compose.test.yaml ps -q --filter "status=running" | wc -l) -le 0 ]; then \
		echo "ERROR: No running containers found. Please start the services using 'docker-compose -f docker-compose.test.yaml up' before running integration tests."; \
		exit 1; \
	fi
	@echo "%%%%%% RUNNING GO INTEGRATION TESTS %%%%%%"
	@go test -v ./integration/...
	@echo "%%%%%% GO INTEGRATION TESTS COMPLETED %%%%%%"
	@echo "%%%%%% RUNNING PYTHON INTEGRATION TESTS %%%%%%"
	@python3 -m pytest -v ./integration/...
	@echo "%%%%%% ALL INTEGRATION TESTS COMPLETED %%%%%%"

test-e2e:
	@echo "%%%%%% RUNNING END-TO-END TESTS %%%%%%"
	@bash ./e2e/full_workflow_test.sh
	@echo "%%%%%% END-TO-END TESTS COMPLETED %%%%%%"
test-integration-all: test-integration
test-e2e-all: test-e2e
test-performance-all: test-performance

PERFORMANCE_TEST_TYPES := smoke load stress
test-performance:
	@if [ $$(docker-compose -f docker-compose.test.yaml ps -q --filter "status=running" | wc -l) -le 0 ]; then \
		echo "ERROR: No running containers found. Please start the services using 'docker-compose -f docker-compose.test.yaml up' before running performance tests."; \
		exit 1; \
	fi
	@echo "%%%%%% RUNNING ALL PERFORMANCE TESTS %%%%%%"
	@for test in $(PERFORMANCE_TEST_TYPES); do \
		echo "RUNNING $$(echo $$test | tr '[:lower:]' '[:upper:]') PERFORMANCE TESTS..."; \
		$(MAKE) test-performance-$$test; \
	done
	@echo "%%%%%% ALL PERFORMANCE TESTS COMPLETED %%%%%%"
test-performance-smoke:
	@if [ $$(docker-compose -f docker-compose.test.yaml ps -q --filter "status=running" | wc -l) -le 0 ]; then \
		echo "ERROR: No running containers found. Please start the services using 'docker-compose -f docker-compose.test.yaml up' before running performance tests."; \
		exit 1; \
	fi
	@k6 run -o experimental-prometheus-rw ./performance/smoke_test.js
test-performance-load:
	@if [ $$(docker-compose -f docker-compose.test.yaml ps -q --filter "status=running" | wc -l) -le 0 ]; then \
		echo "ERROR: No running containers found. Please start the services using 'docker-compose -f docker-compose.test.yaml up' before running performance tests."; \
		exit 1; \
	fi
	@k6 run -o experimental-prometheus-rw ./performance/load_test.js
test-performance-stress:
	@if [ $$(docker-compose -f docker-compose.test.yaml ps -q --filter "status=running" | wc -l) -le 0 ]; then \
		echo "ERROR: No running containers found. Please start the services using 'docker-compose -f docker-compose.test.yaml up' before running performance tests."; \
		exit 1; \
	fi
	@k6 run -o experimental-prometheus-rw ./performance/stress_test.js

