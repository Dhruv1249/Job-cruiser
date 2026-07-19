import unittest
from unittest.mock import Mock, patch, mock_open
from scrapper import process_company, main

class TestScrapperOrchestrator(unittest.TestCase):
    """
    Unit tests for the main scraper orchestrator functions
    """

    @patch("scrapper.GreenhouseClient")
    @patch("scrapper.save_json")
    @patch("scrapper.ensure_dir")
    def test_process_company_greenhouse_success(
        self,
        mock_ensure_dir,
        mock_save_json,
        mock_greenhouse_client_class
    ):
        """
        Verify process_company processes Greenhouse companies correctly
        """
        mock_client = Mock()
        mock_client.board_exists.return_value = True
        mock_client.get_jobs.return_value = [
            {"job_id": 1, "title": "Developer"}
        ]
        mock_greenhouse_client_class.return_value = mock_client

        result = process_company("airbnb", "greenhouse", run_id=None)

        self.assertEqual(result["status"], "success")
        self.assertEqual(result["job_count"], 1)
        mock_client.board_exists.assert_called_once_with("airbnb")
        mock_client.get_jobs.assert_called_once_with("airbnb")
        self.assertEqual(mock_save_json.call_count, 2)

    @patch("scrapper.LeverClient")
    @patch("scrapper.save_json")
    @patch("scrapper.ensure_dir")
    def test_process_company_lever_invalid(
        self,
        mock_ensure_dir,
        mock_save_json,
        mock_lever_client_class
    ):
        """
        Verify process_company returns invalid status if board does not exist
        """
        mock_client = Mock()
        mock_client.board_exists.return_value = False
        mock_lever_client_class.return_value = mock_client

        result = process_company("invalid-slug", "lever", run_id=None)

        self.assertEqual(result["status"], "invalid")
        mock_client.board_exists.assert_called_once_with("invalid-slug")
        mock_client.get_jobs.assert_not_called()

    @patch("scrapper.load_yaml_config")
    @patch("scrapper.start_run")
    @patch("scrapper.finish_run")
    @patch("scrapper.process_company")
    @patch("scrapper.save_json")
    @patch("scrapper.ensure_dir")
    def test_main_orchestration(
        self,
        mock_ensure_dir,
        mock_save_json,
        mock_process_company,
        mock_finish_run,
        mock_start_run,
        mock_load_yaml_config
    ):
        """
        Verify main runs the scraper jobs for all configured platforms
        """
        mock_load_yaml_config.return_value = {
            "greenhouse": ["airbnb"],
            "lever": ["stripe"]
        }
        mock_start_run.return_value = "run-123"
        mock_process_company.return_value = {"status": "success"}

        main()

        mock_start_run.assert_called_once()
        self.assertEqual(mock_process_company.call_count, 2)
        mock_process_company.assert_any_call("airbnb", "greenhouse", "run-123")
        mock_process_company.assert_any_call("stripe", "lever", "run-123")
        mock_finish_run.assert_called_once_with("run-123", "success")
