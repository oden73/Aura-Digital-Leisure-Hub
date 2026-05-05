from __future__ import annotations

from functools import lru_cache

from fastapi import APIRouter
from pydantic import BaseModel

from app.services.llm_reasoning import LLMReasoningService

router = APIRouter(tags=["assistant"])


class HistoryEntry(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    message: str
    history: list[HistoryEntry] = []


class ChatResponse(BaseModel):
    text: str
    recommendation_ids: list[str] = []


@lru_cache(maxsize=1)
def _get_service() -> LLMReasoningService:
    return LLMReasoningService()


@router.post("/assistant/chat", response_model=ChatResponse)
def assistant_chat(payload: ChatRequest) -> ChatResponse:
    result = _get_service().chat(
        message=payload.message,
        history=[e.model_dump() for e in payload.history],
    )
    return ChatResponse(
        text=result["text"],
        recommendation_ids=result["recommendation_ids"],  # type: ignore[arg-type]
    )
