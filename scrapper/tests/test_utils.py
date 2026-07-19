import unittest
from unittest.mock import mock_open, patch
from job_sources.utils import load_yaml_config

class TestUtils(unittest.TestCase):
    """
    Unit tests for utility functions
    """

    def test_load_yaml_config_success(self):
        """
        Verify load_yaml_config parses yaml content correctly
        """
        yaml_content = """
        # Test configuration
        greenhouse:
          - airbnb
          - stripe
        
        lever:
          - atlassian
          - netflix
        
        ashby:
          - notion
          - ramp
        """
        with patch("builtins.open", mock_open(read_data=yaml_content)):
            config = load_yaml_config("fake_path.yaml")

        self.assertIn("greenhouse", config)
        self.assertIn("lever", config)
        self.assertIn("ashby", config)
        self.assertEqual(config["greenhouse"], ["airbnb", "stripe"])
        self.assertEqual(config["lever"], ["atlassian", "netflix"])
        self.assertEqual(config["ashby"], ["notion", "ramp"])

    def test_load_yaml_config_empty(self):
        """
        Verify load_yaml_config handles empty files or invalid syntax gracefully
        """
        yaml_content = ""
        with patch("builtins.open", mock_open(read_data=yaml_content)):
            config = load_yaml_config("fake_path.yaml")
        self.assertEqual(config, {})
