#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"

echo "Starting Postgres (backend compose)..."

ENV_FILE="$ROOT_DIR/backend/.env"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "No backend/.env found; using backend/.env.example defaults."
fi

cd "$ROOT_DIR/backend/deployments"

docker compose --env-file "$ENV_FILE" -f docker-compose.backend.yml up -d postgres

echo "Starting ChromaDB..."
docker compose --env-file "$ENV_FILE" -f docker-compose.backend.yml up -d chroma

echo "Waiting for Postgres to become healthy..."
docker compose --env-file "$ENV_FILE" -f docker-compose.backend.yml ps

echo "Done."

