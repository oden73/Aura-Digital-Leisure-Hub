SHELL := /bin/bash

TEST_REPORT?=reports/test-report.txt

.PHONY: test

test:
	@mkdir -p $(dir $(TEST_REPORT))
	@set -o pipefail; { \
		echo "== Go core =="; \
		$(MAKE) -C core-go test TEST_REPORT=reports/test-report.txt; \
		go_status=$$?; \
		echo; \
		echo "== Python AI engine =="; \
		$(MAKE) -C ai-engine-python test TEST_REPORT=reports/test-report.txt; \
		py_status=$$?; \
		echo; \
		echo "== Summary =="; \
		echo "go_status=$$go_status"; \
		echo "python_status=$$py_status"; \
		if [ $$go_status -ne 0 ] || [ $$py_status -ne 0 ]; then exit 1; fi; \
	} 2>&1 | tee $(TEST_REPORT)
