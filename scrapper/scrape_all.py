"""
Unified scraper orchestrator running ATS company boards, job board sources, and automated career page ATS discovery.
"""

from __future__ import annotations

import json
import os
import re
import sys
import threading
import time
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed

import requests
from bs4 import BeautifulSoup
from config import (
    DATA_DIR,
    MAX_WORKERS,
    REQUEST_TIMEOUT,
    BACKEND_API_URL,
    INGEST_API_KEY,
    USER_AGENT,
    PROXIES,
)

from jobspy import scrape_jobs
from jobspy.model import Site

DEFAULT_KEYWORDS = [
    "backend engineer",
    "software engineer",
    "golang developer",
    "python developer",
    "full stack engineer",
    "frontend engineer",
    "cloud engineer",
    "devops engineer",
    "site reliability engineer",
    "platform engineer",
    "infrastructure engineer",
    "systems engineer",
    "rust developer",
    "c++ engineer",
    "distributed systems engineer",
    "microservices engineer",
    "data engineer",
    "mlops engineer",
    "ai infrastructure engineer",
    "genai engineer",
    "kubernetes engineer",
    "aws engineer",
    "node.js developer",
    "react developer",
    "typescript developer",
    "founding engineer",
    "software engineer intern",
    "graduate software engineer",
    "new grad software engineer",
    "junior backend engineer",
    "junior devops engineer",
    "sde i",
]


def fetch_master_keywords() -> list[str]:
    """
    Fetches the latest approved master search keywords from the backend API.
    """
    try:
        response = requests.get(f"{BACKEND_API_URL}/keywords", timeout=10)
        if response.status_code == 200:
            received_keywords = response.json().get("data", [])
            if received_keywords:
                return received_keywords
    except Exception:
        pass
    return DEFAULT_KEYWORDS


KEYWORDS = fetch_master_keywords()

SINGLE_CALL_FEED_SITES = [
    Site.REMOTEOK,
    Site.WEWORKREMOTELY,
    Site.HN_HIRING,
    Site.YC_STARTUP,
    Site.THE_MUSE,
    Site.HIMALAYAS,
    Site.JOBSPRESSO,
    Site.WORKING_NOMADS,
    Site.DIRECT_CAREERS,
]

KEYWORD_SEARCHABLE_INDIA_SITES = [
    Site.INDEED,
    Site.LINKEDIN,
]

KEYWORD_SEARCHABLE_REMOTE_SITES = [
    Site.INDEED,
    Site.DICE,
    Site.AMAZON,
    Site.LINKEDIN,
]

PROXY_REQUIRED_SITES = {
    Site.LINKEDIN,
}

ATS_DETECTION_PATTERNS = [
    (re.compile(r"(?:boards|job-boards|boards\.eu)\.greenhouse\.io/([^/?#]+)"), "greenhouse"),
    (re.compile(r"jobs\.lever\.co/([^/?#]+)"), "lever"),
    (re.compile(r"jobs\.ashbyhq\.com/([^/?#]+)"), "ashby"),
    (re.compile(r"jobs\.smartrecruiters\.com/([^/?#]+)"), "smartrecruiters"),
    (re.compile(r"([a-z0-9-]+)\.wd\d+\.myworkdaysite\.com"), "workday"),
]

CAREER_PATH_PATTERNS = ["/jobs", "/careers", "/career", "/work-with-us", "/join-us", "/openings"]
CAREER_SUBDOMAIN_PATTERNS = ["careers", "jobs"]

HTTP_PROBE_SESSION = requests.Session()
HTTP_PROBE_SESSION.headers.update(
    {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    }
)
HTTP_PROBE_SESSION.max_redirects = 5


def fetch_ats_slugs() -> dict[str, list[str]]:
    """
    Fetches active ATS platform and slug configurations from the backend API.
    """
    request_headers = {
        "X-Ingest-Key": INGEST_API_KEY,
    }
    try:
        response = requests.get(
            f"{BACKEND_API_URL}/scraper/ats-slugs",
            headers=request_headers,
            timeout=15,
        )
        if response.status_code == 200:
            return response.json().get("data", {})
    except Exception:
        pass
    return {}


def register_discovered_ats_slug(platform_name: str, company_slug: str) -> None:
    """
    Registers a newly discovered ATS company slug with the backend API.
    """
    if not company_slug:
        return
    request_headers = {
        "X-Ingest-Key": INGEST_API_KEY,
        "Content-Type": "application/json",
    }
    request_payload = {
        "platform": platform_name,
        "slug": company_slug.strip().lower(),
    }
    try:
        requests.post(
            f"{BACKEND_API_URL}/scraper/register-ats-slug",
            json=request_payload,
            headers=request_headers,
            timeout=10,
        )
    except Exception:
        pass


