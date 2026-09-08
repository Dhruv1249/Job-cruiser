"""
Unit tests for the job search scraper orchestrator module.
"""

import unittest
from unittest.mock import Mock, patch, ANY
from jobspy.model import Site
from scrape_all import (
    normalize_job_post,
    deduplicate_jobs,
    run_orchestration,
    is_location_in_scope,
    extract_ats_slug,
    sanitize_company_name,
    enrich_linkedin_descriptions,
    fetch_single_linkedin_description,
    process_direct_career_company,
    is_valid_direct_career_post,
    validate_proxy,
    filter_healthy_proxies,
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
        mock_job_post.min_amount = None
        mock_job_post.max_amount = None
        mock_job_post.currency = None
        mock_job_post.departments = []
        mock_job_post.offices = []
        mock_job_post.updated_at = ""

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
        self.assertEqual(normalized["salary_min"], 0)
        self.assertEqual(normalized["salary_max"], 0)
        self.assertEqual(normalized["currency"], "")

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

    @patch("scrape_all.load_career_pages")
    @patch("scrape_all.enrich_linkedin_descriptions")
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
        mock_enrich,
        mock_load_career_pages,
    ):
        """
        Verify the orchestration pipeline execution with mocked ATS slugs, job sources, and enrichment trigger.
        """
        mock_load_career_pages.return_value = []
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

        with patch("scrape_all.KEYWORDS", ["backend engineer"]):
            pipeline_result = run_orchestration()

        self.assertIn("manifest", pipeline_result)
        mock_process_company.assert_any_call("airbnb", "greenhouse", "run-test-123")
        mock_process_company.assert_any_call("spotify", "lever", "run-test-123")
        mock_finish_run.assert_called_once_with("run-test-123", "success", None, ANY, ANY)
        mock_enrich.assert_called_once()

    @patch("scrape_all.load_career_pages")
    @patch("scrape_all.enrich_linkedin_descriptions")
    @patch("scrape_all.fetch_ats_slugs")
    @patch("scrape_all.scrape_jobs")
    @patch("scrape_all.process_company")
    @patch("scrape_all.start_run")
    @patch("scrape_all.finish_run")
    @patch("scrape_all.save_json")
    def test_run_orchestration_disables_linkedin_fetch_description_in_discovery(
        self,
        mock_save_json,
        mock_finish_run,
        mock_start_run,
        mock_process_company,
        mock_scrape_jobs,
        mock_fetch_slugs,
        mock_enrich,
        mock_load_career_pages,
    ):
        """
        Verify that linkedin_fetch_description is disabled during discovery pass and enrichment is triggered.
        """
        mock_load_career_pages.return_value = []
        mock_fetch_slugs.return_value = {}
        mock_start_run.return_value = "run-test-linkedin"
        mock_dataframe = Mock()
        mock_dataframe.empty = True
        mock_dataframe.itertuples.return_value = []
        mock_scrape_jobs.return_value = mock_dataframe

        with patch("scrape_all.KEYWORDS", ["backend engineer"]), \
             patch("scrape_all.KEYWORD_SEARCHABLE_INDIA_SITES", [Site.LINKEDIN]), \
             patch("scrape_all.KEYWORD_SEARCHABLE_REMOTE_SITES", []), \
             patch("scrape_all.SINGLE_CALL_FEED_SITES", []):
            run_orchestration()

        linkedin_calls = [
            call_kwargs for _, call_kwargs in mock_scrape_jobs.call_args_list
            if call_kwargs.get("site_name") == [Site.LINKEDIN]
        ]
        self.assertTrue(len(linkedin_calls) > 0)
        for call_kwargs in linkedin_calls:
            self.assertFalse(call_kwargs.get("linkedin_fetch_description", False))
        mock_enrich.assert_called_once()

    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
    ):
        """
        Verify that pending LinkedIn jobs are fetched from backend, descriptions are scraped and posted.
        """
        mock_pending_response = Mock()
        mock_pending_response.status_code = 200
        mock_pending_response.json.return_value = {
            "data": [
                {
                    "id": "job-uuid-1",
                    "url": "https://www.linkedin.com/jobs/view/backend-dev-4462613977",
                    "title": "Backend Dev",
                }
            ]
        }
        mock_requests_get.return_value = mock_pending_response

        mock_session_instance = Mock()
        mock_detail_response = Mock()
        mock_detail_response.status_code = 200
        mock_detail_response.url = "https://www.linkedin.com/jobs/view/4462613977"
        mock_detail_response.text = """
        <html>
            <body>
                <div class="description__text">
                    <p>We are looking for a Go engineer to scale distributed databases.</p>
                    <button class="show-more-less-button">Show more</button>
                </div>
            </body>
        </html>
        """
        mock_session_instance.get.return_value = mock_detail_response
        mock_session_class.return_value = mock_session_instance

        mock_update_response = Mock()
        mock_update_response.status_code = 200
        mock_requests_post.return_value = mock_update_response

        enriched_count = enrich_linkedin_descriptions(cooldown_seconds=0, max_jobs=10)
        self.assertEqual(enriched_count, 1)

        mock_requests_post.assert_called_once()
        post_kwargs = mock_requests_post.call_args[1]
        self.assertEqual(len(post_kwargs["json"]["updates"]), 1)
        self.assertEqual(post_kwargs["json"]["updates"][0]["id"], "job-uuid-1")
        self.assertIn("Go engineer", post_kwargs["json"]["updates"][0]["description_text"])
        self.assertNotIn("Show more", post_kwargs["json"]["updates"][0]["description_text"])

    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions_multiple_batches(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
    ):
        """
        Verify that enrichment iterates across multiple batches until pending jobs are exhausted.
        """
        batch_one_response = Mock()
        batch_one_response.status_code = 200
        batch_one_response.json.return_value = {
            "data": [
                {
                    "id": "job-uuid-batch-1",
                    "url": "https://www.linkedin.com/jobs/view/backend-1-11111",
                    "title": "Backend 1",
                }
            ]
        }

        batch_two_response = Mock()
        batch_two_response.status_code = 200
        batch_two_response.json.return_value = {
            "data": [
                {
                    "id": "job-uuid-batch-2",
                    "url": "https://www.linkedin.com/jobs/view/backend-2-22222",
                    "title": "Backend 2",
                }
            ]
        }

        empty_batch_response = Mock()
        empty_batch_response.status_code = 200
        empty_batch_response.json.return_value = {"data": []}

        mock_requests_get.side_effect = [
            batch_one_response,
            batch_two_response,
            empty_batch_response,
        ]

        mock_session_instance = Mock()
        mock_detail_response = Mock()
        mock_detail_response.status_code = 200
        mock_detail_response.url = "https://www.linkedin.com/jobs/view/11111"
        mock_detail_response.text = """
        <html>
            <body>
                <div class="description__text">
                    <p>Description for role.</p>
                </div>
            </body>
        </html>
        """
        mock_session_instance.get.return_value = mock_detail_response
        mock_session_class.return_value = mock_session_instance

        mock_update_response = Mock()
        mock_update_response.status_code = 200
        mock_requests_post.return_value = mock_update_response

        total_enriched = enrich_linkedin_descriptions(cooldown_seconds=0, batch_size=1)
        self.assertEqual(total_enriched, 2)
        self.assertEqual(mock_requests_post.call_count, 2)

    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions_halts_when_batch_only_contains_attempted_jobs(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
    ):
        """
        Verify that enrichment halts when all returned jobs in a batch were already attempted.
        """
        unparseable_job_response = Mock()
        unparseable_job_response.status_code = 200
        unparseable_job_response.json.return_value = {
            "data": [
                {
                    "id": "job-uuid-authwall",
                    "url": "https://www.linkedin.com/jobs/view/authwall-99999",
                    "title": "Authwalled Job",
                }
            ]
        }
        mock_requests_get.return_value = unparseable_job_response

        mock_session_instance = Mock()
        mock_detail_response = Mock()
        mock_detail_response.status_code = 200
        mock_detail_response.url = "https://www.linkedin.com/authwall"
        mock_detail_response.text = "<html><body>Authwall</body></html>"
        mock_session_instance.get.return_value = mock_detail_response
        mock_session_class.return_value = mock_session_instance

        total_enriched = enrich_linkedin_descriptions(cooldown_seconds=0, batch_size=1)
        self.assertEqual(total_enriched, 0)
        self.assertEqual(mock_requests_get.call_count, 2)
        mock_requests_post.assert_not_called()

    @patch("scrape_all.stream_ingest_jobs")
    def test_process_direct_career_company(self, mock_stream_ingest):
        """
        Verify process_direct_career_company scrapes, normalizes, filters in-scope jobs, and streams them.
        """
        mock_scraper = Mock()
        mock_job = Mock()
        mock_job.id = "https://stripe.com/jobs/1"
        mock_job.title = "Software Engineer"
        mock_job.company = "Stripe"
        mock_job.job_url = "https://stripe.com/jobs/1"
        mock_job.location = Mock(country="Remote")
        mock_job.description = "We are seeking a senior Go backend engineer to build scalable payment infrastructure services globally."
        mock_job.is_remote = True
        mock_job.date_posted = None

        mock_scraper.scrape_single_company.return_value = [mock_job]

        result = process_direct_career_company("Stripe", "https://stripe.com/jobs", mock_scraper, "run-test-dc")

        self.assertEqual(result["status"], "success")
        self.assertEqual(len(result["jobs"]), 1)
        self.assertEqual(result["jobs"][0]["company"], "Stripe")
        mock_stream_ingest.assert_called_once_with(result["jobs"], "run-test-dc")

    @patch("scrape_all.time.sleep")
    def test_fetch_single_linkedin_description_success(self, mock_sleep):
        """
        Verify fetch_single_linkedin_description extracts clean text and decomposes buttons.
        """
        mock_session = Mock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.url = "https://www.linkedin.com/jobs/view/9876543210"
        mock_response.text = """
        <html>
            <body>
                <div class="show-more-less-html__markup">
                    <p>Seeking senior distributed systems engineer with Go and Python experience.</p>
                    <button class="show-more-less-button">Show more</button>
                </div>
            </body>
        </html>
        """
        mock_session.get.return_value = mock_response

        job_input = {
            "id": "job-success-1",
            "url": "https://www.linkedin.com/jobs/view/9876543210",
        }
        update_result, status_code, proxy_failed = fetch_single_linkedin_description(job_input, mock_session)

        self.assertEqual(status_code, 200)
        self.assertFalse(proxy_failed)
        self.assertIsNotNone(update_result)
        self.assertEqual(update_result["id"], "job-success-1")
        self.assertIn("distributed systems", update_result["description_text"])
        self.assertNotIn("Show more", update_result["description_text"])

    @patch("scrape_all.time.sleep")
    def test_fetch_single_linkedin_description_authwall(self, mock_sleep):
        """
        Verify fetch_single_linkedin_description skips jobs redirecting to LinkedIn authwalls.
        """
        mock_session = Mock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.url = "https://www.linkedin.com/authwall?trk=bf"
        mock_response.text = "<html><body>Authwall</body></html>"
        mock_session.get.return_value = mock_response

        job_input = {
            "id": "job-authwall-1",
            "url": "https://www.linkedin.com/jobs/view/1234567890",
        }
        update_result, status_code, proxy_failed = fetch_single_linkedin_description(job_input, mock_session)

        self.assertEqual(status_code, 200)
        self.assertFalse(proxy_failed)
        self.assertIsNone(update_result)

    @patch("scrape_all.time.sleep")
    def test_fetch_single_linkedin_description_rate_limited(self, mock_sleep):
        """
        Verify fetch_single_linkedin_description returns 429 status code when rate limited.
        """
        mock_session = Mock()
        mock_response = Mock()
        mock_response.status_code = 429
        mock_response.url = "https://www.linkedin.com/jobs/view/1122334455"
        mock_session.get.return_value = mock_response

        job_input = {
            "id": "job-rate-limited-1",
            "url": "https://www.linkedin.com/jobs/view/1122334455",
        }
        update_result, status_code, proxy_failed = fetch_single_linkedin_description(job_input, mock_session)

        self.assertEqual(status_code, 429)
        self.assertFalse(proxy_failed)
        self.assertIsNone(update_result)

    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions_parallel_workers(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
    ):
        """
        Verify enrichment with bounded worker pool processes multiple jobs in parallel and posts updates.
        """
        mock_pending_response = Mock()
        mock_pending_response.status_code = 200
        mock_pending_response.json.return_value = {
            "data": [
                {
                    "id": f"job-parallel-{index}",
                    "url": f"https://www.linkedin.com/jobs/view/{1000 + index}",
                    "title": f"Role {index}",
                }
                for index in range(4)
            ]
        }
        mock_empty_response = Mock()
        mock_empty_response.status_code = 200
        mock_empty_response.json.return_value = {"data": []}
        mock_requests_get.side_effect = [mock_pending_response, mock_empty_response]

        mock_session_instance = Mock()
        mock_detail_response = Mock()
        mock_detail_response.status_code = 200
        mock_detail_response.url = "https://www.linkedin.com/jobs/view/1000"
        mock_detail_response.text = """
        <html><body><div class="description__text">Software Engineer description</div></body></html>
        """
        mock_session_instance.get.return_value = mock_detail_response
        mock_session_class.return_value = mock_session_instance

        mock_update_response = Mock()
        mock_update_response.status_code = 200
        mock_requests_post.return_value = mock_update_response

        total_enriched = enrich_linkedin_descriptions(cooldown_seconds=0, max_workers=4)
        self.assertEqual(total_enriched, 4)
        mock_requests_post.assert_called_once()
        post_kwargs = mock_requests_post.call_args[1]
        self.assertEqual(len(post_kwargs["json"]["updates"]), 4)

    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions_halts_on_consecutive_rate_limits(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
    ):
        """
        Verify enrichment stops processing when encountering consecutive 429 rate limit responses.
        """
        mock_pending_response = Mock()
        mock_pending_response.status_code = 200
        mock_pending_response.json.return_value = {
            "data": [
                {
                    "id": f"job-rate-limited-{index}",
                    "url": f"https://www.linkedin.com/jobs/view/{2000 + index}",
                    "title": f"Role {index}",
                }
                for index in range(4)
            ]
        }
        mock_requests_get.return_value = mock_pending_response

        mock_session_instance = Mock()
        mock_detail_response = Mock()
        mock_detail_response.status_code = 429
        mock_detail_response.url = "https://www.linkedin.com/jobs/view/2000"
        mock_session_instance.get.return_value = mock_detail_response
        mock_session_class.return_value = mock_session_instance

        total_enriched = enrich_linkedin_descriptions(cooldown_seconds=0, max_workers=2)
        self.assertEqual(total_enriched, 0)
        mock_requests_post.assert_not_called()

    @patch("scrape_all.time.sleep")
    def test_fetch_single_linkedin_description_routes_via_proxy(self, mock_sleep):
        """
        Verify fetch_single_linkedin_description passes proxy dictionary to requests.
        """
        mock_session = Mock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.url = "https://www.linkedin.com/jobs/view/7788990011"
        mock_response.text = "<html><body><div class='description__text'>Proxy routed job</div></body></html>"
        mock_session.get.return_value = mock_response

        job_input = {
            "id": "job-proxy-1",
            "url": "https://www.linkedin.com/jobs/view/7788990011",
        }
        update_result, status_code, proxy_failed = fetch_single_linkedin_description(
            job_input, mock_session, proxy_url="http://34.120.50.10:8888"
        )

        self.assertEqual(status_code, 200)
        self.assertFalse(proxy_failed)
        self.assertIsNotNone(update_result)
        mock_session.get.assert_called_once()
        call_kwargs = mock_session.get.call_args[1]
        self.assertEqual(
            call_kwargs.get("proxies"),
            {
                "http": "http://34.120.50.10:8888",
                "https": "http://34.120.50.10:8888",
            },
        )

    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions_dynamic_proxy_scaling(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
    ):
        """
        Verify enrichment dynamically distributes jobs across available routes when proxies are configured.
        """
        mock_pending_response = Mock()
        mock_pending_response.status_code = 200
        mock_pending_response.json.return_value = {
            "data": [
                {
                    "id": f"job-proxy-batch-{index}",
                    "url": f"https://www.linkedin.com/jobs/view/{5000 + index}",
                    "title": f"Proxy Role {index}",
                }
                for index in range(4)
            ]
        }
        mock_empty_response = Mock()
        mock_empty_response.status_code = 200
        mock_empty_response.json.return_value = {"data": []}
        mock_requests_get.side_effect = [mock_pending_response, mock_empty_response]

        mock_session_instance = Mock()
        mock_detail_response = Mock()
        mock_detail_response.status_code = 200
        mock_detail_response.url = "https://www.linkedin.com/jobs/view/5000"
        mock_detail_response.text = "<html><body><div class='description__text'>Role description</div></body></html>"
        mock_session_instance.get.return_value = mock_detail_response
        mock_session_class.return_value = mock_session_instance

        mock_update_response = Mock()
        mock_update_response.status_code = 200
        mock_requests_post.return_value = mock_update_response

        configured_proxies = [
            "http://proxy1:8888",
            "http://proxy2:8888",
        ]
        with patch("scrape_all.PROXIES", configured_proxies):
            total_enriched = enrich_linkedin_descriptions(cooldown_seconds=0, max_workers=None)

        self.assertEqual(total_enriched, 4)
        mock_requests_post.assert_called_once()
        self.assertEqual(mock_session_instance.get.call_count, 4)

    @patch("scrape_all.requests.head")
    def test_validate_proxy_healthy(self, mock_requests_head):
        """
        Verify validate_proxy returns true when proxy responds with valid HTTP status.
        """
        mock_response = Mock()
        mock_response.status_code = 200
        mock_requests_head.return_value = mock_response

        self.assertTrue(validate_proxy("http://valid-proxy:8888"))
        mock_requests_head.assert_called_once()
        kwargs = mock_requests_head.call_args[1]
        self.assertEqual(
            kwargs.get("proxies"),
            {
                "http": "http://valid-proxy:8888",
                "https": "http://valid-proxy:8888",
            },
        )

    @patch("scrape_all.requests.head")
    def test_validate_proxy_unhealthy_on_exception(self, mock_requests_head):
        """
        Verify validate_proxy returns false when proxy request raises an exception.
        """
        import requests
        mock_requests_head.side_effect = requests.exceptions.ProxyError("Connection refused")

        self.assertFalse(validate_proxy("http://broken-proxy:8888"))

    @patch("scrape_all.validate_proxy")
    def test_filter_healthy_proxies(self, mock_validate_proxy):
        """
        Verify filter_healthy_proxies returns only responsive proxies from candidate list.
        """
        mock_validate_proxy.side_effect = lambda proxy, timeout=3.0: "good" in proxy

        candidate_proxies = [
            "http://good-proxy-1:8888",
            "http://bad-proxy-1:8888",
            "http://good-proxy-2:8888",
        ]
        healthy = filter_healthy_proxies(candidate_proxies)
        self.assertEqual(healthy, ["http://good-proxy-1:8888", "http://good-proxy-2:8888"])

    @patch("scrape_all.time.sleep")
    def test_fetch_single_linkedin_description_proxy_error_in_flight_fallback(self, mock_sleep):
        """
        Verify fetch_single_linkedin_description falls back to direct connection when proxy fails in-flight.
        """
        import requests
        mock_session = Mock()
        proxy_error = requests.exceptions.ProxyError("Proxy tunnel failed")
        direct_success_response = Mock()
        direct_success_response.status_code = 200
        direct_success_response.url = "https://www.linkedin.com/jobs/view/9988776655"
        direct_success_response.text = (
            "<html><body><div class='description__text'>Direct fallback succeeded</div></body></html>"
        )
        mock_session.get.side_effect = [proxy_error, direct_success_response]

        job_input = {
            "id": "job-fallback-1",
            "url": "https://www.linkedin.com/jobs/view/9988776655",
        }
        update_result, status_code, proxy_failed = fetch_single_linkedin_description(
            job_input, mock_session, proxy_url="http://failing-proxy:8888"
        )

        self.assertEqual(status_code, 200)
        self.assertTrue(proxy_failed)
        self.assertIsNotNone(update_result)
        self.assertIn("Direct fallback succeeded", update_result["description_text"])
        self.assertEqual(mock_session.get.call_count, 2)
        second_call_kwargs = mock_session.get.call_args_list[1][1]
        self.assertIsNone(second_call_kwargs.get("proxies"))

    @patch("scrape_all.filter_healthy_proxies")
    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions_filters_dead_proxies_at_startup(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
        mock_filter_healthy_proxies,
    ):
        """
        Verify enrichment filters dead proxies and proceeds with remaining healthy routes.
        """
        mock_filter_healthy_proxies.return_value = []

        mock_pending_response = Mock()
        mock_pending_response.status_code = 200
        mock_pending_response.json.return_value = {
            "data": [
                {
                    "id": "job-dead-proxy-1",
                    "url": "https://www.linkedin.com/jobs/view/12345",
                    "title": "Role 1",
                }
            ]
        }
        mock_empty_response = Mock()
        mock_empty_response.status_code = 200
        mock_empty_response.json.return_value = {"data": []}
        mock_requests_get.side_effect = [mock_pending_response, mock_empty_response]

        mock_session_instance = Mock()
        mock_detail_response = Mock()
        mock_detail_response.status_code = 200
        mock_detail_response.url = "https://www.linkedin.com/jobs/view/12345"
        mock_detail_response.text = "<html><body><div class='description__text'>Direct content</div></body></html>"
        mock_session_instance.get.return_value = mock_detail_response
        mock_session_class.return_value = mock_session_instance

        mock_update_response = Mock()
        mock_update_response.status_code = 200
        mock_requests_post.return_value = mock_update_response

        with patch("scrape_all.PROXIES", ["http://dead-proxy:8888"]):
            total_enriched = enrich_linkedin_descriptions(cooldown_seconds=0, max_workers=None)

        self.assertEqual(total_enriched, 1)
        mock_filter_healthy_proxies.assert_called_once_with(["http://dead-proxy:8888"])

    @patch("scrape_all.filter_healthy_proxies")
    @patch("scrape_all.fetch_single_linkedin_description")
    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions_evicts_failing_proxies_mid_run(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
        mock_fetch,
        mock_filter_healthy_proxies,
    ):
        """
        Verify that a proxy failing repeatedly mid-run is evicted from active routes, allowing surviving routes to finish.
        """
        mock_filter_healthy_proxies.return_value = ["http://dying-proxy:8888", "http://healthy-proxy:8888"]

        first_batch = {
            "data": [
                {"id": f"job-circuit-{index}", "url": f"https://www.linkedin.com/jobs/view/{7000 + index}"}
                for index in range(4)
            ]
        }
        empty_batch = {"data": []}
        mock_requests_get.side_effect = [
            Mock(status_code=200, json=lambda: first_batch),
            Mock(status_code=200, json=lambda: empty_batch),
        ]

        def simulate_fetch(job_record, session, halt_event, proxy_url):
            if proxy_url == "http://dying-proxy:8888":
                return {"id": job_record["id"], "description_text": "fallback text"}, 200, True
            return {"id": job_record["id"], "description_text": "healthy text"}, 200, False

        mock_fetch.side_effect = simulate_fetch
        mock_requests_post.return_value = Mock(status_code=200)

        with patch("scrape_all.PROXIES", ["http://dying-proxy:8888", "http://healthy-proxy:8888"]):
            total_enriched = enrich_linkedin_descriptions(cooldown_seconds=0, max_workers=3)

        self.assertEqual(total_enriched, 4)
        mock_requests_post.assert_called_once()

    @patch("scrape_all.validate_proxy")
    @patch("scrape_all.filter_healthy_proxies")
    @patch("scrape_all.fetch_single_linkedin_description")
    @patch("scrape_all.time.sleep")
    @patch("requests.post")
    @patch("requests.Session")
    @patch("requests.get")
    def test_enrich_linkedin_descriptions_restores_proxy_after_cooldown(
        self,
        mock_requests_get,
        mock_session_class,
        mock_requests_post,
        mock_sleep,
        mock_fetch,
        mock_filter_healthy_proxies,
        mock_validate_proxy,
    ):
        """
        Verify that a proxy placed in cooldown is probed and restored to active routes upon recovery.
        """
        mock_filter_healthy_proxies.return_value = ["http://recovering-proxy:8888"]
        mock_validate_proxy.return_value = True

        first_batch = {
            "data": [
                {"id": f"job-recover-fail-{index}", "url": f"https://www.linkedin.com/jobs/view/{8000 + index}"}
                for index in range(4)
            ]
        }
        second_batch = {
            "data": [
                {"id": f"job-recover-pass-{index}", "url": f"https://www.linkedin.com/jobs/view/{8010 + index}"}
                for index in range(2)
            ]
        }
        empty_batch = {"data": []}
        batch_sequence = [
            Mock(status_code=200, json=lambda: first_batch),
            Mock(status_code=200, json=lambda: second_batch),
            Mock(status_code=200, json=lambda: empty_batch),
        ]
        batch_iterator = iter(batch_sequence)
        virtual_clock_time = 100.0

        def get_virtual_time():
            return virtual_clock_time

        def mock_requests_get_handler(*args, **kwargs):
            nonlocal virtual_clock_time
            if mock_requests_get.call_count > 1:
                virtual_clock_time = 500.0
            return next(batch_iterator)

        mock_requests_get.side_effect = mock_requests_get_handler

        def simulate_fetch(job_record, session, halt_event, proxy_url):
            if "fail" in job_record["id"]:
                return {"id": job_record["id"], "description_text": "text"}, 200, True
            return {"id": job_record["id"], "description_text": "text"}, 200, False

        mock_fetch.side_effect = simulate_fetch
        mock_requests_post.return_value = Mock(status_code=200)

        with patch("scrape_all.time.time", side_effect=get_virtual_time):
            with patch("scrape_all.PROXIES", ["http://recovering-proxy:8888"]):
                total_enriched = enrich_linkedin_descriptions(cooldown_seconds=0, max_workers=2)

        self.assertEqual(total_enriched, 6)
        mock_validate_proxy.assert_called()


