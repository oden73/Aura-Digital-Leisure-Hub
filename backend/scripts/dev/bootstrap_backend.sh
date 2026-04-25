#!/usr/bin/env bash
set -euo pipefail

echo "Bootstrapping backend skeleton..."
echo "1) Go dependencies"
(cd backend/core-go && go mod tidy)

echo "2) Python virtual environment"
python3 -m venv .venv
source .venv/bin/activate
pip install -r backend/ai-engine-python/requirements.txt

echo "3) Local database (optional)"
echo "   Run: backend/scripts/dev/db_up.sh"

echo "Done."