def generate_career_page_candidates(company_name: str, company_domain: str = "") -> list[str]:
    """
    Generates candidate career URLs from a company name and optional verified domain.
    """
    candidates = []

    if company_domain:
        cleaned_domain = company_domain.strip().lower().replace("http://", "").replace("https://", "").rstrip("/")
        if cleaned_domain:
            for path in CAREER_PATH_PATTERNS:
                candidates.append(f"https://{cleaned_domain}{path}")
            for subdomain in CAREER_SUBDOMAIN_PATTERNS:
                candidates.append(f"https://{subdomain}.{cleaned_domain}")

    sanitized = re.sub(
        r"\b(inc\.?|llc\.?|ltd\.?|corp\.?|co\.?|plc\.?|technologies|solutions|services|group)\b",
        "",
        company_name,
        flags=re.IGNORECASE,
    )
    url_slug = re.sub(r"[^a-z0-9]", "", sanitized.lower().strip())
    if url_slug and len(url_slug) >= 3:
        for path in CAREER_PATH_PATTERNS:
            candidates.append(f"https://{url_slug}.com{path}")
        for subdomain in CAREER_SUBDOMAIN_PATTERNS:
            candidates.append(f"https://{subdomain}.{url_slug}.com")

    seen = set()
    unique_candidates = []
    for candidate in candidates:
        if candidate not in seen:
            seen.add(candidate)
            unique_candidates.append(candidate)

    return unique_candidates


def probe_single_career_page(company_name: str, company_domain: str = "") -> tuple[str, str] | None:
    """
    Probes candidate URLs for a company and extracts ATS configuration details if matched.
    """
    for candidate_url in generate_career_page_candidates(company_name, company_domain):
        try:
            response = HTTP_PROBE_SESSION.get(candidate_url, timeout=8, allow_redirects=True)
            for pattern, platform_name in ATS_DETECTION_PATTERNS:
                if pattern.search(response.url):
                    return platform_name, pattern.search(response.url).group(1).lower().strip("/")
                if response.status_code == 200 and pattern.search(response.text):
                    return platform_name, pattern.search(response.text).group(1).lower().strip("/")
        except Exception:
            continue
    return None


def probe_unmapped_companies(concurrency: int = 20, max_probes_per_run: int = 500) -> int:
    """
    Probes a batch of unmapped companies from the backend and registers newly found ATS integrations.
    """
    request_headers = {
        "X-Ingest-Key": INGEST_API_KEY,
    }
    try:
        api_response = requests.get(
            f"{BACKEND_API_URL}/scraper/companies",
            headers=request_headers,
            timeout=15,
        )
        if api_response.status_code != 200:
            return 0
        raw_company_data = api_response.json().get("data", [])
    except Exception:
        return 0

    probed_batch = raw_company_data[:max_probes_per_run]
    discovered_count = 0

    with ThreadPoolExecutor(max_workers=concurrency) as probe_executor:
        future_map = {}
        for target in probed_batch:
            if isinstance(target, dict):
                company_name = target.get("name", "")
                company_domain = target.get("domain", "")
            else:
                company_name = str(target)
                company_domain = ""

            if company_name:
                future = probe_executor.submit(probe_single_career_page, company_name, company_domain)
                future_map[future] = company_name

        for future in as_completed(future_map):
            try:
                result = future.result()
                if result:
                    platform_name, company_slug = result
                    register_discovered_ats_slug(platform_name, company_slug)
                    discovered_count += 1
            except Exception:
                pass

    return discovered_count


def start_run() -> str | None:
    """
    Registers a new scraper run with the backend telemetry system.
    """
    request_headers = {
        "X-Ingest-Key": INGEST_API_KEY,
        "Content-Type": "application/json",
    }
    try:
        target_endpoint = f"{BACKEND_API_URL}/scraper/start"
        response = requests.post(target_endpoint, headers=request_headers, timeout=20)
        if response.status_code == 200:
            return response.json().get("run_id")
    except Exception:
        pass
    return None


