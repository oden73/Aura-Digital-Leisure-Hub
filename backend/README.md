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
