# Backend Project Structure

## `core-go`

- `cmd/api` - service entrypoint.
- `internal/app` - app bootstrap.
- `internal/config` - runtime config.
- `internal/transport/http` - API Gateway handlers and routing.
- `internal/usecase` - application use cases.
- `internal/domain` - business entities and domain services.
- `internal/infrastructure` - adapters: DB, external APIs, AI-engine client.

## `ai-engine-python`

- `app/main.py` - FastAPI app entrypoint.
- `app/api/routes` - HTTP endpoints for CB recommendations and embeddings.
- `app/domain/schemas` - request/response contracts.
- `app/services` - content similarity, cold start, reasoning.
- `app/adapters` - vector storage, DB readers, ML provider adapters.

## `contracts`

- `proto` - gRPC contracts between services.
- `openapi` - REST contract placeholders.

## `deployments`

- `docker-compose.backend.yml` - local run for backend-only services.
