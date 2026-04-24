from typing import Any

from pydantic import BaseModel


class RecommendationRequest(BaseModel):
    user_id: str
    candidate_ids: list[str] = []
    limit: int = 20
    context: dict[str, Any] = {}


class ScoredItem(BaseModel):
    item_id: str
    score: float
    source: str = "cb"


class RecommendationResponse(BaseModel):
    items: list[ScoredItem]
    reasoning: str | None = None
