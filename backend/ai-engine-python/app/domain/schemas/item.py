"""Pydantic schemas for multimedia items.

Aligned with docs/predone/data_model.md and docs/predone/diagrams/db_scheme.puml.
"""

from __future__ import annotations

from datetime import date
from enum import Enum

from pydantic import BaseModel, Field


class MediaType(str, Enum):
    BOOK = "book"
    CINEMA = "cinema"
    GAME = "game"


class BaseItemCriteria(BaseModel):
    """Общие критерии, применимые ко всем медиа."""

    genre: str | None = None
    setting: str | None = None
    themes: str | None = None
    tonality: str | None = None
    target_audience: str | None = None


class BookDetails(BaseModel):
    author: str | None = None
    publisher: str | None = None
    literary_form: str | None = None
    volume_format: str | None = None
    narrative_type: str | None = None
    artistic_style: str | None = None
    page_count: int | None = None


class CinemaDetails(BaseModel):
    director: str | None = None
    cast: str | None = None
    format: str | None = None
    production_method: str | None = None
    visual_style: str | None = None
    plot_structure: str | None = None
    duration_mins: int | None = None


class GameDetails(BaseModel):
    developer: str | None = None
    gameplay_genre: str | None = None
    platforms: str | None = None
    player_count: str | None = None
    perspective: str | None = None
    plot_genre: str | None = None
    world_structure: str | None = None
    monetization: str | None = None


class ItemMetadata(BaseModel):
    item_id: str
    title: str
    original_title: str | None = None
    description: str | None = None
    release_date: date | None = None
    cover_image_url: str | None = None
    average_rating: float | None = None
    media_type: MediaType
    criteria: BaseItemCriteria = Field(default_factory=BaseItemCriteria)
    book_details: BookDetails | None = None
    cinema_details: CinemaDetails | None = None
    game_details: GameDetails | None = None
