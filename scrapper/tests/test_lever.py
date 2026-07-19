import unittest
from unittest.mock import Mock
from job_sources.lever import LeverClient

class TestLeverClient(unittest.TestCase):
    """
    Unit tests for LeverClient class
    """

    def setUp(self):
        self.mock_session = Mock()
        self.client = LeverClient(session=self.mock_session)

    def test_board_exists_true(self):
        """
        Verify board_exists returns True when response status code is 200
        """
        mock_response = Mock()
        mock_response.status_code = 200
        self.mock_session.get.return_value = mock_response

        exists = self.client.board_exists("stripe")
        self.assertTrue(exists)
        self.mock_session.get.assert_called_once_with(
            "https://api.lever.co/v0/postings/stripe?limit=1",
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
        Verify get_jobs fetches and normalizes Lever jobs correctly
        """
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = [
            {
                "id": "lever-123",
                "title": "Backend Engineer",
                "createdAt": 1721415600000,
                "hostedUrl": "https://jobs.lever.co/stripe/lever-123",
                "categories": {
                    "commitment": "Full-time",
                    "department": "Engineering",
                    "location": "Remote, US",
                    "team": "Payments"
                },
                "description": "<p>Build the future of payments...</p>",
                "descriptionPlain": "Build the future of payments...",
                "lists": [
                    {
                        "text": "What you'll do",
                        "content": "<ul><li>Write clean Python/Go code</li></ul>"
                    }
                ]
            }
        ]
        self.mock_session.get.return_value = mock_response

        jobs = self.client.get_jobs("stripe")
        self.assertEqual(len(jobs), 1)
        job = jobs[0]
        self.assertEqual(job["job_id"], "lever-123")
        self.assertEqual(job["title"], "Backend Engineer")
        self.assertEqual(job["absolute_url"], "https://jobs.lever.co/stripe/lever-123")
        self.assertEqual(job["location"], "Remote, US")
        self.assertEqual(job["departments"], ["Engineering", "Payments"])
        self.assertEqual(job["offices"], ["Remote, US"])
        self.assertIn("Build the future of payments", job["description_text"])
        self.assertIn("Write clean Python/Go code", job["description_text"])

    def test_get_jobs_not_found(self):
        """
        Verify get_jobs returns empty list when API returns 404
        """
        mock_response = Mock()
        mock_response.status_code = 404
        self.mock_session.get.return_value = mock_response

        jobs = self.client.get_jobs("invalid-company-name")
        self.assertEqual(jobs, [])
