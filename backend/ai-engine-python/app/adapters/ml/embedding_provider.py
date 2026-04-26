"""Embedding provider backed by sentence-transformers.

The model is loaded lazily on first use so unit tests that monkey-patch the
provider do not pay the import cost.
"""

from __future__ import annotations

from threading import Lock
from typing import Iterable, Sequence


class EmbeddingProvider:
    """Wraps a sentence-transformers model and exposes batch + single-shot APIs."""

    def __init__(self, model_name: str) -> None:
        self._model_name = model_name
        self._model = None
        self._lock = Lock()

    @property
    def model_name(self) -> str:
        return self._model_name

    def _ensure_model(self):
        if self._model is not None:
            return self._model
        with self._lock:
            if self._model is None:
                from sentence_transformers import SentenceTransformer

                self._model = SentenceTransformer(self._model_name)
        return self._model

    def embed_text(self, text: str) -> list[float]:
        return self.embed_batch([text])[0]

    def embed_batch(self, texts: Sequence[str]) -> list[list[float]]:
        if not texts:
            return []
        model = self._ensure_model()
        vectors = model.encode(
            list(texts),
            convert_to_numpy=True,
            normalize_embeddings=True,  # cosine similarity becomes a dot product
            show_progress_bar=False,
        )
        return [vec.tolist() for vec in vectors]

    @staticmethod
    def cosine(a: Iterable[float], b: Iterable[float]) -> float:
        """Cosine similarity for two raw vectors (used in tests / fallback paths)."""

        import math

        ax = list(a)
        bx = list(b)
        if not ax or not bx or len(ax) != len(bx):
            return 0.0
        dot = 0.0
        na = 0.0
        nb = 0.0
        for x, y in zip(ax, bx):
            dot += x * y
            na += x * x
            nb += y * y
        denom = math.sqrt(na) * math.sqrt(nb)
        if denom == 0.0:
            return 0.0
        return dot / denom
