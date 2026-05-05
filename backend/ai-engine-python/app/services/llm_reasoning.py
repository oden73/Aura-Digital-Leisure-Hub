from __future__ import annotations

import json
import logging

from openai import OpenAI

from app.core.config import get_settings

logger = logging.getLogger(__name__)

_SYSTEM_PROMPT = (
    "You are Aura, an expert cross-media curator. "
    "You help users find hidden connections between games, books, and movies "
    "based on themes, tonality, and narrative DNA. "
    "When the user describes something they love, explain the connections and "
    "suggest titles they might enjoy. "
    "Always respond in JSON with the following shape: "
    '{"text": "<conversational response>", "recommendation_ids": []}. '
    "The recommendation_ids array should remain empty — item IDs are resolved externally."
)


class LLMReasoningService:
    def __init__(self) -> None:
        settings = get_settings()
        self._client = OpenAI(
            api_key=settings.openrouter_api_key,
            base_url="https://openrouter.ai/api/v1",
        )
        self._model = settings.openrouter_model

    def explain_recommendations(self, user_id: str, item_ids: list[str]) -> str:
        _ = user_id
        _ = item_ids
        return ""

    def chat(
        self,
        message: str,
        history: list[dict[str, str]] | None = None,
    ) -> dict[str, object]:
        messages: list[dict[str, str]] = [{"role": "system", "content": _SYSTEM_PROMPT}]
        for entry in history or []:
            role = entry.get("role", "user")
            content = entry.get("content", "")
            if role in {"user", "assistant"} and content:
                messages.append({"role": role, "content": content})
        messages.append({"role": "user", "content": message})

        try:
            completion = self._client.chat.completions.create(
                model=self._model,
                messages=messages,  # type: ignore[arg-type]
                response_format={"type": "json_object"},
                temperature=0.7,
            )
            raw = completion.choices[0].message.content or "{}"
            result = json.loads(raw)
            return {
                "text": str(result.get("text", "")),
                "recommendation_ids": list(result.get("recommendation_ids", [])),
            }
        except Exception:
            logger.exception("OpenRouter chat failed")
            return {
                "text": "I'm having trouble connecting right now. Please try again in a moment.",
                "recommendation_ids": [],
            }
