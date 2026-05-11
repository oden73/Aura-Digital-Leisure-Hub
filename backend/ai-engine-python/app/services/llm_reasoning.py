from __future__ import annotations

import json
import logging
import re

from app.core.config import get_settings

logger = logging.getLogger(__name__)

_SYSTEM_PROMPT = (
    "You are Aura, an expert cross-media curator. "
    "You help users find hidden connections between games, books, and movies "
    "based on themes, tonality, and narrative DNA. "
    "When the user describes something they love, explain the connections and "
    "suggest titles they might enjoy. "
    "You MUST respond with valid JSON and nothing else — no markdown, no backticks. "
    'Use exactly this shape: {"text": "<conversational response>", "recommendation_ids": []}. '
    "The recommendation_ids array must remain empty."
)


def _extract_json(raw: str) -> dict:
    """Extract a JSON object from a string that may contain extra text or markdown."""
    raw = raw.strip()
    # Strip markdown code fences if present
    raw = re.sub(r"^```(?:json)?\s*", "", raw)
    raw = re.sub(r"\s*```$", "", raw)
    raw = raw.strip()
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        # Find the first {...} block
        match = re.search(r"\{.*\}", raw, re.DOTALL)
        if match:
            try:
                return json.loads(match.group())
            except json.JSONDecodeError:
                pass
    return {}


class LLMReasoningService:
    def __init__(self) -> None:
        from openai import OpenAI

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
                temperature=0.7,
            )
            raw = completion.choices[0].message.content or "{}"
            result = _extract_json(raw)
            text = str(result.get("text", raw if not result else ""))
            return {
                "text": text,
                "recommendation_ids": list(result.get("recommendation_ids", [])),
            }
        except Exception:
            logger.exception("OpenRouter chat failed")
            return {
                "text": "I'm having trouble connecting right now. Please try again in a moment.",
                "recommendation_ids": [],
            }
