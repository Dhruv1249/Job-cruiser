import unittest
from unittest.mock import Mock
from job_sources.workday import WorkdayClient

class TestWorkdayClient(unittest.TestCase):
    """
    Unit tests for WorkdayClient class
    """

    def setUp(self):
        self.mock_session = Mock()
        self.client = WorkdayClient(session=self.mock_session)

    def test_board_exists_true(self):
        """
        Verify board_exists returns True when response status code is 200
        """
        mock_response = Mock()
        mock_response.status_code = 200
        self.mock_session.post.return_value = mock_response

        exists = self.client.board_exists("netflix")
        self.assertTrue(exists)

    def test_board_exists_false(self):
        """
        Verify board_exists returns False when response status code is 404
        """
        mock_response = Mock()
        mock_response.status_code = 404
        self.mock_session.post.return_value = mock_response

        exists = self.client.board_exists("invalid-company")
        self.assertFalse(exists)

    def test_get_jobs_success(self):
        """
        Verify get_jobs fetches and normalizes Workday postings correctly
        """
        mock_list_response = Mock()
        mock_list_response.status_code = 200
        mock_list_response.json.return_value = {
            "jobPostings": [
                {
                    "externalPath": "/job/SF/Engineer_R1",
                    "title": "Staff Engineer",
                    "postedOn": "2026-07-19",
                    "locationsText": "San Francisco, CA"
                }
            ]
        }

        mock_detail_response = Mock()
        mock_detail_response.status_code = 200
        mock_detail_response.json.return_value = {
            "jobPostingInfo": {
                "id": "R1",
                "jobDescription": "<p>Build great things...</p>"
            }
        }

        self.mock_session.post.return_value = mock_list_response
        self.mock_session.get.return_value = mock_detail_response

        jobs = self.client.get_jobs("netflix")
        self.assertEqual(len(jobs), 1)
        job = jobs[0]
        self.assertEqual(job["job_id"], "R1")
        self.assertEqual(job["title"], "Staff Engineer")
        self.assertEqual(job["location"], "San Francisco, CA")
        self.assertIn("Build great things", job["description_text"])
