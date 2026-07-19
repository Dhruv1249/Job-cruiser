import unittest
from unittest.mock import Mock
from job_sources.ashby import AshbyClient

class TestAshbyClient(unittest.TestCase):
    """
    Unit tests for AshbyClient class
    """

    def setUp(self):
        self.mock_session = Mock()
        self.client = AshbyClient(session=self.mock_session)

    def test_board_exists_true(self):
        """
        Verify board_exists returns True when response status code is 200
        """
        mock_response = Mock()
        mock_response.status_code = 200
        self.mock_session.get.return_value = mock_response

        exists = self.client.board_exists("notion")
        self.assertTrue(exists)
        self.mock_session.get.assert_called_once_with(
            "https://api.ashbyhq.com/posting-api/job-board/notion",
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
        Verify get_jobs fetches and normalizes Ashby jobs correctly
        """
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "jobs": [
                {
                    "id": "ashby-123",
                    "title": "Staff Engineer",
                    "publishedAt": "2026-07-19T20:00:00.000Z",
                    "jobUrl": "https://jobs.ashbyhq.com/notion/ashby-123",
                    "location": "New York, NY",
                    "department": "Engineering",
                    "team": "Core Workspace",
                    "descriptionHtml": "<h1>Overview</h1><p>We are hiring a Staff Engineer...</p>"
                }
            ]
        }
        self.mock_session.get.return_value = mock_response

        jobs = self.client.get_jobs("notion")
        self.assertEqual(len(jobs), 1)
        job = jobs[0]
        self.assertEqual(job["job_id"], "ashby-123")
        self.assertEqual(job["title"], "Staff Engineer")
        self.assertEqual(job["absolute_url"], "https://jobs.ashbyhq.com/notion/ashby-123")
        self.assertEqual(job["location"], "New York, NY")
        self.assertEqual(job["departments"], ["Engineering", "Core Workspace"])
        self.assertEqual(job["offices"], ["New York, NY"])
        self.assertIn("We are hiring a Staff Engineer", job["description_text"])

    def test_get_jobs_not_found(self):
        """
        Verify get_jobs returns empty list when API returns 404
        """
        mock_response = Mock()
        mock_response.status_code = 404
        self.mock_session.get.return_value = mock_response

        jobs = self.client.get_jobs("invalid-company-name")
        self.assertEqual(jobs, [])
