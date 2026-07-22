"""
End-to-end integration tests for the scrape → normalize → ingest pipeline.

These tests verify the full flow from job source clients through scrape_all normalisation
functions to the POST payload that would be sent to the Go backend, using a mock HTTP
server in place of the real Go API.
"""

import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from unittest.mock import Mock, patch

import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scrape_all import (
    deduplicate_jobs,
    extract_ats_slug,
    is_location_in_scope,
    normalize_job_post,
    sanitize_company_name,
)


class CapturedRequest:
    """Holds a single captured HTTP request body and path for later assertion."""

    def __init__(self, path: str, body: dict):
        self.path = path
        self.body = body


class MockBackendRequestHandler(BaseHTTPRequestHandler):
    """
    Minimal HTTP handler that records inbound POST requests and responds
    with the canned JSON payloads the scraper expects.
    """

    captured_requests: list[CapturedRequest] = []
    start_run_response = {"run_id": "test-run-001"}
    ingest_response = {"jobs_added": 0, "message": "ok"}

    def log_message(self, format, *args):
        pass

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length)) if length else {}
        MockBackendRequestHandler.captured_requests.append(CapturedRequest(self.path, body))

        if self.path == "/api/scraper/start":
            self._respond(200, MockBackendRequestHandler.start_run_response)
        elif self.path == "/api/scraper/ingest-raw":
            received_count = len(body.get("jobs", []))
            self._respond(200, {"jobs_added": received_count, "message": "ok"})
        elif self.path == "/api/scraper/finish":
            self._respond(200, {"message": "ok"})
        else:
            self._respond(404, {"error": "not found"})

    def _respond(self, status_code: int, payload: dict):
        encoded = json.dumps(payload).encode()
        self.send_response(status_code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)


class TestNormalizeJobPostDictInput(unittest.TestCase):
    """Unit tests for normalize_job_post when the input is a plain dict (ATS client output)."""

    def test_all_fields_present_returns_correct_schema(self):
        """
        Verify that a fully-populated dict job is normalised into the expected flat schema.
        """
        raw = {
            "job_id": "gh-9999",
            "title": "Senior Backend Engineer",
            "company": "Notion",
            "absolute_url": "https://boards.greenhouse.io/notion/jobs/9999",
            "location": "Bengaluru, India",
            "departments": ["Engineering"],
            "offices": ["Bengaluru HQ"],
            "description_text": "Build core infra with python and golang.",
            "updated_at": "2026-07-19T00:00:00Z",
        }
        result = normalize_job_post(raw, "greenhouse")

        self.assertEqual(result["job_id"], "gh-9999")
        self.assertEqual(result["title"], "Senior Backend Engineer")
        self.assertEqual(result["company"], "Notion")
        self.assertEqual(result["absolute_url"], "https://boards.greenhouse.io/notion/jobs/9999")
        self.assertEqual(result["location"], "Bengaluru, India")
        self.assertEqual(result["source"], "greenhouse")
        self.assertEqual(result["departments"], ["Engineering"])
        self.assertEqual(result["offices"], ["Bengaluru HQ"])

    def test_missing_optional_fields_default_to_empty(self):
        """
        Verify that missing optional fields default to empty strings/lists rather than raising.
        """
        raw = {
            "title": "Intern",
            "absolute_url": "https://example.com/job/1",
        }
        result = normalize_job_post(raw, "lever")

        self.assertEqual(result["job_id"], "")
        self.assertEqual(result["location"], "")
        self.assertEqual(result["description_text"], "")
        self.assertEqual(result["departments"], [])

    def test_company_name_override_takes_precedence_over_dict_field(self):
        """
        Verify that an explicit company_name argument overrides the company value in the dict.
        """
        raw = {"title": "Developer", "company": "Should Be Ignored", "absolute_url": "https://x.com/1"}
        result = normalize_job_post(raw, "ashby", company_name="ActualCompany")

        self.assertEqual(result["company"], "ActualCompany")


