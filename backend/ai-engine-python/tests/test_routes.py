from __future__ import annotations

import unittest
from unittest.mock import patch

from fastapi import HTTPException

from app.api.routes import assistant, embeddings, recommendations
from app.services.content_similarity import CBScoredItem


class FakeEngine:
    def __init__(self, scored=None, meta=None, exc: Exception | None = None) -> None:
        self.scored = scored or []
        self.meta = meta or {}
        self.exc = exc
        self.calls: list[dict[str, object]] = []

    def score_for_user(self, **kwargs):
        self.calls.append(kwargs)
        if self.exc is not None:
            raise self.exc
        return self.scored, self.meta


class FakeProvider:
    def __init__(self, vector=None, exc: Exception | None = None) -> None:
        self.vector = vector or [0.1, 0.2]
        self.exc = exc
        self.texts: list[str] = []

    def embed_text(self, text: str) -> list[float]:
        self.texts.append(text)
        if self.exc is not None:
            raise self.exc
        return self.vector


class FakeStore:
    def __init__(self, exc: Exception | None = None) -> None:
        self.exc = exc
        self.upserts: list[dict[str, object]] = []

    def upsert(self, **kwargs) -> None:
        self.upserts.append(kwargs)
        if self.exc is not None:
            raise self.exc


class FakeAssistantService:
    def __init__(self) -> None:
        self.calls: list[dict[str, object]] = []

    def chat(self, message: str, history=None):
        self.calls.append({"message": message, "history": history})
        return {"text": "answer", "recommendation_ids": ["i-1"]}


class RecommendationRouteTests(unittest.TestCase):
    def test_cb_recommendations_maps_scored_items(self) -> None:
        engine = FakeEngine(
            scored=[CBScoredItem(item_id="i-1", score=0.9, match_reason="similar")],
            meta={"alpha": 7.0},
        )
        payload = recommendations.RecommendationRequest(
            user_id="u-1", candidate_ids=["i-1"], limit=5
        )

        response = recommendations.cb_recommendations(payload, engine=engine)

        self.assertEqual(response.items[0].item_id, "i-1")
        self.assertEqual(response.items[0].source, "cb")
        self.assertEqual(response.items[0].match_reason, "similar")
        self.assertEqual(response.metadata, {"alpha": 7.0})
        self.assertEqual(
            engine.calls[0],
            {"user_id": "u-1", "candidate_ids": ["i-1"], "limit": 5},
        )

    def test_cb_recommendations_rejects_missing_user(self) -> None:
        payload = recommendations.RecommendationRequest(user_id="")

        with self.assertRaises(HTTPException) as caught:
            recommendations.cb_recommendations(payload, engine=FakeEngine())

        self.assertEqual(caught.exception.status_code, 400)

    def test_cb_recommendations_adds_reason_for_empty_result(self) -> None:
        payload = recommendations.RecommendationRequest(user_id="u-1")
        engine = FakeEngine(meta={"reason": "no_ratings"})

        response = recommendations.cb_recommendations(payload, engine=engine)

        self.assertEqual(response.items, [])
        self.assertEqual(response.reasoning, "no CB recommendations: no_ratings")

    def test_cb_recommendations_wraps_engine_failure(self) -> None:
        payload = recommendations.RecommendationRequest(user_id="u-1")

        with patch.object(recommendations.logger, "exception"), self.assertRaises(HTTPException) as caught:
            recommendations.cb_recommendations(payload, engine=FakeEngine(exc=RuntimeError("boom")))

        self.assertEqual(caught.exception.status_code, 500)


class EmbeddingRouteTests(unittest.TestCase):
    def test_generate_embedding_embeds_and_upserts_vector(self) -> None:
        provider = FakeProvider(vector=[1.0, 0.0])
        store = FakeStore()
        payload = embeddings.EmbeddingRequest(item_id="i-1", text="hello")

        response = embeddings.generate_embedding(payload, provider=provider, store=store)

        self.assertEqual(response.item_id, "i-1")
        self.assertEqual(response.vector, [1.0, 0.0])
        self.assertEqual(provider.texts, ["hello"])
        self.assertEqual(store.upserts, [{"item_id": "i-1", "vector": [1.0, 0.0]}])

    def test_generate_embedding_rejects_missing_fields(self) -> None:
        payload = embeddings.EmbeddingRequest(item_id="", text="")

        with self.assertRaises(HTTPException) as caught:
            embeddings.generate_embedding(payload, provider=FakeProvider(), store=FakeStore())

        self.assertEqual(caught.exception.status_code, 400)

    def test_generate_embedding_wraps_provider_failure(self) -> None:
        payload = embeddings.EmbeddingRequest(item_id="i-1", text="hello")

        with patch.object(embeddings.logger, "exception"), self.assertRaises(HTTPException) as caught:
            embeddings.generate_embedding(
                payload, provider=FakeProvider(exc=RuntimeError("down")), store=FakeStore()
            )

        self.assertEqual(caught.exception.status_code, 500)
        self.assertEqual(caught.exception.detail, "embedding failure")

    def test_generate_embedding_wraps_store_failure(self) -> None:
        payload = embeddings.EmbeddingRequest(item_id="i-1", text="hello")

        with patch.object(embeddings.logger, "exception"), self.assertRaises(HTTPException) as caught:
            embeddings.generate_embedding(
                payload, provider=FakeProvider(), store=FakeStore(exc=RuntimeError("down"))
            )

        self.assertEqual(caught.exception.status_code, 500)
        self.assertEqual(caught.exception.detail, "vector store failure")


class AssistantRouteTests(unittest.TestCase):
    def test_assistant_chat_uses_service(self) -> None:
        service = FakeAssistantService()
        payload = assistant.ChatRequest(
            message="hello",
            history=[assistant.HistoryEntry(role="user", content="previous")],
        )

        with patch.object(assistant, "_get_service", return_value=service):
            response = assistant.assistant_chat(payload)

        self.assertEqual(response.text, "answer")
        self.assertEqual(response.recommendation_ids, ["i-1"])
        self.assertEqual(
            service.calls,
            [{"message": "hello", "history": [{"role": "user", "content": "previous"}]}],
        )


if __name__ == "__main__":
    unittest.main()
