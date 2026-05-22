SHELL := /bin/bash

APP_NAME=core-api
GO?=$(shell command -v go || command -v /usr/local/go/bin/go || command -v /snap/bin/go)
TEST_REPORT?=reports/test-report.txt
COVERAGE_PROFILE?=reports/coverage.out

# Coverage scope for the merged -coverpkg profile:
# - internal/infrastructure/repository/postgres: exercised by integration tests (-tags=integration)
# - cmd/api, cmd/backfill-embeddings: thin entrypoints; behaviour is covered via library packages
COVERPKG?=$(shell $(GO) list ./... | grep -v '/internal/infrastructure/repository/postgres$$' | grep -vE '/cmd/(api|backfill-embeddings)$$' | paste -sd, -)

.PHONY: run build test test-integration test-e2e test-with-db fmt

run:
	$(GO) run ./cmd/api

build:
	$(GO) build -o bin/$(APP_NAME) ./cmd/api

test:
	@mkdir -p $(dir $(TEST_REPORT)) $(dir $(COVERAGE_PROFILE))
	@set -o pipefail; { \
		if [ -z "$(GO)" ]; then \
			echo "go binary not found in PATH, /usr/local/go/bin/go, or /snap/bin/go"; \
			exit 127; \
		fi; \
		$(GO) test ./... -coverpkg=$(COVERPKG) -coverprofile=$(COVERAGE_PROFILE); \
		test_status=$$?; \
		if [ -f "$(COVERAGE_PROFILE)" ]; then \
			echo; \
			echo "Coverage summary:"; \
			$(GO) tool cover -func=$(COVERAGE_PROFILE); \
		fi; \
		exit $$test_status; \
	} 2>&1 | tee $(TEST_REPORT)

# Postgres-backed tests only (requires INTEGRATION_DATABASE_URL).
test-integration:
	@if [ -z "$$INTEGRATION_DATABASE_URL" ]; then \
		echo "INTEGRATION_DATABASE_URL is not set. Example:"; \
		echo "  export INTEGRATION_DATABASE_URL='postgres://aura:aura@127.0.0.1:5432/aura_test?sslmode=disable'"; \
		exit 1; \
	fi
	$(GO) test -tags=integration ./internal/infrastructure/repository/postgres/... -count=1 -timeout=180s

test-e2e:
	@if [ -z "$$INTEGRATION_DATABASE_URL" ]; then \
		echo "INTEGRATION_DATABASE_URL is not set. Example:"; \
		echo "  export INTEGRATION_DATABASE_URL='postgres://aura:aura@127.0.0.1:5432/aura_test?sslmode=disable'"; \
		exit 1; \
	fi
	$(GO) test -tags=e2e ./internal/e2e/... -count=1 -timeout=180s

# Unit tests plus integration and e2e (same DSN as CI).
test-with-db:
	@if [ -z "$$INTEGRATION_DATABASE_URL" ]; then \
		echo "INTEGRATION_DATABASE_URL is not set. Example:"; \
		echo "  export INTEGRATION_DATABASE_URL='postgres://aura:aura@127.0.0.1:5432/aura_test?sslmode=disable'"; \
		exit 1; \
	fi
	@mkdir -p $(dir $(TEST_REPORT)) $(dir $(COVERAGE_PROFILE))
	@set -o pipefail; { \
		if [ -z "$(GO)" ]; then \
			echo "go binary not found in PATH, /usr/local/go/bin/go, or /snap/bin/go"; \
			exit 127; \
		fi; \
		echo "== Go unit tests ./... =="; \
		$(GO) test ./... -coverpkg=$(COVERPKG) -coverprofile=$(COVERAGE_PROFILE); \
		u=$$?; \
		echo; \
		echo "== Go Postgres integration (-tags=integration) =="; \
		$(GO) test -tags=integration ./internal/infrastructure/repository/postgres/... -count=1 -timeout=180s; \
		i=$$?; \
		echo; \
		echo "== Go HTTP e2e (-tags=e2e) =="; \
		$(GO) test -tags=e2e ./internal/e2e/... -count=1 -timeout=180s; \
		e=$$?; \
		echo; \
		echo "== Summary =="; \
		echo "unit_status=$$u integration_status=$$i e2e_status=$$e"; \
		if [ $$u -ne 0 ] || [ $$i -ne 0 ] || [ $$e -ne 0 ]; then exit 1; fi; \
		if [ -f "$(COVERAGE_PROFILE)" ]; then \
			echo; \
			echo "Coverage summary:"; \
			$(GO) tool cover -func=$(COVERAGE_PROFILE); \
		fi; \
		exit 0; \
	} 2>&1 | tee $(TEST_REPORT)

fmt:
	$(GO) fmt ./...
