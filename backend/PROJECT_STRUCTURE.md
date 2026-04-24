# Backend Project Structure

## `core-go`

- `cmd/api` - service entrypoint.
- `internal/app` - composition root; wires dependencies and starts the server.
- `internal/config` - runtime config.
- `internal/transport/http` - API Gateway router + JSON handlers.
- `internal/usecase` - application use cases (`GetRecommendations`,
  `SearchContent`, `UpdateInteraction`, `SyncExternalContent`) and DTOs.
- `internal/domain/entities` - core domain types (`Item`, `BaseItemCriteria`,
  `BookDetails`, `CinemaDetails`, `GameDetails`, `User`, `UserProfile`,
  `Interaction`, `ScoredItem`, `RecommendationFilters`).
- `internal/domain/services/cf` - collaborative filtering module: coordinator,
  user-based pipeline, item-based pipeline, interaction matrix abstraction.
- `internal/domain/services/hybrid` - hybrid ranking module: orchestrator,
  score aggregator, final ranker and pluggable ranking rules.
- `internal/infrastructure/repository/postgres` - repository interfaces for
  users, interactions, metadata and user statistics.
- `internal/infrastructure/external` - external provider adapters
  (Steam/TMDB/Books) and `ExternalData` DTO.
- `internal/infrastructure/clients/ai_engine` - client for the Python AI
  engine (CB scores and reasoning).
- `internal/pkg/filter` - `FilterService` cross-cutting component.
- `internal/pkg/auth` - `AuthenticationService` + `TokenManager` skeleton.

## `ai-engine-python`

- `app/main.py` - FastAPI app entrypoint.
- `app/api/routes` - HTTP endpoints for CB recommendations and embeddings.
- `app/domain/schemas` - request/response contracts and `ItemMetadata` /
  `BookDetails` / `CinemaDetails` / `GameDetails` aligned with
  `docs/predone/data_model.md`.
- `app/services` - content similarity, cold start, LLM reasoning.
- `app/adapters` - vector storage, DB readers, ML provider adapters.

## `contracts`

- `proto` - gRPC contracts between services.
- `openapi` - REST contract placeholders.

## `db/migrations`

- `0001_init.sql` - initial schema covering users, external accounts,
  interactions, base items + media-specific details, and the vector-store
  reference table.

## `deployments`

- `docker-compose.backend.yml` - local run for backend-only services.
