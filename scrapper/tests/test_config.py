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
    Validates that security-critical configuration parameters have safe defaults.
    """

    def test_ingest_api_key_empty_by_default(self):
        """
        Ensures that INGEST_API_KEY does not fall back to insecure hardcoded default strings.
        """
        environment_copy = dict(os.environ)
        environment_copy.pop("INGEST_API_KEY", None)
        environment_copy.pop("INGEST_KEY", None)

        with patch.dict(os.environ, environment_copy, clear=True):
            with patch.object(Path, "exists", return_value=False):
                import config
                importlib.reload(config)
                self.assertEqual(config.INGEST_API_KEY, "")