class TestSanitizeCompanyName(unittest.TestCase):
    """Unit tests for sanitize_company_name edge cases."""

    def test_none_input_returns_unknown(self):
        """Verify None input is safely coerced to 'Unknown'."""
        self.assertEqual(sanitize_company_name(None), "Unknown")

    def test_nan_string_returns_unknown(self):
        """Verify the string 'nan' (from pandas floats) is coerced to 'Unknown'."""
        self.assertEqual(sanitize_company_name("nan"), "Unknown")

    def test_html_stripped_from_name(self):
        """Verify HTML tags in company names are stripped cleanly."""
        self.assertEqual(sanitize_company_name("<b>Acme Corp</b>"), "Acme Corp")

    def test_url_suffix_stripped(self):
        """Verify trailing URL components in company names are removed."""
        result = sanitize_company_name("Acme Corp https://acme.com/careers")
        self.assertNotIn("https://", result)
        self.assertIn("Acme Corp", result)

    def test_long_name_truncated_to_50_chars(self):
        """Verify company names longer than 50 characters are truncated."""
        long_name = "A" * 80
        result = sanitize_company_name(long_name)
        self.assertLessEqual(len(result), 50)

    def test_filesystem_unsafe_chars_replaced(self):
        """Verify filesystem-unsafe characters are replaced with spaces."""
        result = sanitize_company_name("Acme/Corp:Ltd*Co")
        self.assertNotIn("/", result)
        self.assertNotIn(":", result)
        self.assertNotIn("*", result)

    def test_empty_string_returns_unknown(self):
        """Verify empty string input is coerced to 'Unknown'."""
        self.assertEqual(sanitize_company_name(""), "Unknown")


class TestExtractAtsSlug(unittest.TestCase):
    """Unit tests for extract_ats_slug URL pattern matching."""

    def test_greenhouse_boards_url_extracted(self):
        """Verify Greenhouse board URL yields correct platform and slug."""
        platform, slug = extract_ats_slug("https://boards.greenhouse.io/airbnb/jobs/12345")
        self.assertEqual(platform, "greenhouse")
        self.assertEqual(slug, "airbnb")

    def test_lever_url_extracted(self):
        """Verify Lever job URL yields correct platform and slug."""
        platform, slug = extract_ats_slug("https://jobs.lever.co/stripe/abc123")
        self.assertEqual(platform, "lever")
        self.assertEqual(slug, "stripe")

    def test_ashby_url_extracted(self):
        """Verify Ashby job URL yields correct platform and slug."""
        platform, slug = extract_ats_slug("https://jobs.ashbyhq.com/notion/xyz")
        self.assertEqual(platform, "ashby")
        self.assertEqual(slug, "notion")

    def test_smartrecruiters_url_extracted(self):
        """Verify SmartRecruiters job URL yields correct platform and slug."""
        platform, slug = extract_ats_slug("https://jobs.smartrecruiters.com/visa/123")
        self.assertEqual(platform, "smartrecruiters")
        self.assertEqual(slug, "visa")

    def test_workday_url_extracted(self):
        """Verify Workday company subdomain yields correct platform and slug."""
        platform, slug = extract_ats_slug("https://netflix.wd5.myworkdaysite.com/en-US/recruiting/netflix/job")
        self.assertEqual(platform, "workday")
        self.assertEqual(slug, "netflix")

    def test_unrecognised_url_returns_none(self):
        """Verify that unrecognised job board URLs return None."""
        self.assertIsNone(extract_ats_slug("https://careers.example.com/jobs/1"))

    def test_empty_string_returns_none(self):
        """Verify empty URL input returns None."""
        self.assertIsNone(extract_ats_slug(""))


