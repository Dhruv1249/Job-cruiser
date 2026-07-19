import unittest
from unittest.mock import patch, mock_open, MagicMock
from pathlib import Path
import json
from index import build_index

class TestIndexBuilder(unittest.TestCase):
    """
    Unit tests for the index building script
    """

    @patch("builtins.open", new_callable=mock_open)
    def test_index_building(self, mock_file_open):
        """
        Verify index script compiles company summaries into index.json
        """
        mock_data_dir = MagicMock()
        mock_folder = MagicMock()
        mock_folder.is_dir.return_value = True
        mock_folder.__truediv__.return_value = mock_folder
        mock_folder.exists.return_value = True
        mock_data_dir.iterdir.return_value = [mock_folder]

        company_data = {"company": "airbnb", "job_count": 5}
        
        with patch("json.load", return_value=company_data):
            build_index(mock_data_dir)

        mock_file_open.assert_any_call(
            mock_data_dir / "index.json",
            "w",
            encoding="utf-8"
        )
