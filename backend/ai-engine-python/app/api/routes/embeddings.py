from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, HTTPException

from app.adapters.ml.embedding_provider import EmbeddingProvider
from app.adapters.vector_store.chroma_adapter import ChromaVectorStoreAdapter
from app.core.container import get_embedding_provider, get_vector_store
from app.domain.schemas.embedding import EmbeddingRequest, EmbeddingResponse

router = APIRouter(tags=["embeddings"])
logger = logging.getLogger(__name__)


@router.post("/embeddings/generate", response_model=EmbeddingResponse)
def generate_embedding(
    payload: EmbeddingRequest,
    provider: EmbeddingProvider = Depends(get_embedding_provider),
    store: ChromaVectorStoreAdapter = Depends(get_vector_store),
) -> EmbeddingResponse:
    if not payload.item_id or not payload.text:
        raise HTTPException(status_code=400, detail="item_id and text are required")

    try:
        vector = provider.embed_text(payload.text)
    except Exception:
        logger.exception("embedding failed for item_id=%s", payload.item_id)
        raise HTTPException(status_code=500, detail="embedding failure") from None

    try:
        store.upsert(item_id=payload.item_id, vector=vector)
    except Exception:
        logger.exception("vector store upsert failed for item_id=%s", payload.item_id)
        raise HTTPException(status_code=500, detail="vector store failure") from None

    return EmbeddingResponse(item_id=payload.item_id, vector=vector)
