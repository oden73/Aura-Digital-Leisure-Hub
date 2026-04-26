"""Dependency container.

Holds singletons that are expensive to construct (embedding model,
DB connection pool, Chroma client) and exposes them via FastAPI Depends().
"""

from __future__ import annotations

from functools import lru_cache

from app.adapters.db.postgres_reader import PostgresProfileReader
from app.adapters.ml.embedding_provider import EmbeddingProvider
from app.adapters.vector_store.chroma_adapter import ChromaVectorStoreAdapter
from app.core.config import get_settings
from app.services.content_similarity import ContentSimilarityEngine


@lru_cache(maxsize=1)
def get_profile_reader() -> PostgresProfileReader:
    return PostgresProfileReader(get_settings().database_url)


@lru_cache(maxsize=1)
def get_vector_store() -> ChromaVectorStoreAdapter:
    settings = get_settings()
    return ChromaVectorStoreAdapter(
        url=settings.chroma_url, collection_name=settings.chroma_collection
    )


@lru_cache(maxsize=1)
def get_embedding_provider() -> EmbeddingProvider:
    return EmbeddingProvider(get_settings().embedding_model)


@lru_cache(maxsize=1)
def get_content_engine() -> ContentSimilarityEngine:
    settings = get_settings()
    return ContentSimilarityEngine(
        profile=get_profile_reader(),
        store=get_vector_store(),
        per_seed_neighbors=settings.cb_per_seed_neighbors,
    )
