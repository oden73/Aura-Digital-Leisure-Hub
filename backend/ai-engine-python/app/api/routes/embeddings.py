from fastapi import APIRouter

from app.domain.schemas.embedding import EmbeddingRequest, EmbeddingResponse

router = APIRouter(tags=["embeddings"])


@router.post("/embeddings/generate", response_model=EmbeddingResponse)
def generate_embedding(payload: EmbeddingRequest) -> EmbeddingResponse:
    return EmbeddingResponse(item_id=payload.item_id, vector=[])