def finish_run(run_id: str, run_status: str, error_message: str | None = None) -> None:
    """
    Notifies the backend telemetry system that the scraper run has concluded.
    """
    request_headers = {
        "X-Ingest-Key": INGEST_API_KEY,
        "Content-Type": "application/json",
    }
    request_payload = {
        "run_id": run_id,
        "status": run_status,
        "error_message": error_message,
    }
    try:
        target_endpoint = f"{BACKEND_API_URL}/scraper/finish"
        requests.post(target_endpoint, json=request_payload, headers=request_headers, timeout=20)
    except Exception:
        pass


def save_json(data_payload, target_file_path: Path) -> None:
    """
    Serializes data to a local JSON file without whitespace indentation for high throughput.
    """
    with open(target_file_path, "w", encoding="utf-8") as file_stream:
        json.dump(data_payload, file_stream, ensure_ascii=False)


def ensure_dir(directory_path: Path) -> None:
    """
    Ensures that the specified directory exists on the filesystem.
    """
    directory_path.mkdir(parents=True, exist_ok=True)


def sanitize_company_name(raw_company_name) -> str:
    """
    Sanitizes raw company names into safe strings suitable for indexing.
    """
    if not raw_company_name or not isinstance(raw_company_name, (str, bytes)):
        return "Unknown"

    company_string = str(raw_company_name).strip()
    if not company_string or company_string.lower() == "nan":
        return "Unknown"

    parsed_html_text = BeautifulSoup(company_string, "html.parser").get_text()
    if "http://" in parsed_html_text or "https://" in parsed_html_text:
        parsed_html_text = parsed_html_text.split("http")[0].strip()

    for invalid_character in ["/", "\\", ":", "*", "?", '"', "<", ">", "|"]:
        parsed_html_text = parsed_html_text.replace(invalid_character, " ")

    cleaned_company_name = parsed_html_text.strip()[:50].strip()
    return cleaned_company_name if cleaned_company_name else "Unknown"


def extract_ats_slug(job_url: str) -> tuple[str, str] | None:
    """
    Extracts ATS platform name and company slug from a job posting URL.
    """
    if not job_url:
        return None

    normalized_url = job_url.lower()

    if "boards.greenhouse.io/" in normalized_url or "boards-api.greenhouse.io/" in normalized_url:
        url_components = [component for component in job_url.split("/") if component]
        for index, component in enumerate(url_components):
            if "greenhouse.io" in component and index + 1 < len(url_components):
                return "greenhouse", url_components[index + 1].split("?")[0].split("#")[0].strip().lower()

    if "jobs.lever.co/" in normalized_url:
        url_components = [component for component in job_url.split("/") if component]
        for index, component in enumerate(url_components):
            if "lever.co" in component and index + 1 < len(url_components):
                return "lever", url_components[index + 1].split("?")[0].split("#")[0].strip().lower()

    if "jobs.ashbyhq.com/" in normalized_url:
        url_components = [component for component in job_url.split("/") if component]
        for index, component in enumerate(url_components):
            if "ashbyhq.com" in component and index + 1 < len(url_components):
                return "ashby", url_components[index + 1].split("?")[0].split("#")[0].strip().lower()

    if "jobs.smartrecruiters.com/" in normalized_url:
        url_components = [component for component in job_url.split("/") if component]
        for index, component in enumerate(url_components):
            if "smartrecruiters.com" in component and index + 1 < len(url_components):
                return "smartrecruiters", url_components[index + 1].split("?")[0].split("#")[0].strip().lower()

    if ".myworkdaysite.com/" in normalized_url:
        host_domain = job_url.split("://")[-1].split("/")[0]
        if "myworkdaysite.com" in host_domain:
            return "workday", host_domain.split(".")[0].strip().lower()

    return None


INDIAN_LOCATIONS = [
    "india", "bengaluru", "bangalore", "hyderabad", "pune", "mumbai", "delhi", "noida",
    "gurgaon", "gurugram", "chennai", "kolkata", "ahmedabad", "indore", "kochi", "trivandrum",
    "chandigarh", "jaipur", "coimbatore", "cochin", "thiruvananthapuram", "karnataka",
    "telangana", "maharashtra", "tamil nadu", "haryana", "uttar pradesh", "kerala",
    "west bengal", "gujarat", "punjab", "rajasthan"
]

REMOTE_INDICATORS = [
    "remote", "wfh", "work from home", "work-from-home", "global", "worldwide", "anywhere",
    "flexible", "distributed", "virtual", "telecommute", "home-based", "home based", "everywhere"
]

