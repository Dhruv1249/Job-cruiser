"""
Unit tests for configuration defaults and environment loading.
"""

import importlib
import os
import unittest
from pathlib import Path
from unittest.mock import patch


class TestConfigDefaults(unittest.TestCase):
    """
    Validates that security-critical configuration parameters enforce presence without fallback.
    """

    def test_missing_ingest_api_key_raises_runtime_error(self):
        """
        Ensures that missing INGEST_API_KEY raises a critical RuntimeError immediately without fallback.
        """
        environment_copy = dict(os.environ)
        environment_copy.pop("INGEST_API_KEY", None)
        environment_copy.pop("INGEST_KEY", None)

        with patch.dict(os.environ, environment_copy, clear=True):
            with patch.object(Path, "exists", return_value=False):
                import config
                with self.assertRaises(RuntimeError):
                    importlib.reload(config)

    def test_configured_environment_variables_loaded_correctly(self):
        """
        Ensures that configured environment variables are loaded accurately.
        """
        test_env = {
            "BACKEND_API_URL": "http://localhost:8080/api",
            "INGEST_API_KEY": "test-ingest-key",
            "GEMINI_API_KEY": "test-gemini-key",
        }
        with patch.dict(os.environ, test_env, clear=True):
            with patch.object(Path, "exists", return_value=False):
                import config
                importlib.reload(config)
                self.assertEqual(config.BACKEND_API_URL, "http://localhost:8080/api")
                self.assertEqual(config.INGEST_API_KEY, "test-ingest-key")
                self.assertEqual(config.GEMINI_API_KEY, "test-gemini-key")
