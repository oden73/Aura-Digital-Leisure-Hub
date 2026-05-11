SHELL := /bin/bash

# Repo root is the parent of this directory (`backend/`). Prefer its `.venv` when present,
# unless PYTHON is passed on the command line or set in the environment.
REPO_ROOT := $(abspath $(CURDIR)/..)
_REPO_VENV_PYTHON := $(REPO_ROOT)/.venv/bin/python
PYTHON ?= $(shell test -x "$(_REPO_VENV_PYTHON)" && printf '%s' "$(_REPO_VENV_PYTHON)" || printf '%s' python3)

TEST_REPORT?=reports/test-report.txt

.PHONY: test test-with-db

test:
	@mkdir -p $(dir $(TEST_REPORT))
	@set -o pipefail; { \
		echo "== Go core =="; \
		$(MAKE) -C core-go test TEST_REPORT=../reports/test-report.txt COVERAGE_PROFILE=../reports/coverage.out; \
		go_status=$$?; \
		echo; \
		echo "== Python AI engine =="; \
		$(MAKE) -C ai-engine-python test TEST_REPORT=reports/test-report.txt PYTHON="$(PYTHON)"; \
		py_status=$$?; \
		echo; \
		echo "== Summary =="; \
		echo "go_status=$$go_status"; \
		echo "python_status=$$py_status"; \
		if [ $$go_status -ne 0 ] || [ $$py_status -ne 0 ]; then exit 1; fi; \
	} 2>&1 | tee $(TEST_REPORT); \
	test_status=$${PIPESTATUS[0]}; \
	$(PYTHON) scripts/generate_visual_reports.py --reports-dir reports; \
	report_status=$$?; \
	if [ $$test_status -ne 0 ]; then exit $$test_status; fi; \
	exit $$report_status

# Full backend: Go unit + Postgres integration + HTTP e2e + Python + visual reports.
# Requires a running Postgres and INTEGRATION_DATABASE_URL (same as CI), e.g.:
#   export INTEGRATION_DATABASE_URL='postgres://aura:aura@127.0.0.1:5432/aura_test?sslmode=disable'
test-with-db:
	@if [ -z "$$INTEGRATION_DATABASE_URL" ]; then \
		echo "INTEGRATION_DATABASE_URL is not set. Example:"; \
		echo "  export INTEGRATION_DATABASE_URL='postgres://aura:aura@127.0.0.1:5432/aura_test?sslmode=disable'"; \
		exit 1; \
	fi
	@mkdir -p $(dir $(TEST_REPORT))
	@set -o pipefail; { \
		echo "== Go core (unit + integration + e2e) =="; \
		$(MAKE) -C core-go test-with-db TEST_REPORT=../reports/test-report.txt COVERAGE_PROFILE=../reports/coverage.out; \
		go_status=$$?; \
		echo; \
		echo "== Python AI engine =="; \
		$(MAKE) -C ai-engine-python test TEST_REPORT=../reports/test-report.txt PYTHON="$(PYTHON)"; \
		py_status=$$?; \
		echo; \
		echo "== Summary =="; \
		echo "go_status=$$go_status"; \
		echo "python_status=$$py_status"; \
		if [ $$go_status -ne 0 ] || [ $$py_status -ne 0 ]; then exit 1; fi; \
	} 2>&1 | tee $(TEST_REPORT); \
	test_status=$${PIPESTATUS[0]}; \
	$(PYTHON) scripts/generate_visual_reports.py --reports-dir reports; \
	report_status=$$?; \
	if [ $$test_status -ne 0 ]; then exit $$test_status; fi; \
	exit $$report_status
