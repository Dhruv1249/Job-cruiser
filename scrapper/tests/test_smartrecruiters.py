import unittest
from unittest.mock import Mock
from job_sources.smartrecruiters import SmartRecruitersClient

class TestSmartRecruitersClient(unittest.TestCase):
    """
    Unit tests for SmartRecruitersClient class
    """

    def setUp(self):
        self.mock_session = Mock()
        self.client = SmartRecruitersClient(session=self.mock_session)

    def test_board_exists_true(self):
        """
        Verify board_exists returns True when response status code is 200
        """
        mock_response = Mock()
        mock_response.status_code = 200
        self.mock_session.get.return_value = mock_response

        exists = self.client.board_exists("visa")
        self.assertTrue(exists)

    def test_board_exists_false(self):
        """
        Verify board_exists returns False when response status code is 404
        """
        mock_response = Mock()
        mock_response.status_code = 404
        self.mock_session.get.return_value = mock_response

        exists = self.client.board_exists("invalid-company")
        self.assertFalse(exists)

    def test_get_jobs_success(self):
        """
        Verify get_jobs fetches and normalizes SmartRecruiters postings correctly
        """
        mock_list_response = Mock()
        mock_list_response.status_code = 200
        mock_list_response.json.return_value = {
            "content": [
                {
                    "id": "sr-123",
                    "name": "Security Engineer",
                    "releasedDate": "2026-07-19T20:00:00.000Z",
                    "location": {
                        "city": "Austin",
                        "region": "TX",
                        "country": "US"
                    },
                    "department": {
                        "name": "Global Security"
                    }
                }
            ]
        }

        mock_detail_response = Mock()
        mock_detail_response.status_code = 200
        mock_detail_response.json.return_value = {
            "jobAd": {
                "sections": {
                    "jobDescription": {
                        "text": "Secure our global network..."
                    }
                }
            }
        }

        self.mock_session.get.side_effect = [mock_list_response, mock_detail_response]

        jobs = self.client.get_jobs("visa")
        self.assertEqual(len(jobs), 1)
        job = jobs[0]
        self.assertEqual(job["job_id"], "sr-123")
        self.assertEqual(job["title"], "Security Engineer")
        self.assertEqual(job["location"], "Austin, TX, US")
        self.assertEqual(job["departments"], ["Global Security"])
        self.assertEqual(job["description_text"], "Secure our global network...")
