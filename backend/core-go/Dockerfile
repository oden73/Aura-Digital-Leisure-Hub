# Aura Go core — REST API + migrations runner in one slim image.
#
# Build context MUST be `backend/` (the parent directory) so that we
# can bake the SQL migration files (backend/db/migrations) into the
# image alongside the Go binaries. The matching docker-compose service
# already declares the correct context.
#
#   docker build -f backend/core-go/Dockerfile backend/

FROM golang:1.23-alpine AS builder

ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOFLAGS=-trimpath

WORKDIR /src

# Cache module downloads independently of source edits.
COPY core-go/go.mod core-go/go.sum ./core-go/
RUN cd core-go && go mod download

# Copy source AFTER deps so most rebuilds skip the download step.
COPY core-go ./core-go
COPY db/migrations ./db/migrations

WORKDIR /src/core-go
RUN go build -o /out/api      ./cmd/api \
 && go build -o /out/migrate  ./cmd/migrate \
 && go build -o /out/backfill ./cmd/backfill-embeddings


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S aura && adduser -S aura -G aura

WORKDIR /app

COPY --from=builder /out/api               /usr/local/bin/api
COPY --from=builder /out/migrate           /usr/local/bin/migrate
COPY --from=builder /out/backfill          /usr/local/bin/backfill-embeddings
COPY --from=builder /src/db/migrations     /app/migrations

USER aura

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=6 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/livez || exit 1

ENTRYPOINT ["api"]
