"""
Unit tests for the job search scraper orchestrator module.
"""

import unittest
from unittest.mock import Mock, patch
from scrape_all import (
    normalize_job_post,
    deduplicate_jobs,
    run_orchestration,
    is_location_in_scope,
    extract_ats_slug,
    sanitize_company_name,
)


class TestScrapeAllOrchestrator(unittest.TestCase):
    """
    Test suite validating normalization, deduplication, scope filtering, and orchestration.
    """

    def test_is_location_in_scope(self):
        """
        Verify location scope filtering accepts Indian and global remote positions while rejecting excluded locales.
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
        self.assertTrue(is_location_in_scope("San Francisco, CA", is_remote_position=True))
        self.assertFalse(is_location_in_scope("US Remote Only"))
        self.assertFalse(is_location_in_scope("UK Remote Only"))
        self.assertFalse(is_location_in_scope("EU Remote Only"))
        self.assertFalse(is_location_in_scope("US Citizenship Required"))
        self.assertFalse(is_location_in_scope("San Francisco, CA", is_remote_position=False))
        self.assertFalse(is_location_in_scope("London, UK"))
        self.assertFalse(is_location_in_scope("Berlin, Germany"))

    def test_normalize_job_post_jobspy(self):
        """
        Verify that job posts from jobspy objects are normalized correctly.
        """
        mock_job_post = Mock()
        mock_job_post.id = "job-123"
        mock_job_post.title = "Software Engineer"
        mock_job_post.company_name = "Notion"
        mock_job_post.job_url = "https://jobs.notion.so/123"
        mock_job_post.description = "We are hiring..."

        mock_location = Mock()
        mock_location.display_location.return_value = "San Francisco, CA"
        mock_job_post.location = mock_location

        mock_date = Mock()
        mock_date.isoformat.return_value = "2026-07-19"
        mock_job_post.date_posted = mock_date

        normalized = normalize_job_post(mock_job_post, "linkedin")

        self.assertEqual(normalized["job_id"], "job-123")
        self.assertEqual(normalized["title"], "Software Engineer")
        self.assertEqual(normalized["absolute_url"], "https://jobs.notion.so/123")
        self.assertEqual(normalized["location"], "San Francisco, CA")
        self.assertEqual(normalized["description_text"], "We are hiring...")
        self.assertEqual(normalized["updated_at"], "2026-07-19T00:00:00Z")

    def test_deduplicate_jobs(self):
        """
        Verify that duplicate jobs based on title, company, and location are removed.
        """
        jobs_list = [
            {
                "title": "Engineer",
                "company": "Stripe",
                "location": "Remote",
                "absolute_url": "https://stripe.com/1",
            },
            {
                "title": "Engineer",
                "company": "Stripe",
                "location": "Remote",
                "absolute_url": "https://stripe.com/2",
            },
            {
                "title": "Designer",
                "company": "Stripe",
                "location": "Remote",
                "absolute_url": "https://stripe.com/3",
            },
        ]
        deduplicated = deduplicate_jobs(jobs_list)
        self.assertEqual(len(deduplicated), 2)

    def test_extract_ats_slug(self):
        """
        Verify ATS platform and slug extraction from various job posting URLs.
        """
        self.assertEqual(
            extract_ats_slug("https://boards.greenhouse.io/stripe/jobs/123"),
            ("greenhouse", "stripe"),
        )
        self.assertEqual(
            extract_ats_slug("https://jobs.lever.co/coda/abc-def"),
            ("lever", "coda"),
        )
        self.assertEqual(
            extract_ats_slug("https://jobs.ashbyhq.com/ramp/xyz"),
            ("ashby", "ramp"),
        )
        self.assertEqual(
            extract_ats_slug("https://jobs.smartrecruiters.com/Visa/743999"),
            ("smartrecruiters", "visa"),
        )
        self.assertIsNone(extract_ats_slug("https://indeed.com/viewjob?jk=123"))

    def test_sanitize_company_name(self):
        """
        Verify company name sanitization removes HTML and invalid characters.
        """
        self.assertEqual(sanitize_company_name("<b>Google</b>"), "Google")
        self.assertEqual(sanitize_company_name("Acme Corp/USA"), "Acme Corp USA")
        self.assertEqual(sanitize_company_name(None), "Unknown")
        self.assertEqual(sanitize_company_name(""), "Unknown")

    @patch("scrape_all.fetch_ats_slugs")
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
        mock_fetch_slugs,
    ):
        """
        Verify the orchestration pipeline execution with mocked ATS slugs and job sources.
        """
        mock_fetch_slugs.return_value = {
            "greenhouse": ["airbnb"],
            "lever": ["spotify"],
        }
        mock_start_run.return_value = "run-test-123"
        mock_process_company.return_value = {
            "company": "airbnb",
            "platform": "greenhouse",
            "jobs": [],
            "status": "success",
        }

        mock_dataframe = Mock()
        mock_dataframe.empty = True
        mock_dataframe.itertuples.return_value = []
        mock_scrape_jobs.return_value = mock_dataframe

        pipeline_result = run_orchestration()

        self.assertIn("manifest", pipeline_result)
        mock_process_company.assert_any_call("airbnb", "greenhouse", "run-test-123")
        mock_process_company.assert_any_call("spotify", "lever", "run-test-123")
        mock_finish_run.assert_called_once_with("run-test-123", "success")


if __name__ == "__main__":
    unittest.main()
