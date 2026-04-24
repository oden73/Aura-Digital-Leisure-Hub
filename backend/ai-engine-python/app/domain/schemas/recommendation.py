from typing import Any

from pydantic import BaseModel

from app.domain.schemas.item import MediaType


class RecommendationRequest(BaseModel):
    user_id: str
    candidate_ids: list[str] = []
    limit: int = 20
    media_types: list[MediaType] = []
    context: dict[str, Any] = {}


class ScoredItem(BaseModel):
    item_id: str
    score: float
    source: str = "cb"
    match_reason: str | None = None


class RecommendationResponse(BaseModel):
    items: list[ScoredItem]
    reasoning: str | None = None
    metadata: dict[str, Any] = {}
