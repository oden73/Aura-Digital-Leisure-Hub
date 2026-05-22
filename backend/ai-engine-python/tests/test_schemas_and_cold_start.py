from __future__ import annotations

import unittest
from datetime import date

from app.domain.schemas.embedding import EmbeddingRequest, EmbeddingResponse
from app.domain.schemas.item import BaseItemCriteria, ItemMetadata, MediaType
from app.domain.schemas.recommendation import (
    RecommendationRequest,
    RecommendationResponse,
    ScoredItem,
)
from app.services.cold_start import ColdStartHandler


class SchemaTests(unittest.TestCase):
    def test_embedding_schemas_round_trip(self) -> None:
        req = EmbeddingRequest(item_id="i-1", text="hello")
        resp = EmbeddingResponse(item_id=req.item_id, vector=[0.1, 0.2])

        self.assertEqual(req.model_dump(), {"item_id": "i-1", "text": "hello"})
        self.assertEqual(resp.vector, [0.1, 0.2])

    def test_recommendation_schemas_defaults_and_values(self) -> None:
        req = RecommendationRequest(user_id="u-1")

        self.assertEqual(req.candidate_ids, [])
        self.assertEqual(req.limit, 20)
        self.assertEqual(req.media_types, [])
        self.assertEqual(req.context, {})

        item = ScoredItem(item_id="i-1", score=0.5, match_reason="similar")
        resp = RecommendationResponse(items=[item], metadata={"alpha": 7})

        self.assertEqual(resp.items[0].source, "cb")
        self.assertEqual(resp.metadata["alpha"], 7)

    def test_item_metadata_accepts_media_specific_fields(self) -> None:
        item = ItemMetadata(
            item_id="i-1",
            title="Title",
            release_date=date(2020, 1, 2),
            average_rating=8.5,
            media_type=MediaType.GAME,
            criteria=BaseItemCriteria(genre="rpg", themes="friendship"),
        )

        self.assertEqual(item.media_type, MediaType.GAME)
        self.assertEqual(item.criteria.genre, "rpg")
        self.assertEqual(item.model_dump()["release_date"], date(2020, 1, 2))


class ColdStartTests(unittest.TestCase):
    def test_generate_initial_recommendations_returns_empty_list(self) -> None:
        self.assertEqual(ColdStartHandler().generate_initial_recommendations("u-1"), [])


if __name__ == "__main__":
    unittest.main()
