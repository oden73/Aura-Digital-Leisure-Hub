from __future__ import annotations

import sys
import types
import unittest
from unittest.mock import patch

from app.adapters.db.postgres_reader import PostgresProfileReader
from app.adapters.vector_store.chroma_adapter import ChromaVectorStoreAdapter


class FakeCursor:
    def __init__(self, rows=None, row=None) -> None:
        self.rows = rows or []
        self.row = row
        self.calls: list[tuple[str, tuple[str, ...]]] = []

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None

    def execute(self, sql: str, params: tuple[str, ...]) -> None:
        self.calls.append((sql, params))

    def fetchall(self):
        return self.rows

    def fetchone(self):
        return self.row


class FakeConnection:
    def __init__(self, cursor: FakeCursor) -> None:
        self._cursor = cursor

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None

    def cursor(self) -> FakeCursor:
        return self._cursor


class FakePool:
    def __init__(self, cursor: FakeCursor) -> None:
        self._cursor = cursor

    def connection(self) -> FakeConnection:
        return FakeConnection(self._cursor)


class FakeCollection:
    def __init__(self) -> None:
        self.upserts: list[dict[str, object]] = []
        self.get_result = {"ids": ["i-1"], "embeddings": [[1.0, 0.0]]}
        self.query_result = {
            "ids": [["i-1", "i-2", "i-3"]],
            "distances": [[0.1, 0.2, 0.3]],
        }

    def upsert(self, **kwargs) -> None:
        self.upserts.append(kwargs)

    def get(self, **kwargs):
        return self.get_result

    def query(self, **kwargs):
        self.query_kwargs = kwargs
        return self.query_result


class FakeClient:
    def __init__(self, collection: FakeCollection) -> None:
        self.collection = collection
        self.calls: list[dict[str, object]] = []

    def get_or_create_collection(self, **kwargs) -> FakeCollection:
        self.calls.append(kwargs)
        return self.collection


class PostgresProfileReaderTests(unittest.TestCase):
    def test_get_user_ratings_reads_rating_rows(self) -> None:
        cursor = FakeCursor(rows=[("i-1", 8), ("i-2", 6.5)])
        reader = PostgresProfileReader("postgres://test")
        reader._pool = FakePool(cursor)

        self.assertEqual(reader.get_user_ratings("u-1"), {"i-1": 8.0, "i-2": 6.5})
        self.assertEqual(cursor.calls[0][1], ("u-1",))

    def test_get_user_mean_returns_float_or_none(self) -> None:
        cursor = FakeCursor(row=(7.5,))
        reader = PostgresProfileReader("postgres://test")
        reader._pool = FakePool(cursor)

        self.assertEqual(reader.get_user_mean("u-1"), 7.5)

        cursor = FakeCursor(row=(None,))
        reader._pool = FakePool(cursor)
        self.assertIsNone(reader.get_user_mean("u-1"))

        cursor = FakeCursor(row=None)
        reader._pool = FakePool(cursor)
        self.assertIsNone(reader.get_user_mean("u-1"))

    def test_get_item_text_concatenates_non_empty_parts(self) -> None:
        cursor = FakeCursor(row=("Title", "", "Description"))
        reader = PostgresProfileReader("postgres://test")
        reader._pool = FakePool(cursor)

        self.assertEqual(reader.get_item_text("i-1"), "Title Description")

        cursor = FakeCursor(row=None)
        reader._pool = FakePool(cursor)
        self.assertIsNone(reader.get_item_text("i-1"))


class ChromaVectorStoreAdapterTests(unittest.TestCase):
    def make_adapter(self) -> tuple[ChromaVectorStoreAdapter, FakeCollection, FakeClient]:
        collection = FakeCollection()
        client = FakeClient(collection)

        chromadb_module = types.SimpleNamespace(
            HttpClient=lambda **kwargs: client,
        )
        config_module = types.SimpleNamespace(
            Settings=lambda **kwargs: {"settings": kwargs},
        )
        with patch.dict(
            sys.modules,
            {
                "chromadb": chromadb_module,
                "chromadb.config": config_module,
            },
        ):
            adapter = ChromaVectorStoreAdapter(
                url="https://chroma.example:9443", collection_name="items"
            )
        return adapter, collection, client

    def test_init_creates_cosine_collection(self) -> None:
        _adapter, _collection, client = self.make_adapter()

        self.assertEqual(
            client.calls,
            [{"name": "items", "metadata": {"hnsw:space": "cosine"}}],
        )

    def test_upsert_uses_metadata_or_sentinel(self) -> None:
        adapter, collection, _client = self.make_adapter()

        adapter.upsert("i-1", [1.0], metadata=None)
        adapter.upsert("i-2", [2.0], metadata={"kind": "game"})

        self.assertEqual(collection.upserts[0]["metadatas"], [{"item_id": "i-1"}])
        self.assertEqual(collection.upserts[1]["metadatas"], [{"kind": "game"}])

    def test_get_vectors_handles_empty_missing_and_present_embeddings(self) -> None:
        adapter, collection, _client = self.make_adapter()

        self.assertEqual(adapter.get_vectors([]), {})

        collection.get_result = {
            "ids": ["i-1", "i-2"],
            "embeddings": [[1.0, 0.0], None],
        }
        self.assertEqual(adapter.get_vectors(["i-1", "i-2"]), {"i-1": [1.0, 0.0]})

    def test_query_converts_distances_and_filters_excluded_ids(self) -> None:
        adapter, collection, _client = self.make_adapter()

        got = adapter.query([1.0, 0.0], k=2, exclude_ids=["i-1"])

        self.assertEqual(got, [("i-2", 0.8), ("i-3", 0.7)])
        self.assertEqual(collection.query_kwargs["n_results"], 3)


if __name__ == "__main__":
    unittest.main()
