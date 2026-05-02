from fastapi import FastAPI
from fastapi.responses import JSONResponse

from app.api.routes import embeddings, recommendations


def create_app() -> FastAPI:
    app = FastAPI(title="Aura AI Engine", version="0.1.0")

    @app.get("/health")
    async def health() -> JSONResponse:
        return JSONResponse({"status": "ok"})

    app.include_router(recommendations.router, prefix="/v1")
    app.include_router(embeddings.router, prefix="/v1")
    return app


app = create_app()
