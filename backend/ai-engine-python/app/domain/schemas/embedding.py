from pydantic import BaseModel


class EmbeddingRequest(BaseModel):
    item_id: str
    text: str


class EmbeddingResponse(BaseModel):
    item_id: str
    vector: list[float]
