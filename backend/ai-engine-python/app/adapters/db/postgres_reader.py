"""Read-side Postgres adapter for the AI engine.

Only fetches what the CB pipeline needs: the user's explicit ratings (Rui)
and the per-user mean used as the adaptive α threshold.
"""

from __future__ import annotations


class PostgresProfileReader:
    def __init__(self, dsn: str) -> None:
        self._dsn = dsn
        self._pool = None

    def _ensure_pool(self):
        if self._pool is not None:
            return self._pool
        # psycopg_pool is imported lazily so unit tests that swap this reader
        # do not need the dependency installed.
        from psycopg_pool import ConnectionPool

        self._pool = ConnectionPool(
            conninfo=self._dsn,
            min_size=1,
            max_size=4,
            kwargs={"autocommit": True},
        )
        return self._pool

    def get_user_ratings(self, user_id: str) -> dict[str, float]:
        """Return {item_id: rating} for every rated interaction of the user."""

        pool = self._ensure_pool()
        with pool.connection() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT item_id, rating
                FROM user_interactions
                WHERE user_id = %s AND rating IS NOT NULL
                """,
                (user_id,),
            )
            rows = cur.fetchall()
        return {item_id: float(rating) for item_id, rating in rows}

    def get_user_mean(self, user_id: str) -> float | None:
        pool = self._ensure_pool()
        with pool.connection() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT AVG(rating)::float8
                FROM user_interactions
                WHERE user_id = %s AND rating IS NOT NULL
                """,
                (user_id,),
            )
            row = cur.fetchone()
        if row is None or row[0] is None:
            return None
        return float(row[0])

    def get_item_text(self, item_id: str) -> str | None:
        """Return concatenated description for embedding generation."""

        pool = self._ensure_pool()
        with pool.connection() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT title, COALESCE(original_title, ''), COALESCE(description, '')
                FROM base_items
                WHERE item_id = %s
                """,
                (item_id,),
            )
            row = cur.fetchone()
        if row is None:
            return None
        title, original_title, description = row
        parts = [str(title), str(original_title), str(description)]
        return " ".join(p for p in parts if p)
