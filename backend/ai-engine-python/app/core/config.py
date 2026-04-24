from pydantic import BaseModel


class Settings(BaseModel):
    app_name: str = "Aura AI Engine"
    host: str = "0.0.0.0"
    port: int = 8090
    chroma_collection: str = "aura-items"


def get_settings() -> Settings:
    return Settings()