class TestDirectCareerValidation(unittest.TestCase):
    """
    Test suite validating is_valid_direct_career_post filtering rules.
    """

    def test_valid_job_post_accepted(self):
        """
        Verify an authentic job posting with a full description passes validation.
        """
        valid_post = {
            "title": "Senior Distributed Systems Engineer",
            "absolute_url": "https://example.com/jobs/senior-systems-engineer",
            "description_text": "We are seeking an experienced distributed systems engineer to build and operate high-throughput event processing pipelines.",
        }
        self.assertTrue(is_valid_direct_career_post(valid_post))

    def test_short_description_rejected(self):
        """
        Verify postings with descriptions shorter than 80 characters are discarded.
        """
        short_post = {
            "title": "Senior Backend Engineer",
            "absolute_url": "https://example.com/jobs/backend",
            "description_text": "We are hiring an engineer.",
        }
        self.assertFalse(is_valid_direct_career_post(short_post))

    def test_synthetic_direct_career_description_rejected(self):
        """
        Verify synthetic placeholder descriptions generated by scrapers are rejected.
        """
        synthetic_post = {
            "title": "Frontend Engineer",
            "absolute_url": "https://example.com/jobs/frontend",
            "description_text": "Direct career posting from ExampleCorp at https://example.com/jobs/frontend with full details.",
        }
        self.assertFalse(is_valid_direct_career_post(synthetic_post))

    def test_synthetic_embedded_ats_description_rejected(self):
        """
        Verify embedded ATS placeholder descriptions generated by scrapers are rejected.
        """
        synthetic_post = {
            "title": "DevOps Engineer",
            "absolute_url": "https://example.com/jobs/devops",
            "description_text": "Embedded ATS posting from ExampleCorp with additional information here.",
        }
        self.assertFalse(is_valid_direct_career_post(synthetic_post))

    def test_non_job_titles_rejected(self):
        """
        Verify generic navigation links and non-job categories are rejected.
        """
        non_job_titles = [
            "Inclusion",
            "Culture and Values",
            "University Programs",
            "Benefits",
            "Learn more",
            "Explore opportunities",
            "Our Teams",
            "Veterans",
            "Indigenous",
        ]
        for title in non_job_titles:
            post = {
                "title": title,
                "absolute_url": "https://example.com/jobs/101",
                "description_text": "We are seeking an experienced distributed systems engineer to build and operate high-throughput event processing pipelines.",
            }
            self.assertFalse(is_valid_direct_career_post(post), f"Expected title {title!r} to be rejected")

    def test_generic_title_prefixes_rejected(self):
        """
        Verify titles starting with generic phrases like 'Learn more' or 'About ' are rejected.
        """
        post = {
            "title": "Learn more about engineering roles",
            "absolute_url": "https://example.com/jobs/102",
            "description_text": "We are seeking an experienced distributed systems engineer to build and operate high-throughput event processing pipelines.",
        }
        self.assertFalse(is_valid_direct_career_post(post))

    def test_non_job_url_segments_rejected(self):
        """
        Verify URLs containing navigation, culture, and DEI path segments are discarded.
        """
        rejected_urls = [
            "https://example.com/careers/inclusion",
            "https://example.com/careers/culture",
            "https://example.com/careers/benefits",
            "https://example.com/careers/teams",
            "https://example.com/careers/students",
            "https://example.com/careers/military",
        ]
        for url in rejected_urls:
            post = {
                "title": "Senior Software Engineer",
                "absolute_url": url,
                "description_text": "We are seeking an experienced distributed systems engineer to build and operate high-throughput event processing pipelines.",
            }
            self.assertFalse(is_valid_direct_career_post(post), f"Expected URL {url!r} to be rejected")

    def test_invalid_url_scheme_rejected(self):
        """
        Verify invalid or relative URLs are rejected.
        """
        post = {
            "title": "Software Engineer",
            "absolute_url": "/jobs/software-engineer",
            "description_text": "We are seeking an experienced distributed systems engineer to build and operate high-throughput event processing pipelines.",
        }
        self.assertFalse(is_valid_direct_career_post(post))

    def test_title_length_bounds_enforced(self):
        """
        Verify titles that are too short (< 4 chars) or too long (> 120 chars) are rejected.
        """
        short_title_post = {
            "title": "Dev",
            "absolute_url": "https://example.com/jobs/dev",
            "description_text": "We are seeking an experienced distributed systems engineer to build and operate high-throughput event processing pipelines.",
        }
        self.assertFalse(is_valid_direct_career_post(short_title_post))

        long_title_post = {
            "title": "A" * 121,
            "absolute_url": "https://example.com/jobs/dev",
            "description_text": "We are seeking an experienced distributed systems engineer to build and operate high-throughput event processing pipelines.",
        }
        self.assertFalse(is_valid_direct_career_post(long_title_post))