EXPLICIT_NON_INDIA_RESTRICTIONS = [
    "us citizenship required", "us citizen required", "us resident only", "must reside in us",
    "must reside in the us", "must reside in the united states", "must reside in uk",
    "must reside in canada", "us security clearance", "active secret clearance",
    "top secret clearance", "us time zone required", "us timezone required", "est timezone only",
    "pst timezone only", "us/canada only", "canada only", "us remote only", "u.s. remote only",
    "uk remote only", "eu remote only", "europe remote only"
]


def is_location_in_scope(location_string: str, is_remote_position: bool = False) -> bool:
    """
    Evaluates whether a job position is in scope for Indian candidates or global remote seekers.
    """
    if not location_string:
        return True

    if is_remote_position:
        return True

    normalized_location = location_string.lower()

    if any(restriction in normalized_location for restriction in EXPLICIT_NON_INDIA_RESTRICTIONS):
        return False

    for indian_region in INDIAN_LOCATIONS:
        if indian_region in normalized_location:
            return True

    if any(indicator in normalized_location for indicator in REMOTE_INDICATORS):
        return True

    return False


def normalize_job_post(raw_job_record, source_identifier: str, company_name: str | None = None) -> dict:
    """
    Normalizes a job listing from diverse source objects into a uniform dictionary representation.
    """
    if isinstance(raw_job_record, dict):
        job_identifier = raw_job_record.get("job_id") or raw_job_record.get("id") or ""
        job_title = raw_job_record.get("title") or ""
        extracted_company = company_name or raw_job_record.get("company") or raw_job_record.get("company_name") or ""
        job_url = raw_job_record.get("absolute_url") or raw_job_record.get("job_url") or ""
        job_location = raw_job_record.get("location") or ""
        job_description = raw_job_record.get("description_text") or raw_job_record.get("description") or ""
        updated_timestamp = raw_job_record.get("updated_at") or raw_job_record.get("date_posted") or ""
        job_departments = raw_job_record.get("departments") or []
        job_offices = raw_job_record.get("offices") or []
    else:
        job_identifier = getattr(raw_job_record, "id", "") or getattr(raw_job_record, "job_id", "") or ""
        job_title = getattr(raw_job_record, "title", "")
        extracted_company = company_name or getattr(raw_job_record, "company_name", "") or getattr(raw_job_record, "company", "") or ""
        job_url = getattr(raw_job_record, "job_url", "") or getattr(raw_job_record, "absolute_url", "") or ""

        location_attribute = getattr(raw_job_record, "location", None)
        if location_attribute and hasattr(location_attribute, "display_location"):
            job_location = location_attribute.display_location()
        else:
            job_location = str(location_attribute) if location_attribute else ""

        job_description = getattr(raw_job_record, "description", "") or getattr(raw_job_record, "description_text", "") or ""

        date_posted_attribute = getattr(raw_job_record, "date_posted", None)
        if date_posted_attribute:
            if hasattr(date_posted_attribute, "isoformat"):
                updated_timestamp = date_posted_attribute.isoformat() + "T00:00:00Z"
            else:
                updated_timestamp = str(date_posted_attribute)
        else:
            updated_timestamp = getattr(raw_job_record, "updated_at", "") or ""

        job_departments = getattr(raw_job_record, "departments", [])
        job_offices = getattr(raw_job_record, "offices", [])

    return {
        "job_id": str(job_identifier),
        "title": job_title,
        "updated_at": updated_timestamp,
        "absolute_url": job_url,
        "location": job_location,
        "departments": job_departments,
        "offices": job_offices,
        "description_text": job_description,
        "company": sanitize_company_name(extracted_company),
        "source": source_identifier,
    }


def deduplicate_jobs(jobs_collection: list[dict]) -> list[dict]:
    """
    Deduplicates a collection of normalized job records based on company, title, and location.
    """
    seen_identifiers = set()
    unique_job_records = []
    for job_record in jobs_collection:
        deduplication_key = (
            (job_record.get("company") or "").strip().lower(),
            (job_record.get("title") or "").strip().lower(),
            (job_record.get("location") or "").strip().lower(),
        )
        if deduplication_key not in seen_identifiers:
            seen_identifiers.add(deduplication_key)
            unique_job_records.append(job_record)
    return unique_job_records


