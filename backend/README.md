# Backend Skeleton

This directory contains the server-side project skeleton split into two services:

- `core-go` - API gateway, orchestration, business use cases, integrations with external APIs.
- `ai-engine-python` - AI-focused service (embeddings, similarity search, cold start, LLM reasoning).
- `contracts` - cross-service contracts (gRPC/OpenAPI placeholders).
- `deployments` - local backend-only infrastructure manifests.
- `scripts` - helper scripts for local backend development.

The structure follows the design artifacts from `docs/predone` and keeps clear layering:

- Application/API layer
- Use cases
- Domain models/services
- Infrastructure adapters

## Database (local)

Schema migrations live in `backend/db/migrations`.

Local Postgres is provided via `backend/deployments/docker-compose.backend.yml` and applies
`0001_init.sql` automatically on the first startup (via `docker-entrypoint-initdb.d`).

Quick start:

- Copy `backend/.env.example` to `backend/.env`
- Run `backend/scripts/dev/db_up.sh`
