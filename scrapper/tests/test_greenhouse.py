import unittest
from unittest.mock import Mock, patch
from job_sources.greenhouse import GreenhouseClient

class TestGreenhouseClient(unittest.TestCase):
    """
    Unit tests for GreenhouseClient class
    """

    def setUp(self):
        self.mock_session = Mock()
        self.client = GreenhouseClient(session=self.mock_session)

    def test_board_exists_true(self):
        """
        Verify board_exists returns True when response status code is 200
        """
        mock_response = Mock()
        mock_response.status_code = 200
        self.mock_session.get.return_value = mock_response

        exists = self.client.board_exists("airbnb")
        self.assertTrue(exists)
        self.mock_session.get.assert_called_once_with(
            "https://boards-api.greenhouse.io/v1/boards/airbnb",
            timeout=20
        )

    def test_board_exists_false(self):
        """
        Verify board_exists returns False when response status code is 404
        """
        mock_response = Mock()
        mock_response.status_code = 404
        self.mock_session.get.return_value = mock_response

        exists = self.client.board_exists("invalid-company-name")
        self.assertFalse(exists)

    def test_get_jobs_success(self):
        """
        Verify get_jobs fetches and normalizes jobs correctly
        """
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jobs": [
                {
                    "id": 12345,
                    "title": "Software Engineer",
                    "updated_at": "2026-07-19T20:00:00Z",
                    "absolute_url": "https://boards.greenhouse.io/airbnb/jobs/12345",
                    "language": "en",
                    "location": {"name": "San Francisco, CA"},
                    "departments": [{"name": "Engineering"}],
                    "offices": [{"name": "SF HQ"}],
                    "content": "<p>We are looking for a Software Engineer...</p>"
                }
            ]
        }
        self.mock_session.get.return_value = mock_response

        jobs = self.client.get_jobs("airbnb")
        self.assertEqual(len(jobs), 1)
        job = jobs[0]
        self.assertEqual(job["job_id"], 12345)
        self.assertEqual(job["title"], "Software Engineer")
        self.assertEqual(job["absolute_url"], "https://boards.greenhouse.io/airbnb/jobs/12345")
        self.assertEqual(job["location"], "San Francisco, CA")
        self.assertEqual(job["departments"], ["Engineering"])
        self.assertEqual(job["offices"], ["SF HQ"])
        self.assertIn("We are looking for a Software Engineer", job["description_text"])

    def test_get_jobs_not_found(self):
        """
        Verify get_jobs returns empty list when API returns 404
        """
        mock_response = Mock()
        mock_response.status_code = 404
        self.mock_session.get.return_value = mock_response

        jobs = self.client.get_jobs("invalid-company-name")
        self.assertEqual(jobs, [])
