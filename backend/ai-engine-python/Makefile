SHELL := /bin/bash

PYTHON?=python3
TEST_REPORT?=reports/test-report.txt
COVERAGE_FILE?=reports/.coverage

.PHONY: test

test:
	@mkdir -p $(dir $(TEST_REPORT)) $(dir $(COVERAGE_FILE))
	@set -o pipefail; { \
		if $(PYTHON) -m coverage --version >/dev/null 2>&1; then \
			COVERAGE_FILE=$(COVERAGE_FILE) $(PYTHON) -m coverage run --source=app -m unittest discover -s tests; \
			test_status=$$?; \
			echo; \
			echo "Coverage summary:"; \
			COVERAGE_FILE=$(COVERAGE_FILE) $(PYTHON) -m coverage report -m; \
			exit $$test_status; \
		else \
			echo "coverage.py not found; using stdlib trace fallback"; \
			$(PYTHON) scripts/trace_coverage.py --coverdir reports/pytrace --start-dir tests; \
		fi; \
	} 2>&1 | tee $(TEST_REPORT)
