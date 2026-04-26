# Aura AI engine — FastAPI service powering content-based recommendations.
#
# Two-stage build keeps the runtime image small and avoids re-downloading
# Python wheels on every code change. Dependencies are installed first
# (in a layer keyed by requirements.txt) so source-only edits stay
# under a couple of seconds.

FROM python:3.12-slim AS deps

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# Build deps for psycopg / chromadb / numpy wheels are already provided
# as manylinux wheels for x86_64; only curl/ca-certs are needed for the
# image-level healthcheck.
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install --prefix=/install -r requirements.txt


FROM python:3.12-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    AI_ENGINE_HOST=0.0.0.0 \
    AI_ENGINE_PORT=8090 \
    HF_HOME=/data/hf-cache

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 1000 --create-home --home-dir /home/aura aura \
    && mkdir -p /data/hf-cache \
    && chown -R aura:aura /data /home/aura

COPY --from=deps /install /usr/local
COPY app ./app

USER aura

EXPOSE 8090

HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=6 \
    CMD curl --fail --silent --show-error http://localhost:8090/v1/healthz \
        || curl --fail --silent --show-error http://localhost:8090/openapi.json >/dev/null \
        || exit 1

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8090"]
