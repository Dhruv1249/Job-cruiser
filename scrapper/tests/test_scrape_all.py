import unittest
from unittest.mock import Mock, patch
from scrape_all import normalize_job_post, deduplicate_jobs, run_orchestration, is_location_in_scope

class TestScrapeAllOrchestrator(unittest.TestCase):
    """
    Unit tests for the scrape_all orchestrator
    """

    def test_is_location_in_scope(self):
        """
        Verify location scope filtering accepts India/remote and rejects unknown/non-India by default
        """
        self.assertTrue(is_location_in_scope(""))
        self.assertTrue(is_location_in_scope(None))
        self.assertTrue(is_location_in_scope("Bengaluru, India"))
        self.assertTrue(is_location_in_scope("Hyderabad, Telangana"))
        self.assertTrue(is_location_in_scope("Pune, Maharashtra"))
        self.assertTrue(is_location_in_scope("Anywhere in India - Remote"))
        self.assertTrue(is_location_in_scope("Remote"))
        self.assertTrue(is_location_in_scope("Global Remote"))
        self.assertTrue(is_location_in_scope("Worldwide"))
        self.assertTrue(is_location_in_scope("Work from home"))
        self.assertTrue(is_location_in_scope("Distributed"))
        self.assertTrue(is_location_in_scope("HQ situated in Paris, WFH available"))
        self.assertTrue(is_location_in_scope("Paris (Remote Eligible)"))
        self.assertTrue(is_location_in_scope("Berlin / Remote"))
        self.assertFalse(is_location_in_scope("US Remote Only"))
        self.assertFalse(is_location_in_scope("UK Remote Only"))
        self.assertFalse(is_location_in_scope("EU Remote Only"))
        self.assertFalse(is_location_in_scope("US Citizenship Required"))
        self.assertFalse(is_location_in_scope("San Francisco, CA"))
        self.assertFalse(is_location_in_scope("London, UK"))
        self.assertFalse(is_location_in_scope("Berlin, Germany"))
        self.assertFalse(is_location_in_scope("Toronto, Canada"))
        self.assertFalse(is_location_in_scope("New York, NY"))
        self.assertFalse(is_location_in_scope("Tokyo, Japan"))
        self.assertFalse(is_location_in_scope("Seoul, South Korea"))


    def test_normalize_job_post_jobspy(self):
        """
        Verify that job posts from jobspy are normalized correctly
        """
        jobspy_post = Mock()
        jobspy_post.id = "job-123"
        jobspy_post.title = "Software Engineer"
        jobspy_post.company_name = "Notion"
        jobspy_post.job_url = "https://jobs.notion.so/123"
        jobspy_post.description = "We are hiring..."
        
        mock_location = Mock()
        mock_location.display_location.return_value = "San Francisco, CA"
        jobspy_post.location = mock_location
        
        mock_date = Mock()
        mock_date.isoformat.return_value = "2026-07-19"
        jobspy_post.date_posted = mock_date

        normalized = normalize_job_post(jobspy_post, "linkedin")

        self.assertEqual(normalized["job_id"], "job-123")
        self.assertEqual(normalized["title"], "Software Engineer")
        self.assertEqual(normalized["absolute_url"], "https://jobs.notion.so/123")
        self.assertEqual(normalized["location"], "San Francisco, CA")
        self.assertEqual(normalized["description_text"], "We are hiring...")
        self.assertEqual(normalized["updated_at"], "2026-07-19T00:00:00Z")

    def test_deduplicate_jobs(self):
        """
        Verify that duplicate jobs based on title, company, and location are removed
        """
        jobs = [
            {
                "title": "Engineer",
                "company": "Stripe",
                "location": "Remote",
                "absolute_url": "https://stripe.com/1"
            },
            {
                "title": "Engineer",
                "company": "Stripe",
                "location": "Remote",
                "absolute_url": "https://stripe.com/2"
            },
            {
                "title": "Designer",
                "company": "Stripe",
                "location": "Remote",
                "absolute_url": "https://stripe.com/3"
            }
        ]
        deduped = deduplicate_jobs(jobs)
        self.assertEqual(len(deduped), 2)

    @patch("scrape_all.load_yaml_config")
    @patch("scrape_all.scrape_jobs")
    @patch("scrape_all.process_company")
    @patch("scrape_all.start_run")
    @patch("scrape_all.finish_run")
    @patch("scrape_all.save_json")
    def test_run_orchestration(
        self,
        mock_save_json,
        mock_finish_run,
        mock_start_run,
        mock_process_company,
        mock_scrape_jobs,
        mock_load_config
    ):
        """
        Verify the complete orchestration pipeline execution
        """
        mock_load_config.return_value = {
            "greenhouse": ["airbnb"],
            "lever": ["spotify"],
            "ashby": []
        }
        mock_start_run.return_value = "run-123"
        mock_process_company.return_value = {
            "company": "airbnb",
            "platform": "greenhouse",
            "jobs": [],
            "status": "success"
        }

        mock_pandas_df = Mock()
        mock_pandas_df.itertuples.return_value = []
        mock_scrape_jobs.return_value = mock_pandas_df

        results = run_orchestration()

        self.assertIn("manifest", results)
        self.assertGreaterEqual(mock_scrape_jobs.call_count, 75)
        mock_process_company.assert_any_call("airbnb", "greenhouse", "run-123")
        mock_process_company.assert_any_call("spotify", "lever", "run-123")
        mock_finish_run.assert_called_once_with("run-123", "success")
