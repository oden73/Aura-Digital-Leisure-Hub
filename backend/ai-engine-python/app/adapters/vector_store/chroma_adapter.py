"""Adapter around chromadb for the content embedding store.

The collection is configured with cosine distance, which combined with
EmbeddingProvider's normalize_embeddings=True yields true cosine similarity
in [0, 1] after the (1 - distance) conversion below.
"""

from __future__ import annotations

from typing import Iterable
from urllib.parse import urlparse


class ChromaVectorStoreAdapter:
    def __init__(self, url: str, collection_name: str) -> None:
        parsed = urlparse(url)
        host = parsed.hostname or "localhost"
        port = parsed.port or 8000
        ssl = parsed.scheme == "https"

        import chromadb
        from chromadb.config import Settings as ChromaSettings

        self._client = chromadb.HttpClient(
            host=host,
            port=port,
            ssl=ssl,
            settings=ChromaSettings(anonymized_telemetry=False),
        )
        self._collection = self._client.get_or_create_collection(
            name=collection_name,
            metadata={"hnsw:space": "cosine"},
        )

    def upsert(self, item_id: str, vector: list[float], metadata: dict | None = None) -> None:
        # Chroma requires non-empty metadata; fall back to a minimal sentinel.
        effective_meta = metadata if metadata else {"item_id": item_id}
        self._collection.upsert(
            ids=[item_id],
            embeddings=[vector],
            metadatas=[effective_meta],
        )

    def get_vectors(self, item_ids: Iterable[str]) -> dict[str, list[float]]:
        ids = list(item_ids)
        if not ids:
            return {}
        result = self._collection.get(ids=ids, include=["embeddings"])
        out: dict[str, list[float]] = {}
        for got_id, vec in zip(result.get("ids", []), result.get("embeddings", []) or []):
            if vec is None:
                continue
            out[got_id] = list(vec)
        return out

    def query(
        self,
        vector: list[float],
        k: int,
        exclude_ids: Iterable[str] | None = None,
    ) -> list[tuple[str, float]]:
        """Return [(item_id, cosine_similarity)] sorted by similarity desc.

        cosine_similarity = 1 - cosine_distance (Chroma stores distance).
        """

        excluded = set(exclude_ids or [])
        res = self._collection.query(
            query_embeddings=[vector],
            n_results=max(k + len(excluded), 1),
            include=["distances"],
        )
        ids = (res.get("ids") or [[]])[0]
        distances = (res.get("distances") or [[]])[0]

        out: list[tuple[str, float]] = []
        for item_id, dist in zip(ids, distances):
            if item_id in excluded:
                continue
            out.append((item_id, 1.0 - float(dist)))
            if len(out) >= k:
                break
        return out
