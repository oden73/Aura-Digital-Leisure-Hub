from __future__ import annotations

import unittest

from app.adapters.ml.embedding_provider import EmbeddingProvider


class FakeVector:
    def __init__(self, values: list[float]) -> None:
        self._values = values

    def tolist(self) -> list[float]:
        return self._values


class FakeModel:
    def __init__(self) -> None:
        self.calls: list[dict[str, object]] = []

    def encode(self, texts, **kwargs):
        self.calls.append({"texts": list(texts), **kwargs})
        return [FakeVector([float(len(text)), 1.0]) for text in texts]


class EmbeddingProviderTests(unittest.TestCase):
    def test_model_name_is_exposed_without_loading_model(self) -> None:
        provider = EmbeddingProvider("test-model")

        self.assertEqual(provider.model_name, "test-model")
        self.assertIsNone(provider._model)

    def test_embed_batch_returns_empty_without_loading_model(self) -> None:
        provider = EmbeddingProvider("test-model")

        self.assertEqual(provider.embed_batch([]), [])
        self.assertIsNone(provider._model)

    def test_embed_batch_uses_loaded_model(self) -> None:
        provider = EmbeddingProvider("test-model")
        fake_model = FakeModel()
        provider._model = fake_model

        vectors = provider.embed_batch(["abc", "d"])

        self.assertEqual(vectors, [[3.0, 1.0], [1.0, 1.0]])
        self.assertEqual(fake_model.calls[0]["texts"], ["abc", "d"])
        self.assertTrue(fake_model.calls[0]["convert_to_numpy"])
        self.assertTrue(fake_model.calls[0]["normalize_embeddings"])
        self.assertFalse(fake_model.calls[0]["show_progress_bar"])

    def test_embed_text_returns_first_batch_vector(self) -> None:
        provider = EmbeddingProvider("test-model")
        provider._model = FakeModel()

        self.assertEqual(provider.embed_text("abcd"), [4.0, 1.0])

    def test_cosine_handles_valid_and_invalid_vectors(self) -> None:
        self.assertAlmostEqual(EmbeddingProvider.cosine([1, 0], [1, 0]), 1.0)
        self.assertAlmostEqual(EmbeddingProvider.cosine([1, 0], [0, 1]), 0.0)
        self.assertEqual(EmbeddingProvider.cosine([], [1]), 0.0)
        self.assertEqual(EmbeddingProvider.cosine([1], [1, 2]), 0.0)
        self.assertEqual(EmbeddingProvider.cosine([0, 0], [1, 2]), 0.0)


if __name__ == "__main__":
    unittest.main()
