from __future__ import annotations

import os
import unittest
from unittest.mock import patch

from app.core import config
from app.core.config import Settings, get_settings
from app.main import create_app


class SettingsTests(unittest.TestCase):
    def tearDown(self) -> None:
        config._cached = None

    def test_from_env_uses_defaults(self) -> None:
        with patch.dict(os.environ, {}, clear=True):
            settings = Settings.from_env()

        self.assertEqual(settings.host, "0.0.0.0")
        self.assertEqual(settings.port, 8090)
        self.assertEqual(settings.chroma_collection, "aura-items")
        self.assertEqual(settings.cb_default_limit, 20)
        self.assertEqual(settings.cb_per_seed_neighbors, 50)
        self.assertEqual(settings.openrouter_api_key, "")

    def test_from_env_applies_overrides(self) -> None:
        env = {
            "AI_ENGINE_HOST": "127.0.0.1",
            "AI_ENGINE_PORT": "9000",
            "DATABASE_URL": "postgres://test",
            "CHROMA_URL": "https://chroma.example:9443",
            "CHROMA_COLLECTION": "items-test",
            "EMBEDDING_MODEL": "model-test",
            "CB_DEFAULT_LIMIT": "7",
            "CB_PER_SEED_NEIGHBORS": "8",
            "OPENROUTER_API_KEY": "key",
            "OPENROUTER_MODEL": "model",
        }
        with patch.dict(os.environ, env, clear=True):
            settings = Settings.from_env()

        self.assertEqual(settings.host, "127.0.0.1")
        self.assertEqual(settings.port, 9000)
        self.assertEqual(settings.database_url, "postgres://test")
        self.assertEqual(settings.chroma_url, "https://chroma.example:9443")
        self.assertEqual(settings.chroma_collection, "items-test")
        self.assertEqual(settings.embedding_model, "model-test")
        self.assertEqual(settings.cb_default_limit, 7)
        self.assertEqual(settings.cb_per_seed_neighbors, 8)
        self.assertEqual(settings.openrouter_api_key, "key")
        self.assertEqual(settings.openrouter_model, "model")

    def test_get_settings_caches_instance(self) -> None:
        config._cached = None
        with patch.object(Settings, "from_env", wraps=Settings.from_env) as mocked:
            first = get_settings()
            second = get_settings()

        self.assertIs(first, second)
        self.assertEqual(mocked.call_count, 1)


class AppFactoryTests(unittest.TestCase):
    def test_create_app_registers_core_routes(self) -> None:
        app = create_app()
        paths = {route.path for route in app.routes}

        self.assertIn("/health", paths)
        self.assertIn("/v1/recommendations/cb", paths)
        self.assertIn("/v1/embeddings/generate", paths)
        self.assertIn("/v1/assistant/chat", paths)


if __name__ == "__main__":
    unittest.main()
