from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, HTTPException

from app.core.container import get_content_engine
from app.domain.schemas.recommendation import (
    RecommendationRequest,
    RecommendationResponse,
    ScoredItem,
)
from app.services.content_similarity import ContentSimilarityEngine

router = APIRouter(tags=["recommendations"])
logger = logging.getLogger(__name__)


@router.post("/recommendations/cb", response_model=RecommendationResponse)
def cb_recommendations(
    payload: RecommendationRequest,
    engine: ContentSimilarityEngine = Depends(get_content_engine),
) -> RecommendationResponse:
    if not payload.user_id:
        raise HTTPException(status_code=400, detail="user_id is required")

    try:
        scored, meta = engine.score_for_user(
            user_id=payload.user_id,
            candidate_ids=payload.candidate_ids or None,
            limit=payload.limit or 20,
        )
    except Exception:  # noqa: BLE001 — surface as 500 with logged context
        logger.exception("CB scoring failed for user_id=%s", payload.user_id)
        raise HTTPException(status_code=500, detail="cb engine failure") from None

    items = [
        ScoredItem(
            item_id=s.item_id,
            score=s.score,
            source="cb",
            match_reason=s.match_reason,
        )
        for s in scored
    ]
    reasoning = None
    if not items and meta.get("reason"):
        reasoning = f"no CB recommendations: {meta['reason']}"
    return RecommendationResponse(items=items, reasoning=reasoning, metadata=meta)