class TestDeduplicateJobs(unittest.TestCase):
    """Unit tests for the deduplication logic."""

    def test_identical_company_title_location_deduped(self):
        """
        Verify that two jobs with the same (company, title, location) triple are collapsed
        into one, retaining the first occurrence.
        """
        jobs = [
            {"title": "Backend Engineer", "company": "Stripe", "location": "Remote", "absolute_url": "https://a.com"},
            {"title": "Backend Engineer", "company": "Stripe", "location": "Remote", "absolute_url": "https://b.com"},
        ]
        result = deduplicate_jobs(jobs)
        self.assertEqual(len(result), 1)
        self.assertEqual(result[0]["absolute_url"], "https://a.com")

    def test_case_insensitive_deduplication(self):
        """Verify deduplication is case-insensitive for all three key fields."""
        jobs = [
            {"title": "Backend Engineer", "company": "stripe", "location": "remote", "absolute_url": "https://a.com"},
            {"title": "BACKEND ENGINEER", "company": "Stripe", "location": "Remote", "absolute_url": "https://b.com"},
        ]
        result = deduplicate_jobs(jobs)
        self.assertEqual(len(result), 1)

    def test_different_locations_not_deduped(self):
        """Verify that identical title+company but different locations are kept as separate jobs."""
        jobs = [
            {"title": "Engineer", "company": "Notion", "location": "Bengaluru", "absolute_url": "https://a.com"},
            {"title": "Engineer", "company": "Notion", "location": "Remote", "absolute_url": "https://b.com"},
        ]
        result = deduplicate_jobs(jobs)
        self.assertEqual(len(result), 2)

    def test_empty_list_returns_empty_list(self):
        """Verify that an empty jobs list returns an empty list."""
        self.assertEqual(deduplicate_jobs([]), [])


