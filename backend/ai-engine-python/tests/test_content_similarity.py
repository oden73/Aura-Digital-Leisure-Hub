"""Unit tests for the CB engine.

Verifies the formula from docs/predone/recomendations.md (1.2.4):

    r_hat(u, i) = max_{j in I_u, r_uj > α} ρ(e_i, e_j) * r_uj

with α = mean(u) and ρ = cosine. Uses tiny in-memory fakes for
PostgresProfileReader and the vector store.

Written with the standard-library unittest framework so it can run without
extra deps (`python3 -m unittest discover`).
"""

from __future__ import annotations

import math
import unittest

from app.services.content_similarity import ContentSimilarityEngine


class FakeProfile:
    def __init__(self, ratings: dict[str, float]) -> None:
        self._ratings = ratings

    def get_user_ratings(self, user_id: str) -> dict[str, float]:
        return dict(self._ratings)

    def get_user_mean(self, user_id: str) -> float | None:
        if not self._ratings:
            return None
        return sum(self._ratings.values()) / len(self._ratings)


class FakeStore:
    def __init__(self, vectors: dict[str, list[float]]) -> None:
        self._vectors = vectors

    def get_vectors(self, item_ids):
        return {iid: list(self._vectors[iid]) for iid in item_ids if iid in self._vectors}

    def query(self, vector, k, exclude_ids=None):
        excluded = set(exclude_ids or [])
        ranked = []
        for iid, vec in self._vectors.items():
            if iid in excluded:
                continue
            denom = math.sqrt(sum(x * x for x in vector)) * math.sqrt(sum(x * x for x in vec))
            sim = 0.0 if denom == 0 else sum(a * b for a, b in zip(vector, vec)) / denom
            ranked.append((iid, sim))
        ranked.sort(key=lambda r: -r[1])
        return ranked[:k]


def _cosine(a, b):
    denom = math.sqrt(sum(x * x for x in a)) * math.sqrt(sum(x * x for x in b))
    if denom == 0:
        return 0.0
    return sum(x * y for x, y in zip(a, b)) / denom


class ContentSimilarityEngineTests(unittest.TestCase):
    def test_max_aggregation_with_explicit_candidates(self):
        # User: liked (above α), neutral, disliked. mean(u)=6, so I_u^+ = {liked}.
        profile = FakeProfile({"liked": 9.0, "neutral": 6.0, "disliked": 3.0})
        store = FakeStore(
            {
                "liked": [1.0, 0.0],
                "near_liked": [0.9, 0.1],
                "far_from_liked": [0.0, 1.0],
            }
        )
        engine = ContentSimilarityEngine(profile=profile, store=store)

        scored, meta = engine.score_for_user(
            user_id="u-1",
            candidate_ids=["near_liked", "far_from_liked"],
            limit=10,
        )

        self.assertEqual(meta["alpha"], 6.0)
        self.assertEqual(meta["positives"], 1)
        ids = [s.item_id for s in scored]
        self.assertEqual(set(ids), {"near_liked", "far_from_liked"})

        expected_top = _cosine([1.0, 0.0], [0.9, 0.1]) * 9.0
        self.assertEqual(scored[0].item_id, "near_liked")
        self.assertAlmostEqual(scored[0].score, expected_top, places=9)
        self.assertIn("liked", scored[0].match_reason or "")

    def test_skips_user_already_rated_candidates(self):
        profile = FakeProfile({"liked": 8.0, "owned": 5.0})
        store = FakeStore(
            {
                "liked": [1.0, 0.0],
                "owned": [0.5, 0.5],
                "fresh": [0.7, 0.3],
            }
        )
        engine = ContentSimilarityEngine(profile=profile, store=store)
        scored, _ = engine.score_for_user(
            user_id="u",
            candidate_ids=["owned", "fresh"],
            limit=10,
        )
        ids = [s.item_id for s in scored]
        self.assertNotIn("owned", ids)
        self.assertEqual(ids, ["fresh"])

    def test_returns_empty_when_no_positive_ratings(self):
        # All ratings are equal to the mean -> nothing strictly above α.
        profile = FakeProfile({"a": 5.0, "b": 5.0})
        store = FakeStore({"a": [1.0], "b": [1.0]})
        engine = ContentSimilarityEngine(profile=profile, store=store)
        scored, meta = engine.score_for_user(
            user_id="u", candidate_ids=["a", "b"], limit=10
        )
        self.assertEqual(scored, [])
        self.assertEqual(meta["reason"], "no_positive_ratings")

    def test_uses_ann_when_no_explicit_candidates(self):
        profile = FakeProfile({"liked": 9.0, "ok": 5.0})  # mean=7, I_u^+ = {liked}
        store = FakeStore(
            {
                "liked": [1.0, 0.0],
                "ok": [0.0, 1.0],
                "candidate_a": [0.95, 0.05],
                "candidate_b": [0.5, 0.5],
            }
        )
        engine = ContentSimilarityEngine(profile=profile, store=store, per_seed_neighbors=5)
        scored, meta = engine.score_for_user(user_id="u", candidate_ids=None, limit=10)

        ids = [s.item_id for s in scored]
        self.assertNotIn("liked", ids)
        self.assertNotIn("ok", ids)
        self.assertEqual(ids[0], "candidate_a")
        self.assertGreaterEqual(meta["candidates"], 1)

    def test_top_score_picks_strongest_seed_per_candidate(self):
        # j1=6.0, j2=9.0 -> mean=7.5 -> I_u^+ = {j2}.
        profile = FakeProfile({"j1": 6.0, "j2": 9.0})
        store = FakeStore(
            {
                "j1": [1.0, 0.0],
                "j2": [0.0, 1.0],
                "i": [0.6, 0.8],
            }
        )
        engine = ContentSimilarityEngine(profile=profile, store=store)
        scored, _ = engine.score_for_user(
            user_id="u", candidate_ids=["i"], limit=10
        )
        self.assertEqual(len(scored), 1)
        expected = _cosine([0.6, 0.8], [0.0, 1.0]) * 9.0
        self.assertAlmostEqual(scored[0].score, expected, places=9)


if __name__ == "__main__":
    unittest.main()
