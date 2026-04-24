from fastapi import APIRouter

from app.domain.schemas.recommendation import RecommendationRequest, RecommendationResponse

router = APIRouter(tags=["recommendations"])


@router.post("/recommendations/cb", response_model=RecommendationResponse)
def cb_recommendations(payload: RecommendationRequest) -> RecommendationResponse:
    return RecommendationResponse(items=[], reasoning=None)