def process_company(company_slug: str, platform_name: str, run_id: str | None = None) -> dict:
    """
    Scrapes job listings for a designated company slug on a specific ATS platform.
    """
    try:
        scraped_dataframe = scrape_jobs(
            site_name=[platform_name],
            search_term=company_slug,
            results_wanted=100,
        )
        extracted_jobs = []
        for job_row in scraped_dataframe.itertuples():
            normalized_post = normalize_job_post(job_row, platform_name, company_slug)
            is_remote_flag = getattr(job_row, "is_remote", False) or "remote" in normalized_post["location"].lower()
            if is_location_in_scope(normalized_post["location"], is_remote_flag):
                extracted_jobs.append(normalized_post)
        execution_status = "success"
    except Exception:
        extracted_jobs = []
        execution_status = "failed"

    return {
        "company": company_slug,
        "platform": platform_name,
        "jobs": extracted_jobs,
        "status": execution_status,
    }


def run_orchestration() -> dict:
    """
    Executes the entire job scraping, career page discovery, and ingestion pipeline concurrently.
    """
    ensure_dir(DATA_DIR)

    active_ats_platform_slugs = fetch_ats_slugs()
    run_identifier = start_run()
    aggregated_raw_jobs = []
    run_manifest = []
    source_statistics = {}
    source_stats_lock = threading.Lock()

    def record_source_telemetry(source_key: str, jobs_count: int, elapsed_seconds: float, error_detail: str | None = None) -> None:
        with source_stats_lock:
            source_statistics[source_key] = {
                "jobs_found": jobs_count,
                "duration_seconds": round(elapsed_seconds, 2),
                "error": error_detail,
            }

    try:
        with ThreadPoolExecutor(max_workers=MAX_WORKERS) as primary_executor:
            discovery_future = primary_executor.submit(probe_unmapped_companies, MAX_WORKERS, 500)

            company_futures = []
            for platform_name, company_slugs in active_ats_platform_slugs.items():
                for company_slug in company_slugs:
                    company_futures.append(
                        primary_executor.submit(process_company, company_slug, platform_name, run_identifier)
                    )

            board_raw_jobs_lock = threading.Lock()

            def scrape_board_site_keyword(
                target_site: Site,
                search_query: str,
                target_location: str | None,
                is_remote_search: bool,
            ) -> None:
                """
                Scrapes a single job board for a designated search query and location with 5-minute timeout.
                """
                telemetry_key = f"{target_site.value}:{search_query}:{target_location or 'remote'}"
                start_timestamp = time.time()
                sub_executor = ThreadPoolExecutor(max_workers=1)
                scraped_dataframe = None
                caught_error = None

                try:
                    scraping_arguments = {
                        "site_name": [target_site],
                        "search_term": search_query,
                        "results_wanted": 200,
                        "hours_old": 24,
                    }
                    if target_site == Site.DIRECT_CAREERS:
                        scraping_arguments["results_wanted"] = 5000
                        scraping_arguments.pop("hours_old", None)

                    if target_location:
                        scraping_arguments["location"] = target_location
                    if is_remote_search:
                        scraping_arguments["is_remote"] = True
                    if target_site == Site.INDEED and target_location == "India":
                        scraping_arguments["country_indeed"] = "india"
                    if target_site in PROXY_REQUIRED_SITES and PROXIES:
                        scraping_arguments["proxies"] = PROXIES

                    future_result = sub_executor.submit(scrape_jobs, **scraping_arguments)
                    scraped_dataframe = future_result.result(timeout=REQUEST_TIMEOUT)
                    sub_executor.shutdown(wait=False)
                except Exception as execution_err:
                    caught_error = str(execution_err)
                    sub_executor.shutdown(wait=False, cancel_futures=True)

                elapsed_time = time.time() - start_timestamp

                if scraped_dataframe is None or scraped_dataframe.empty:
                    record_source_telemetry(telemetry_key, 0, elapsed_time, caught_error)
                    return

                parsed_posts = []
                for job_row in scraped_dataframe.itertuples():
                    site_source = getattr(job_row, "site", target_site.value)
                    normalized_post = normalize_job_post(job_row, site_source)
                    is_remote_flag = (
                        is_remote_search
                        or getattr(job_row, "is_remote", False)
                        or "remote" in normalized_post["location"].lower()
                    )
                    if is_location_in_scope(normalized_post["location"], is_remote_flag):
                        parsed_posts.append(normalized_post)

                record_source_telemetry(telemetry_key, len(parsed_posts), elapsed_time, caught_error)

                if parsed_posts:
                    with board_raw_jobs_lock:
                        aggregated_raw_jobs.extend(parsed_posts)

            board_futures = []

            keyword_feed_sites = [site for site in SINGLE_CALL_FEED_SITES if site != Site.DIRECT_CAREERS]
            for feed_site in keyword_feed_sites:
                board_futures.append(
                    primary_executor.submit(
                        scrape_board_site_keyword, feed_site, "software engineer", None, True
                    )
                )

            board_futures.append(
                primary_executor.submit(
                    scrape_board_site_keyword, Site.DIRECT_CAREERS, None, None, True
                )
            )

            for search_term in KEYWORDS:
                for india_site in KEYWORD_SEARCHABLE_INDIA_SITES:
                    board_futures.append(
                        primary_executor.submit(
                            scrape_board_site_keyword, india_site, search_term, "India", False
                        )
                    )
                for remote_site in KEYWORD_SEARCHABLE_REMOTE_SITES:
                    board_futures.append(
                        primary_executor.submit(
                            scrape_board_site_keyword, remote_site, search_term, None, True
                        )
                    )

            for completed_future in as_completed(company_futures):
                execution_result = completed_future.result()
                run_manifest.append(
                    {
                        "company": execution_result["company"],
                        "platform": execution_result["platform"],
                        "job_count": len(execution_result.get("jobs", [])),
                        "status": execution_result["status"],
                    }
                )
                if execution_result["status"] == "success":
                    aggregated_raw_jobs.extend(execution_result["jobs"])

            for completed_future in as_completed(board_futures):
                completed_future.result()

            try:
                discovered_count = discovery_future.result()
                if discovered_count > 0:
                    print(f"[Discovery] Successfully identified and registered {discovered_count} new ATS boards during run.")
            except Exception:
                pass

        deduplicated_job_records = deduplicate_jobs(aggregated_raw_jobs)

        for job_record in deduplicated_job_records:
            discovered_ats_details = extract_ats_slug(job_record["absolute_url"])
            if discovered_ats_details:
                register_discovered_ats_slug(discovered_ats_details[0], discovered_ats_details[1])

        save_json(deduplicated_job_records, DATA_DIR / "raw_jobs.json")
        save_json(source_statistics, DATA_DIR / "source_stats.json")
        print(f"[Scraper] Saved {len(deduplicated_job_records)} raw scraped jobs to raw_jobs.json", flush=True)
        print(f"[Scraper] Recorded telemetry across {len(source_statistics)} source search combinations.", flush=True)

        if not run_identifier:
            run_identifier = start_run()

        if run_identifier:
            ingest_request_headers = {
                "X-Ingest-Key": INGEST_API_KEY,
                "Content-Type": "application/json",
            }
            ingest_endpoint_url = f"{BACKEND_API_URL}/scraper/ingest-raw"
            batch_chunk_size = 500
            total_jobs_added = 0
            for chunk_offset in range(0, len(deduplicated_job_records), batch_chunk_size):
                job_batch_chunk = deduplicated_job_records[chunk_offset:chunk_offset + batch_chunk_size]
                try:
                    ingest_response = requests.post(
                        ingest_endpoint_url,
                        json={"run_id": run_identifier, "jobs": job_batch_chunk},
                        headers=ingest_request_headers,
                        timeout=REQUEST_TIMEOUT,
                    )
                    if ingest_response.status_code == 200:
                        added_count = ingest_response.json().get("jobs_added", 0)
                        total_jobs_added += added_count
                        print(
                            f"[Scraper] Ingested batch {chunk_offset // batch_chunk_size + 1}/{(len(deduplicated_job_records) + batch_chunk_size - 1) // batch_chunk_size} ({added_count} jobs)",
                            flush=True,
                        )
                    else:
                        print(
                            f"[Scraper] Batch {chunk_offset // batch_chunk_size + 1} returned {ingest_response.status_code}: {ingest_response.text[:200]}",
                            flush=True,
                        )
                except Exception as batch_error:
                    print(
                        f"[Scraper] Failed to POST chunk {chunk_offset // batch_chunk_size + 1}: {batch_error}",
                        flush=True,
                    )
            print(f"[Scraper] Backend raw ingestion complete. Total jobs added: {total_jobs_added}", flush=True)

        save_json(run_manifest, DATA_DIR / "manifest.json")

        if run_identifier:
            finish_run(run_identifier, "success")

        return {"status": "success", "manifest": run_manifest, "source_stats": source_statistics}

    except Exception as execution_exception:
        if run_identifier:
            finish_run(run_identifier, "failed", str(execution_exception))
        raise execution_exception


if __name__ == "__main__":
    if "--test" in sys.argv:
        KEYWORDS[:] = ["golang developer", "backend engineer"]
        print(f"[Scraper] Running in TEST mode. Keywords reduced to: {KEYWORDS}", flush=True)
    run_orchestration()
