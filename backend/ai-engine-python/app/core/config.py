"""Runtime configuration for the AI engine.

All values are read from environment variables with sensible defaults so the
service can boot in development without any extra setup.
"""

from __future__ import annotations

import os

from pydantic import BaseModel


class Settings(BaseModel):
    app_name: str = "Aura AI Engine"
    host: str = "0.0.0.0"
    port: int = 8090

    # Postgres connection used by PostgresProfileReader.
    database_url: str = "postgres://aura:aura@localhost:5432/aura?sslmode=disable"

    # ChromaDB endpoint and collection name.
    chroma_url: str = "http://localhost:8000"
    chroma_collection: str = "aura-items"

    # Embedding model used by EmbeddingProvider.
    # all-MiniLM-L6-v2: 384-dim, ~80 MB, multilingual-friendly enough for an MVP.
    embedding_model: str = "sentence-transformers/all-MiniLM-L6-v2"

    # Defaults for the content-based recommendation pipeline.
    cb_default_limit: int = 20
    cb_per_seed_neighbors: int = 50

    @classmethod
    def from_env(cls) -> "Settings":
        return cls(
            host=os.getenv("AI_ENGINE_HOST", cls.model_fields["host"].default),
            port=int(os.getenv("AI_ENGINE_PORT", cls.model_fields["port"].default)),
            database_url=os.getenv("DATABASE_URL", cls.model_fields["database_url"].default),
            chroma_url=os.getenv("CHROMA_URL", cls.model_fields["chroma_url"].default),
            chroma_collection=os.getenv(
                "CHROMA_COLLECTION", cls.model_fields["chroma_collection"].default
            ),
            embedding_model=os.getenv(
                "EMBEDDING_MODEL", cls.model_fields["embedding_model"].default
            ),
            cb_default_limit=int(
                os.getenv("CB_DEFAULT_LIMIT", cls.model_fields["cb_default_limit"].default)
            ),
            cb_per_seed_neighbors=int(
                os.getenv(
                    "CB_PER_SEED_NEIGHBORS", cls.model_fields["cb_per_seed_neighbors"].default
                )
            ),
        )


_cached: Settings | None = None


def get_settings() -> Settings:
    global _cached
    if _cached is None:
        _cached = Settings.from_env()
    return _cached
