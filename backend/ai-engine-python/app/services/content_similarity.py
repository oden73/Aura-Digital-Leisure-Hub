"""Content-based recommendation engine.

Implements the formula from docs/predone/recomendations.md, section 1.2.4:

    r_hat(u, i) = max_{j in I_u, r_uj > α} ρ(e_i, e_j) * r_uj

with:
- ρ = cosine similarity (chosen via product:question item="cb_metric"),
- α = mean(u) — adaptive per-user threshold (chosen via product:question
  item="cb_alpha"),
- e_* — content embeddings produced by EmbeddingProvider and stored in
  ChromaDB.

Candidates can be supplied explicitly (Go core may pass a CF candidate pool)
or generated on the fly via per-seed ANN queries.
"""

from __future__ import annotations

import math
from dataclasses import dataclass
from typing import Iterable, Protocol


@dataclass(frozen=True)
class CBScoredItem:
    item_id: str
    score: float
    match_reason: str | None = None


class ProfileReader(Protocol):
    def get_user_ratings(self, user_id: str) -> dict[str, float]: ...

    def get_user_mean(self, user_id: str) -> float | None: ...


class VectorStore(Protocol):
    def get_vectors(self, item_ids: Iterable[str]) -> dict[str, list[float]]: ...

    def query(
        self,
        vector: list[float],
        k: int,
        exclude_ids: Iterable[str] | None = None,
    ) -> list[tuple[str, float]]: ...


def _cosine(a: list[float], b: list[float]) -> float:
    if not a or not b or len(a) != len(b):
        return 0.0
    dot = 0.0
    na = 0.0
    nb = 0.0
    for x, y in zip(a, b):
        dot += x * y
        na += x * x
        nb += y * y
    denom = math.sqrt(na) * math.sqrt(nb)
    if denom == 0.0:
        return 0.0
    return dot / denom


class ContentSimilarityEngine:
    """Compute CB scores by max-aggregation over user's positively-rated items."""

    def __init__(
        self,
        profile: ProfileReader,
        store: VectorStore,
        per_seed_neighbors: int = 50,
    ) -> None:
        self._profile = profile
        self._store = store
        self._per_seed_neighbors = per_seed_neighbors

    def score_for_user(
        self,
        user_id: str,
        candidate_ids: list[str] | None = None,
        limit: int = 20,
    ) -> tuple[list[CBScoredItem], dict]:
        ratings = self._profile.get_user_ratings(user_id)
        if not ratings:
            return [], {"reason": "no_ratings"}

        alpha = self._profile.get_user_mean(user_id) or 0.0
        positives = {iid: r for iid, r in ratings.items() if r > alpha}
        if not positives:
            return [], {"reason": "no_positive_ratings", "alpha": alpha}

        seed_vectors = self._store.get_vectors(positives.keys())
        if not seed_vectors:
            return [], {"reason": "no_seed_embeddings", "alpha": alpha}

        # Build the candidate pool: explicit ids or per-seed ANN.
        excluded = set(ratings.keys())
        if candidate_ids:
            candidate_pool = [c for c in candidate_ids if c not in excluded]
        else:
            candidate_pool = self._collect_ann_candidates(seed_vectors, excluded)
        if not candidate_pool:
            return [], {"reason": "no_candidates", "alpha": alpha}

        candidate_vectors = self._store.get_vectors(candidate_pool)
        if not candidate_vectors:
            return [], {"reason": "no_candidate_embeddings", "alpha": alpha}

        # Apply the spec formula: max over positively-rated seeds.
        scored: list[CBScoredItem] = []
        for item_id, e_i in candidate_vectors.items():
            best_score = -math.inf
            best_seed: str | None = None
            for seed_id, e_j in seed_vectors.items():
                rho = _cosine(e_i, e_j)
                value = rho * positives[seed_id]
                if value > best_score:
                    best_score = value
                    best_seed = seed_id
            if best_score == -math.inf:
                continue
            reason = (
                f"similar to your highly-rated item {best_seed}"
                if best_seed is not None
                else None
            )
            scored.append(CBScoredItem(item_id=item_id, score=best_score, match_reason=reason))

        scored.sort(key=lambda s: (-s.score, s.item_id))
        return scored[: max(limit, 0)], {
            "alpha": alpha,
            "positives": len(positives),
            "candidates": len(candidate_pool),
        }

    def _collect_ann_candidates(
        self,
        seed_vectors: dict[str, list[float]],
        excluded: set[str],
    ) -> list[str]:
        seen: set[str] = set()
        out: list[str] = []
        for vec in seed_vectors.values():
            hits = self._store.query(
                vector=vec,
                k=self._per_seed_neighbors,
                exclude_ids=excluded,
            )
            for cid, _score in hits:
                if cid in seen or cid in excluded:
                    continue
                seen.add(cid)
                out.append(cid)
        return out
