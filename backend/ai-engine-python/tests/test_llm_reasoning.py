from __future__ import annotations

import unittest
from unittest.mock import patch

from app.services.llm_reasoning import LLMReasoningService, _extract_json


class FakeMessage:
    def __init__(self, content: str | None) -> None:
        self.content = content


class FakeChoice:
    def __init__(self, content: str | None) -> None:
        self.message = FakeMessage(content)


class FakeCompletion:
    def __init__(self, content: str | None) -> None:
        self.choices = [FakeChoice(content)]


class FakeCompletions:
    def __init__(self, content: str | None = None, exc: Exception | None = None) -> None:
        self.content = content
        self.exc = exc
        self.calls: list[dict[str, object]] = []

    def create(self, **kwargs):
        self.calls.append(kwargs)
        if self.exc is not None:
            raise self.exc
        return FakeCompletion(self.content)


class FakeChat:
    def __init__(self, completions: FakeCompletions) -> None:
        self.completions = completions


class FakeClient:
    def __init__(self, completions: FakeCompletions) -> None:
        self.chat = FakeChat(completions)


def make_service(completions: FakeCompletions) -> LLMReasoningService:
    service = LLMReasoningService.__new__(LLMReasoningService)
    service._client = FakeClient(completions)
    service._model = "test-model"
    return service


class ExtractJSONTests(unittest.TestCase):
    def test_extract_json_accepts_plain_json(self) -> None:
        self.assertEqual(_extract_json('{"text": "ok", "recommendation_ids": []}')["text"], "ok")

    def test_extract_json_strips_markdown_fence(self) -> None:
        got = _extract_json('```json\n{"text": "ok", "recommendation_ids": []}\n```')

        self.assertEqual(got["text"], "ok")

    def test_extract_json_finds_embedded_object(self) -> None:
        got = _extract_json('prefix {"text": "ok", "recommendation_ids": ["i"]} suffix')

        self.assertEqual(got["recommendation_ids"], ["i"])

    def test_extract_json_returns_empty_for_invalid_input(self) -> None:
        self.assertEqual(_extract_json("not-json"), {})


class LLMReasoningServiceTests(unittest.TestCase):
    def test_chat_sends_history_and_parses_json(self) -> None:
        completions = FakeCompletions('{"text": "Try this", "recommendation_ids": ["i-1"]}')
        service = make_service(completions)

        result = service.chat(
            "hello",
            history=[
                {"role": "user", "content": "previous"},
                {"role": "assistant", "content": "answer"},
                {"role": "system", "content": "ignored"},
                {"role": "user", "content": ""},
            ],
        )

        self.assertEqual(result, {"text": "Try this", "recommendation_ids": ["i-1"]})
        call = completions.calls[0]
        self.assertEqual(call["model"], "test-model")
        self.assertEqual(call["temperature"], 0.7)
        messages = call["messages"]
        self.assertEqual(messages[-1], {"role": "user", "content": "hello"})
        self.assertIn({"role": "user", "content": "previous"}, messages)
        self.assertIn({"role": "assistant", "content": "answer"}, messages)
        self.assertNotIn({"role": "system", "content": "ignored"}, messages)

    def test_chat_falls_back_to_raw_text_when_json_is_missing(self) -> None:
        service = make_service(FakeCompletions("plain response"))

        result = service.chat("hello")

        self.assertEqual(result["text"], "plain response")
        self.assertEqual(result["recommendation_ids"], [])

    def test_chat_returns_safe_error_on_client_failure(self) -> None:
        service = make_service(FakeCompletions(exc=RuntimeError("down")))

        with patch("app.services.llm_reasoning.logger.exception"):
            result = service.chat("hello")

        self.assertEqual(result["recommendation_ids"], [])
        self.assertIn("trouble connecting", result["text"])

    def test_explain_recommendations_is_currently_empty(self) -> None:
        service = make_service(FakeCompletions())

        self.assertEqual(service.explain_recommendations("u-1", ["i-1"]), "")


if __name__ == "__main__":
    unittest.main()