class TestDirectCareerProcessing(unittest.TestCase):
    """
    Test suite validating process_direct_career_company ATS discovery and junk filtering.
    """

    @patch("scrape_all.register_discovered_ats_slug")
    @patch("scrape_all.stream_ingest_jobs")
    def test_process_direct_career_company_discovers_ats_slug(
        self, mock_stream_ingest, mock_register_ats_slug
    ):
        """
        Verify that URLs pointing to recognized ATS platforms trigger ATS slug registration.
        """
        mock_scraper = Mock()
        mock_job = Mock()
        mock_job.id = "https://boards.greenhouse.io/airbnb/jobs/45678"
        mock_job.title = "Software Engineer"
        mock_job.company = "Airbnb"
        mock_job.job_url = "https://boards.greenhouse.io/airbnb/jobs/45678"
        mock_job.location = Mock(country="Remote")
        mock_job.description = "We are looking for a software engineer to join our infrastructure platform team in building global services."
        mock_job.is_remote = True
        mock_job.date_posted = None

        mock_scraper.scrape_single_company.return_value = [mock_job]

        result = process_direct_career_company(
            "Airbnb", "https://airbnb.com/careers", mock_scraper, "run-ats-discovery"
        )

        mock_register_ats_slug.assert_called_with("greenhouse", "airbnb")
        self.assertEqual(result["status"], "success")

    @patch("scrape_all.register_discovered_ats_slug")
    @patch("scrape_all.stream_ingest_jobs")
    def test_process_direct_career_company_discards_junk_links(
        self, mock_stream_ingest, mock_register_ats_slug
    ):
        """
        Verify that navigation and marketing links are filtered out before ingestion.
        """
        mock_scraper = Mock()

        junk_job_inclusion = Mock()
        junk_job_inclusion.id = "https://example.com/careers/inclusion"
        junk_job_inclusion.title = "Inclusion"
        junk_job_inclusion.company = "ExampleCorp"
        junk_job_inclusion.job_url = "https://example.com/careers/inclusion"
        junk_job_inclusion.location = Mock(country="Remote")
        junk_job_inclusion.description = "Direct career posting from ExampleCorp at https://example.com/careers/inclusion"
        junk_job_inclusion.is_remote = True
        junk_job_inclusion.date_posted = None

        junk_job_culture = Mock()
        junk_job_culture.id = "https://example.com/careers/culture"
        junk_job_culture.title = "Our Culture"
        junk_job_culture.company = "ExampleCorp"
        junk_job_culture.job_url = "https://example.com/careers/culture"
        junk_job_culture.location = Mock(country="Remote")
        junk_job_culture.description = "Direct career posting from ExampleCorp at https://example.com/careers/culture"
        junk_job_culture.is_remote = True
        junk_job_culture.date_posted = None

        valid_job = Mock()
        valid_job.id = "https://example.com/jobs/cloud-architect"
        valid_job.title = "Cloud Infrastructure Architect"
        valid_job.company = "ExampleCorp"
        valid_job.job_url = "https://example.com/jobs/cloud-architect"
        valid_job.location = Mock(country="Remote")
        valid_job.description = "We are seeking a seasoned Cloud Infrastructure Architect to lead our multi-region Kubernetes deployments."
        valid_job.is_remote = True
        valid_job.date_posted = None

        mock_scraper.scrape_single_company.return_value = [
            junk_job_inclusion,
            junk_job_culture,
            valid_job,
        ]

        result = process_direct_career_company(
            "ExampleCorp", "https://example.com/careers", mock_scraper, "run-filter-junk"
        )

        self.assertEqual(result["status"], "success")
        self.assertEqual(len(result["jobs"]), 1)
        self.assertEqual(result["jobs"][0]["title"], "Cloud Infrastructure Architect")


if __name__ == "__main__":
    unittest.main()