class TestIngestPipelineEndToEnd(unittest.TestCase):
    """
    End-to-end integration tests that spin up a mock HTTP server and verify the
    scraper's start → ingest-raw → finish HTTP flow produces correct payloads.
    """

    @classmethod
    def setUpClass(cls):
        MockBackendRequestHandler.captured_requests = []
        cls.server = HTTPServer(("127.0.0.1", 0), MockBackendRequestHandler)
        cls.port = cls.server.server_address[1]
        cls.server_thread = threading.Thread(target=cls.server.serve_forever)
        cls.server_thread.daemon = True
        cls.server_thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def setUp(self):
        MockBackendRequestHandler.captured_requests.clear()

    def _make_sample_jobs(self, count: int) -> list[dict]:
        return [
            {
                "job_id": f"job-{i}",
                "title": f"Backend Engineer {i}",
                "company": "Acme Corp",
                "absolute_url": f"https://boards.greenhouse.io/acme/jobs/{i}",
                "location": "Bengaluru, India",
                "departments": ["Engineering"],
                "offices": ["BLR"],
                "description_text": f"Job description for role {i}",
                "updated_at": "2026-07-19T00:00:00Z",
                "source": "greenhouse",
            }
            for i in range(count)
        ]

    def test_start_run_sends_correct_headers(self):
        """
        Verify that start_run posts to /api/scraper/start with the X-Ingest-Key header
        and receives a run_id back.
        """
        import requests as req_lib

        url = f"http://127.0.0.1:{self.port}/api/scraper/start"
        response = req_lib.post(url, headers={"X-Ingest-Key": "dev-ingest-key-12345", "Content-Type": "application/json"}, timeout=5)

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["run_id"], "test-run-001")
        captured = MockBackendRequestHandler.captured_requests
        self.assertEqual(len(captured), 1)
        self.assertEqual(captured[0].path, "/api/scraper/start")

    def test_ingest_raw_payload_structure(self):
        """
        Verify that posting a batch of normalised jobs to /api/scraper/ingest-raw sends
        correctly structured JSON with run_id and jobs array.
        """
        import requests as req_lib

        sample_jobs = self._make_sample_jobs(3)
        payload = {"run_id": "test-run-001", "jobs": sample_jobs}
        url = f"http://127.0.0.1:{self.port}/api/scraper/ingest-raw"
        response = req_lib.post(url, json=payload, headers={"X-Ingest-Key": "dev-key", "Content-Type": "application/json"}, timeout=5)

        self.assertEqual(response.status_code, 200)
        resp_json = response.json()
        self.assertEqual(resp_json["jobs_added"], 3)

        captured = MockBackendRequestHandler.captured_requests
        self.assertEqual(len(captured), 1)
        sent_body = captured[0].body
        self.assertEqual(sent_body["run_id"], "test-run-001")
        self.assertEqual(len(sent_body["jobs"]), 3)
        first_job = sent_body["jobs"][0]
        self.assertIn("title", first_job)
        self.assertIn("absolute_url", first_job)
        self.assertIn("location", first_job)

    def test_chunked_ingest_sends_multiple_batches(self):
        """
        Verify that a jobs list larger than the chunk size results in multiple POST requests
        to /api/scraper/ingest-raw, each carrying at most chunk_size jobs.
        """
        import requests as req_lib

        chunk_size = 3
        total_jobs = self._make_sample_jobs(7)
        url = f"http://127.0.0.1:{self.port}/api/scraper/ingest-raw"
        headers = {"X-Ingest-Key": "key", "Content-Type": "application/json"}
        total_added = 0

        for batch_start in range(0, len(total_jobs), chunk_size):
            chunk = total_jobs[batch_start : batch_start + chunk_size]
            resp = req_lib.post(url, json={"run_id": "run-xyz", "jobs": chunk}, headers=headers, timeout=5)
            self.assertEqual(resp.status_code, 200)
            total_added += resp.json()["jobs_added"]

        self.assertEqual(total_added, 7)
        self.assertEqual(len(MockBackendRequestHandler.captured_requests), 3)
        batch_sizes = [len(r.body["jobs"]) for r in MockBackendRequestHandler.captured_requests]
        self.assertEqual(batch_sizes, [3, 3, 1])

    def test_finish_run_success_sends_correct_status(self):
        """
        Verify that the finish_run POST sends status='success' and a run_id to
        /api/scraper/finish.
        """
        import requests as req_lib

        url = f"http://127.0.0.1:{self.port}/api/scraper/finish"
        payload = {"run_id": "test-run-001", "status": "success", "error_message": None}
        response = req_lib.post(url, json=payload, headers={"X-Ingest-Key": "key", "Content-Type": "application/json"}, timeout=5)

        self.assertEqual(response.status_code, 200)
        captured = MockBackendRequestHandler.captured_requests
        self.assertEqual(captured[0].path, "/api/scraper/finish")
        self.assertEqual(captured[0].body["status"], "success")
        self.assertEqual(captured[0].body["run_id"], "test-run-001")

    @patch("scrape_all.BACKEND_API_URL", "")
    @patch("scrape_all.start_run")
    def test_orchestration_skips_ingest_when_no_run_id(self, mock_start_run):
        """
        Verify that when start_run returns None (backend unavailable), the orchestrator
        skips the ingest step rather than crashing.
        """
        mock_start_run.return_value = None

        with patch("scrape_all.load_yaml_config", return_value={}), \
             patch("scrape_all.scrape_jobs") as mock_scrape, \
             patch("scrape_all.save_json"), \
             patch("scrape_all.finish_run") as mock_finish:

            mock_df = Mock()
            mock_df.empty = True
            mock_scrape.return_value = mock_df

            from scrape_all import run_orchestration
            result = run_orchestration()

        self.assertEqual(result["status"], "success")
        mock_finish.assert_not_called()

    def test_normalised_jobs_are_location_filtered_before_ingest(self):
        """
        Verify that out-of-scope jobs (e.g. US-only onsite) are excluded from the
        normalised job list before it would be sent to the backend.
        """
        in_scope_jobs = [
            {"title": "Engineer", "company": "A", "location": "Bengaluru", "absolute_url": "https://a.com/1", "source": "greenhouse"},
            {"title": "Engineer", "company": "B", "location": "Remote", "absolute_url": "https://b.com/1", "source": "lever"},
        ]
        out_of_scope_jobs = [
            {"title": "Engineer", "company": "C", "location": "San Francisco, CA", "absolute_url": "https://c.com/1", "source": "linkedin"},
            {"title": "Engineer", "company": "D", "location": "US Remote Only", "absolute_url": "https://d.com/1", "source": "indeed"},
        ]
        all_jobs = in_scope_jobs + out_of_scope_jobs

        filtered = [
            job for job in all_jobs
            if is_location_in_scope(job["location"])
        ]

        self.assertEqual(len(filtered), 2)
        urls = {j["absolute_url"] for j in filtered}
        self.assertIn("https://a.com/1", urls)
        self.assertIn("https://b.com/1", urls)
        self.assertNotIn("https://c.com/1", urls)
        self.assertNotIn("https://d.com/1", urls)


if __name__ == "__main__":
    unittest.main()
