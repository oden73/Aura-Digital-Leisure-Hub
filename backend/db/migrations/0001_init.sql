-- Initial schema for Aura backend.
-- Aligned with docs/predone/diagrams/db_scheme.puml and docs/predone/data_model.md.

BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enumerations -------------------------------------------------------------

CREATE TYPE media_type AS ENUM ('book', 'cinema', 'game');

CREATE TYPE external_service AS ENUM (
    'steam',
    'epic_games',
    'kinopoisk',
    'netflix',
    'goodreads',
    'yandex_books'
);

CREATE TYPE interaction_status AS ENUM (
    'planned',
    'in_progress',
    'completed',
    'dropped'
);

-- User management ----------------------------------------------------------

CREATE TABLE users (
    user_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username       VARCHAR(50)  NOT NULL UNIQUE,
    email          VARCHAR(255) NOT NULL UNIQUE,
    password_hash  TEXT         NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE external_accounts (
    account_id            BIGSERIAL PRIMARY KEY,
    user_id               UUID             NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    service_name          external_service NOT NULL,
    external_user_id      VARCHAR(255)     NOT NULL,
    external_profile_url  TEXT,
    last_synced_at        TIMESTAMPTZ,
    UNIQUE (service_name, external_user_id)
);

CREATE INDEX idx_external_accounts_user ON external_accounts(user_id);

-- Core multimedia catalog --------------------------------------------------
-- Common base criteria (genre, setting, themes, tonality, target_audience)
-- mirror the "Общие критерии" section of docs/predone/data_model.md.

CREATE TABLE base_items (
    item_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title            VARCHAR(255) NOT NULL,
    original_title   VARCHAR(255),
    description      TEXT,
    release_date     DATE,
    cover_image_url  TEXT,
    average_rating   NUMERIC(3, 2),
    media_type       media_type   NOT NULL,
    -- Base criteria shared by books, cinema and games.
    genre            VARCHAR(100),
    setting          VARCHAR(100),
    themes           TEXT,
    tonality         VARCHAR(100),
    target_audience  VARCHAR(50),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_base_items_media_type ON base_items(media_type);
CREATE INDEX idx_base_items_genre      ON base_items(genre);

-- Media-specific details ---------------------------------------------------

CREATE TABLE book_details (
    item_id         UUID PRIMARY KEY REFERENCES base_items(item_id) ON DELETE CASCADE,
    author          VARCHAR(255),
    publisher       VARCHAR(255),
    literary_form   VARCHAR(100),
    volume_format   VARCHAR(100),
    narrative_type  VARCHAR(100),
    artistic_style  VARCHAR(100),
    page_count      INT
);

-- `cast` is a reserved word in SQL; stored as `cast_list`.
CREATE TABLE cinema_details (
    item_id            UUID PRIMARY KEY REFERENCES base_items(item_id) ON DELETE CASCADE,
    director           VARCHAR(255),
    cast_list          TEXT,
    format             VARCHAR(100),
    production_method  VARCHAR(100),
    visual_style       VARCHAR(100),
    plot_structure     VARCHAR(100),
    duration_mins      INT
);

CREATE TABLE game_details (
    item_id         UUID PRIMARY KEY REFERENCES base_items(item_id) ON DELETE CASCADE,
    developer       VARCHAR(255),
    gameplay_genre  VARCHAR(100),
    platforms       VARCHAR(255),
    player_count    VARCHAR(100),
    perspective     VARCHAR(100),
    plot_genre      VARCHAR(100),
    world_structure VARCHAR(100),
    monetization    VARCHAR(100)
);

-- Interactions (Rui matrix) -----------------------------------------------

CREATE TABLE user_interactions (
    interaction_id  BIGSERIAL PRIMARY KEY,
    user_id         UUID               NOT NULL REFERENCES users(user_id)      ON DELETE CASCADE,
    item_id         UUID               NOT NULL REFERENCES base_items(item_id) ON DELETE CASCADE,
    status          interaction_status NOT NULL DEFAULT 'planned',
    rating          INT                CHECK (rating BETWEEN 1 AND 10),
    is_favorite     BOOLEAN            NOT NULL DEFAULT FALSE,
    review_text     TEXT,
    updated_at      TIMESTAMPTZ        NOT NULL DEFAULT now(),
    UNIQUE (user_id, item_id)
);

CREATE INDEX idx_interactions_user ON user_interactions(user_id);
CREATE INDEX idx_interactions_item ON user_interactions(item_id);

-- AI search layer ---------------------------------------------------------
-- Actual vectors live in ChromaDB; this table keeps a reference and bookkeeping.
CREATE TABLE vector_store (
    item_id             UUID PRIMARY KEY REFERENCES base_items(item_id) ON DELETE CASCADE,
    embedding_dim       INT         NOT NULL DEFAULT 1536,
    embedding_ref       VARCHAR(255),
    last_vectorized_at  TIMESTAMPTZ
);

COMMIT;
