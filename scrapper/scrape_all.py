"""
Unified scraper orchestrator running ATS company boards, job board sources, and automated career page ATS discovery.
"""

from __future__ import annotations

import json
import logging
import math
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

IS_VERBOSE = (
    "--verbose" in sys.argv
    or "-v" in sys.argv
    or os.environ.get("VERBOSE", "").lower() in ("1", "true", "yes")
)

logger = logging.getLogger("scraper")
logger.setLevel(logging.DEBUG if IS_VERBOSE else logging.INFO)
handler = logging.StreamHandler()
handler.setFormatter(logging.Formatter("%(asctime)s IST [%(levelname)s] %(message)s", datefmt="%Y-%m-%d %H:%M:%S"))
logger.addHandler(handler)


def retry_with_backoff(func, max_attempts, backoff_seconds, label):
    """
    Executes a function with retries and exponential backoff.
    """
    attempt = 1
    while attempt <= max_attempts:
        try:
            return func()
        except Exception as e:
            if attempt == max_attempts:
                raise e
            logger.warning(f"{label} Failed (attempt {attempt}/{max_attempts}): {type(e).__name__}: {str(e)}")
            time.sleep(backoff_seconds)
            attempt += 1


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
    except Exception as keyword_fetch_error:
        logger.warning(f"[orchestrator] Failed to fetch master keywords from backend: {type(keyword_fetch_error).__name__}: {keyword_fetch_error}. Using defaults.")
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
]

PROXY_REQUIRED_SITES = {
    Site.LINKEDIN,
}

LINKEDIN_QUERY_LOCK = threading.Lock()

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
    url = f"{BACKEND_API_URL}/scraper/ats-slugs"
    logger.debug(f"[ats-slugs] Fetching from {url}")
    try:
        response = requests.get(url, headers=request_headers, timeout=15)
        if response.status_code == 200:
            data = response.json().get("data", {})
            slugs_str = ", ".join(f"{k}={len(v)}" for k, v in data.items())
            logger.info(f"[ats-slugs] Fetched: {slugs_str}")
            return data
    except Exception as e:
        logger.error(f"[ats-slugs] Failed to fetch ats slugs: {e}")
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
    except Exception as e:
        logger.error(f"[ats-discovery] Error registering discovered slug {platform_name} {company_slug}: {e}")


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


def probe_single_career_page(company_name: str, company_domain: str = "") -> tuple[str, str, str] | None:
    """
    Probes candidate URLs for a company and extracts ATS configuration details if matched.
    """
    for candidate_url in generate_career_page_candidates(company_name, company_domain):
        try:
            response = HTTP_PROBE_SESSION.get(candidate_url, timeout=8, allow_redirects=True)
            for pattern, platform_name in ATS_DETECTION_PATTERNS:
                if pattern.search(response.url):
                    return platform_name, pattern.search(response.url).group(1).lower().strip("/"), response.url
                if response.status_code == 200 and pattern.search(response.text):
                    return platform_name, pattern.search(response.text).group(1).lower().strip("/"), response.url
        except Exception as probe_error:
            logger.debug(f"[ats-discovery] Probe failed for {candidate_url}: {type(probe_error).__name__}: {probe_error}")
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
    except Exception as e:
        logger.error(f"[ats-discovery] Failed to fetch unmapped companies: {e}")
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
                    platform_name, company_slug, url = result
                    logger.info(f"[ats-discovery] Found new {platform_name} slug: {company_slug} at {url}")
                    register_discovered_ats_slug(platform_name, company_slug)
                    discovered_count += 1
            except Exception as e:
                logger.error(f"[ats-discovery] Probe failed: {e}")

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
            run_id = response.json().get("run_id")
            logger.info(f"[orchestrator] Started run {run_id}")
            return run_id
    except Exception as e:
        logger.error(f"[orchestrator] Failed to start run: {e}")
    return None


def finish_run(run_id: str, run_status: str, error_message: str | None = None, source_statistics: dict | None = None, total_jobs: int = 0) -> None:
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
        "sources_hit": source_statistics or {},
    }
    logger.info(f"[orchestrator] Run {run_id} finished with status={run_status}, jobs_added={total_jobs}")
    try:
        target_endpoint = f"{BACKEND_API_URL}/scraper/finish"
        requests.post(target_endpoint, json=request_payload, headers=request_headers, timeout=20)
    except Exception as e:
        logger.error(f"[orchestrator] Failed to finish run {run_id}: {e}")


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


def sanitize_scalar(value, default=None):
    """
    Returns a JSON-safe scalar, replacing any float nan/inf (which pandas uses for missing values)
    with the specified default so json.dumps never raises ValueError.
    """
    if isinstance(value, float) and not math.isfinite(value):
        return default
    return value


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
        salary_min = raw_job_record.get("salary_min") or 0
        salary_max = raw_job_record.get("salary_max") or 0
        currency = raw_job_record.get("currency") or ""
    else:
        job_identifier = sanitize_scalar(getattr(raw_job_record, "id", "")) or sanitize_scalar(getattr(raw_job_record, "job_id", "")) or ""
        job_title = sanitize_scalar(getattr(raw_job_record, "title", ""), default="") or ""
        extracted_company = company_name or sanitize_scalar(getattr(raw_job_record, "company_name", ""), default="") or sanitize_scalar(getattr(raw_job_record, "company", ""), default="") or ""
        job_url = sanitize_scalar(getattr(raw_job_record, "job_url", ""), default="") or sanitize_scalar(getattr(raw_job_record, "absolute_url", ""), default="") or ""

        location_attribute = getattr(raw_job_record, "location", None)
        if location_attribute and hasattr(location_attribute, "display_location"):
            job_location = location_attribute.display_location()
        else:
            job_location = str(location_attribute) if location_attribute and not (isinstance(location_attribute, float) and not math.isfinite(location_attribute)) else ""

        job_description = sanitize_scalar(getattr(raw_job_record, "description", ""), default="") or sanitize_scalar(getattr(raw_job_record, "description_text", ""), default="") or ""

        date_posted_attribute = getattr(raw_job_record, "date_posted", None)
        if date_posted_attribute and not (isinstance(date_posted_attribute, float) and not math.isfinite(date_posted_attribute)):
            if hasattr(date_posted_attribute, "isoformat"):
                updated_timestamp = date_posted_attribute.isoformat() + "T00:00:00Z"
            else:
                updated_timestamp = str(date_posted_attribute)
        else:
            updated_timestamp = sanitize_scalar(getattr(raw_job_record, "updated_at", ""), default="") or ""

        job_departments = getattr(raw_job_record, "departments", []) or []
        job_offices = getattr(raw_job_record, "offices", []) or []

        raw_salary_min = sanitize_scalar(getattr(raw_job_record, "min_amount", None), default=None)
        raw_salary_max = sanitize_scalar(getattr(raw_job_record, "max_amount", None), default=None)
        try:
            salary_min = int(raw_salary_min) if raw_salary_min is not None else 0
        except (TypeError, ValueError):
            salary_min = 0
        try:
            salary_max = int(raw_salary_max) if raw_salary_max is not None else 0
        except (TypeError, ValueError):
            salary_max = 0
        currency = sanitize_scalar(getattr(raw_job_record, "currency", ""), default="") or ""

    return {
        "job_id": str(job_identifier),
        "title": job_title,
        "updated_at": updated_timestamp,
        "absolute_url": job_url,
        "location": job_location,
        "departments": job_departments if isinstance(job_departments, list) else [],
        "offices": job_offices if isinstance(job_offices, list) else [],
        "description_text": job_description,
        "company": sanitize_company_name(extracted_company),
        "source": source_identifier,
        "salary_min": salary_min,
        "salary_max": salary_max,
        "currency": currency,
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
    logger.info(f"[{platform_name}:{company_slug}] Starting scrape")
    start_time = time.time()
    extracted_jobs = []
    execution_status = "failed"
    
    def do_scrape():
        return scrape_jobs(
            site_name=[platform_name],
            search_term=company_slug,
            results_wanted=100,
            verbose=2 if IS_VERBOSE else 1,
        )

    try:
        scraped_dataframe = retry_with_backoff(do_scrape, 3, 5, f"[{platform_name}:{company_slug}]")
        raw_count = 0
        if scraped_dataframe is not None and not scraped_dataframe.empty:
            raw_count = len(scraped_dataframe)
            for job_row in scraped_dataframe.itertuples():
                normalized_post = normalize_job_post(job_row, platform_name, company_slug)
                is_remote_flag = getattr(job_row, "is_remote", False) or "remote" in normalized_post["location"].lower()
                if is_location_in_scope(normalized_post["location"], is_remote_flag):
                    extracted_jobs.append(normalized_post)
                else:
                    logger.debug(f"[{platform_name}:{company_slug}] Filtered out '{normalized_post['title']}' at '{normalized_post['location']}' (not in scope)")
        
        execution_status = "success"
        elapsed = time.time() - start_time
        logger.info(f"[{platform_name}:{company_slug}] Scraped {raw_count} raw → {len(extracted_jobs)} in-scope jobs in {elapsed:.1f}s")
    except Exception as e:
        logger.error(f"[{platform_name}:{company_slug}] All 3 attempts failed, skipping.")

    return {
        "company": company_slug,
        "platform": platform_name,
        "jobs": extracted_jobs,
        "status": execution_status,
    }


def enrich_linkedin_descriptions(
    cooldown_seconds: int = 60,
    batch_size: int = 100,
    max_jobs: int | None = None,
    since_minutes: int = 120,
) -> int:
    """
    Fetches job descriptions for LinkedIn jobs with empty descriptions in batches after a rate-limit cooldown.
    """
    if max_jobs is not None and max_jobs <= 0:
        return 0

    request_headers = {
        "X-Ingest-Key": INGEST_API_KEY,
    }

    if cooldown_seconds > 0:
        logger.info(f"[enrichment] Waiting {cooldown_seconds}s cooldown before fetching LinkedIn descriptions...")
        time.sleep(cooldown_seconds)

    enrichment_session = requests.Session()
    enrichment_session.headers.update(
        {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
            "Accept-Language": "en-US,en;q=0.9",
        }
    )

    total_enriched_count = 0
    attempted_job_identifiers = set()
    consecutive_rate_limits = 0

    while True:
        if max_jobs is not None and total_enriched_count >= max_jobs:
            break

        current_batch_limit = batch_size
        if max_jobs is not None:
            remaining_quota = max_jobs - total_enriched_count
            current_batch_limit = min(batch_size, remaining_quota)

        pending_endpoint = (
            f"{BACKEND_API_URL}/scraper/jobs-without-description?source=linkedin&limit={current_batch_limit}&since_minutes={since_minutes}"
        )
        try:
            response = requests.get(pending_endpoint, headers=request_headers, timeout=15)
            if response.status_code != 200:
                break
            pending_jobs = response.json().get("data", [])
        except Exception as fetch_error:
            logger.error(f"[enrichment] Failed to fetch pending LinkedIn jobs from backend: {type(fetch_error).__name__}: {fetch_error}")
            break

        unattempted_jobs = [
            job_record for job_record in pending_jobs if job_record.get("id") not in attempted_job_identifiers
        ]
        if not unattempted_jobs:
            break

        logger.info(f"[enrichment] Fetched batch of {len(unattempted_jobs)} pending LinkedIn jobs")

        batch_enriched_updates = []
        for job_record in unattempted_jobs:
            if max_jobs is not None and total_enriched_count + len(batch_enriched_updates) >= max_jobs:
                break

            job_identifier = job_record.get("id")
            job_url = job_record.get("url", "")
            if not job_identifier:
                continue

            attempted_job_identifiers.add(job_identifier)

            if not job_url:
                continue

            linkedin_id_match = re.search(r"/view/(?:[a-zA-Z0-9-]+-)?(\d+)", job_url)
            if not linkedin_id_match:
                continue
            linkedin_job_identifier = linkedin_id_match.group(1)

            logger.debug(f"[enrichment] Fetching description for job {job_identifier} ({job_url})")

            try:
                detail_response = enrichment_session.get(
                    f"https://www.linkedin.com/jobs/view/{linkedin_job_identifier}",
                    timeout=12,
                )
                if detail_response.status_code == 429:
                    consecutive_rate_limits += 1
                    logger.warning(f"[enrichment] Rate limited (429) on job {job_identifier}, sleeping 10s (consecutive={consecutive_rate_limits})")
                    if consecutive_rate_limits >= 2:
                        logger.info("[enrichment] Encountered 429 rate limit twice, halting enrichment pass.")
                        break
                    time.sleep(10)
                    continue

                consecutive_rate_limits = 0
                if detail_response.status_code == 200:
                    blocked_redirect_markers = [
                        "linkedin.com/signup",
                        "linkedin.com/authwall",
                        "linkedin.com/checkpoint",
                        "expired_jd_redirect",
                    ]
                    if any(marker in detail_response.url for marker in blocked_redirect_markers):
                        logger.info(f"[enrichment] Job {job_identifier} redirected to auth wall, skipping")
                        continue

                    detail_soup = BeautifulSoup(detail_response.text, "html.parser")
                    description_element = detail_soup.find(
                        "div",
                        class_=lambda class_name: class_name and "show-more-less-html__markup" in class_name,
                    )
                    if description_element is None:
                        description_element = detail_soup.select_one(
                            ".description__text"
                        ) or detail_soup.select_one(".show-more-less-html")
                        if description_element is not None:
                            for button_element in description_element.find_all(
                                ["button", "span"],
                                class_=lambda class_name: class_name and "show-more-less" in class_name,
                            ):
                                button_element.decompose()

                    if description_element is not None:
                        description_text = description_element.get_text("\n", strip=True)
                        if description_text:
                            logger.debug(f"[enrichment] Got description for job {job_identifier} ({len(description_text)} chars)")
                            batch_enriched_updates.append(
                                {
                                    "id": job_identifier,
                                    "description_text": description_text,
                                }
                            )
                        else:
                            logger.debug(f"[enrichment] No description element found for job {job_identifier}")
                    else:
                        logger.debug(f"[enrichment] No description element found for job {job_identifier}")

            except Exception as e:
                logger.error(f"[enrichment] Exception on job {job_identifier}: {e}")

            time.sleep(1.0)

        if batch_enriched_updates:
            enrich_endpoint = f"{BACKEND_API_URL}/scraper/enrich-descriptions"
            try:
                update_response = requests.post(
                    enrich_endpoint,
                    json={"updates": batch_enriched_updates},
                    headers=request_headers,
                    timeout=20,
                )
                if update_response.status_code == 200:
                    total_enriched_count += len(batch_enriched_updates)
                    logger.info(
                        f"[enrichment] Successfully updated {len(batch_enriched_updates)} LinkedIn job descriptions ({total_enriched_count} total)."
                    )
            except Exception as enrich_post_error:
                logger.error(f"[enrichment] Failed to POST description batch to backend: {type(enrich_post_error).__name__}: {enrich_post_error}")

        if consecutive_rate_limits >= 2:
            break

    logger.info(f"[enrichment] Pass complete: {total_enriched_count} descriptions updated")
    return total_enriched_count


def run_orchestration(target_platform: str | None = None) -> dict:
    """
    Executes the entire job scraping, career page discovery, and ingestion pipeline concurrently.
    """
    ensure_dir(DATA_DIR)

    if target_platform is None:
        target_platform = os.environ.get("TARGET_SITE") or os.environ.get("TARGET_PLATFORM")
    if target_platform:
        target_platform = target_platform.strip().lower()

    active_ats_platform_slugs = fetch_ats_slugs()
    
    total_ats_count = sum(len(slugs) for slugs in active_ats_platform_slugs.values())
    keyword_count = len(KEYWORDS)
    site_lists_active = []
    if target_platform is None or target_platform not in active_ats_platform_slugs:
        site_lists_active.append("board")
    if total_ats_count > 0:
        site_lists_active.append("ats")
        
    logger.info(f"[orchestrator] Startup: fetched ATS slugs across {len(active_ats_platform_slugs)} platforms, keywords={keyword_count}, site_lists_active={site_lists_active}")

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
            if target_platform is None:
                discovery_future = primary_executor.submit(probe_unmapped_companies, MAX_WORKERS, 500)
            else:
                discovery_future = None

            company_futures = []
            for platform_name, company_slugs in active_ats_platform_slugs.items():
                if target_platform is not None and target_platform != platform_name:
                    continue
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
                logger.info(f"[{telemetry_key}] Submitting board scrape")
                start_timestamp = time.time()
                
                def do_board_scrape():
                    sub_executor = ThreadPoolExecutor(max_workers=1)
                    try:
                        scraping_arguments = {
                            "site_name": [target_site],
                            "search_term": search_query,
                            "results_wanted": 200,
                            "hours_old": 24,
                            "verbose": 2 if IS_VERBOSE else 1,
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
                        if target_site == Site.LINKEDIN:
                            scraping_arguments["linkedin_fetch_description"] = False
                            with LINKEDIN_QUERY_LOCK:
                                future_result = sub_executor.submit(scrape_jobs, **scraping_arguments)
                                scraped_dataframe = future_result.result(timeout=REQUEST_TIMEOUT)
                                time.sleep(2.0)
                        else:
                            future_result = sub_executor.submit(scrape_jobs, **scraping_arguments)
                            scraped_dataframe = future_result.result(timeout=REQUEST_TIMEOUT)
                        sub_executor.shutdown(wait=False)
                        return scraped_dataframe
                    except Exception as execution_err:
                        sub_executor.shutdown(wait=False, cancel_futures=True)
                        raise execution_err

                caught_error = None
                scraped_dataframe = None
                try:
                    scraped_dataframe = retry_with_backoff(do_board_scrape, 2, 10, f"[{telemetry_key}]")
                except Exception as e:
                    caught_error = str(e)
                    logger.error(f"[{telemetry_key}] Failed after retries: {type(e).__name__}: {str(e)}")

                elapsed_time = time.time() - start_timestamp

                if scraped_dataframe is None or scraped_dataframe.empty:
                    logger.info(f"[{telemetry_key}] Returned 0 results (empty dataframe)")
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

                logger.info(f"[{telemetry_key}] Got {len(scraped_dataframe)} raw → {len(parsed_posts)} in-scope in {elapsed_time:.1f}s")
                record_source_telemetry(telemetry_key, len(parsed_posts), elapsed_time, caught_error)

                if parsed_posts:
                    with board_raw_jobs_lock:
                        aggregated_raw_jobs.extend(parsed_posts)

            board_futures = []

            if target_platform is None or target_platform not in active_ats_platform_slugs:
                keyword_feed_sites = [site for site in SINGLE_CALL_FEED_SITES if site != Site.DIRECT_CAREERS]
                for feed_site in keyword_feed_sites:
                    if target_platform is not None and target_platform != feed_site.value:
                        continue
                    board_futures.append(
                        primary_executor.submit(
                            scrape_board_site_keyword, feed_site, "software engineer", None, True
                        )
                    )

                if target_platform is None or target_platform == Site.DIRECT_CAREERS.value:
                    board_futures.append(
                        primary_executor.submit(
                            scrape_board_site_keyword, Site.DIRECT_CAREERS, None, None, True
                        )
                    )

                for search_term in KEYWORDS:
                    for india_site in KEYWORD_SEARCHABLE_INDIA_SITES:
                        if target_platform is not None and target_platform != india_site.value:
                            continue
                        board_futures.append(
                            primary_executor.submit(
                                scrape_board_site_keyword, india_site, search_term, "India", False
                            )
                        )
                    for remote_site in KEYWORD_SEARCHABLE_REMOTE_SITES:
                        if target_platform is not None and target_platform != remote_site.value:
                            continue
                        board_futures.append(
                            primary_executor.submit(
                                scrape_board_site_keyword, remote_site, search_term, None, True
                            )
                        )
            
            logger.info(f"[orchestrator] Total futures submitted: {len(company_futures) + len(board_futures)}")

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

            if discovery_future is not None:
                try:
                    discovered_count = discovery_future.result()
                    if discovered_count > 0:
                        logger.info(f"[ats-discovery] Successfully identified and registered {discovered_count} new ATS boards during run.")
                except Exception as discovery_error:
                    logger.error(f"[ats-discovery] Discovery worker raised {type(discovery_error).__name__}: {discovery_error}")

        raw_count_before = len(aggregated_raw_jobs)
        deduplicated_job_records = deduplicate_jobs(aggregated_raw_jobs)
        removed_count = raw_count_before - len(deduplicated_job_records)
        logger.info(f"[orchestrator] Deduplication: {raw_count_before} raw → {len(deduplicated_job_records)} unique jobs (removed {removed_count})")

        for job_record in deduplicated_job_records:
            discovered_ats_details = extract_ats_slug(job_record["absolute_url"])
            if discovered_ats_details:
                register_discovered_ats_slug(discovered_ats_details[0], discovered_ats_details[1])

        save_json(deduplicated_job_records, DATA_DIR / "raw_jobs.json")
        save_json(source_statistics, DATA_DIR / "source_stats.json")
        logger.info(f"[orchestrator] Saved {len(deduplicated_job_records)} raw scraped jobs to raw_jobs.json")
        logger.info(f"[orchestrator] Recorded telemetry across {len(source_statistics)} source search combinations.")

        if not run_identifier:
            run_identifier = start_run()

        total_jobs_added = 0
        if run_identifier:
            ingest_request_headers = {
                "X-Ingest-Key": INGEST_API_KEY,
                "Content-Type": "application/json",
            }
            ingest_endpoint_url = f"{BACKEND_API_URL}/scraper/ingest-raw"
            batch_chunk_size = 500
            total_batches = (len(deduplicated_job_records) + batch_chunk_size - 1) // batch_chunk_size
            for chunk_offset in range(0, len(deduplicated_job_records), batch_chunk_size):
                job_batch_chunk = deduplicated_job_records[chunk_offset:chunk_offset + batch_chunk_size]
                batch_number = chunk_offset // batch_chunk_size + 1
                
                def do_ingest():
                    resp = requests.post(
                        ingest_endpoint_url,
                        json={"run_id": run_identifier, "jobs": job_batch_chunk},
                        headers=ingest_request_headers,
                        timeout=REQUEST_TIMEOUT,
                    )
                    if resp.status_code != 200:
                        logger.warning(f"[ingest] Batch {batch_number} HTTP {resp.status_code}: {resp.text[:200]}")
                        return 0
                    return resp.json().get("jobs_added", 0)
                
                try:
                    added_count = retry_with_backoff(do_ingest, 3, 5, f"[ingest] Batch {batch_number}")
                    if added_count is None:
                        added_count = 0
                    total_jobs_added += added_count
                    logger.info(f"[ingest] Batch {batch_number}/{total_batches}: sending {len(job_batch_chunk)} jobs → got {added_count} new (cumulative {total_jobs_added})")
                except Exception as batch_error:
                    logger.error(f"[ingest] Batch {batch_number} POST failed: {batch_error}")
            
            logger.info(f"[orchestrator] Backend raw ingestion complete. Total jobs added: {total_jobs_added}")

        save_json(run_manifest, DATA_DIR / "manifest.json")

        if target_platform is None or target_platform == Site.LINKEDIN.value:
            current_run_linkedin_count = sum(
                1 for job_record in deduplicated_job_records
                if job_record.get("source") == Site.LINKEDIN.value or job_record.get("source") == "linkedin"
            )
            try:
                enrich_linkedin_descriptions(
                    cooldown_seconds=60,
                    batch_size=100,
                    max_jobs=current_run_linkedin_count,
                    since_minutes=120,
                )
            except Exception as enrichment_err:
                logger.error(f"[enrichment] Enrichment pass failed: {enrichment_err}")

        logger.info(f"[orchestrator] Run complete: {len(deduplicated_job_records)} unique jobs scraped, {total_jobs_added} new jobs ingested to DB")
        
        if source_statistics:
            logger.info("[orchestrator] Source summary:")
            sorted_stats = sorted(source_statistics.items(), key=lambda x: x[1].get("jobs_found", 0), reverse=True)
            for src_key, stats in sorted_stats:
                err_str = f" ERROR: {stats['error']}" if stats.get("error") else ""
                logger.info(f"  {src_key:<35} →  {stats.get('jobs_found', 0)} jobs  ({stats.get('duration_seconds', 0.0)}s){err_str}")

        if run_identifier:
            finish_run(run_identifier, "success", None, source_statistics, total_jobs_added)

        return {"status": "success", "manifest": run_manifest, "source_stats": source_statistics}

    except Exception as execution_exception:
        if run_identifier:
            finish_run(run_identifier, "failed", str(execution_exception), source_statistics, 0)
        raise execution_exception


if __name__ == "__main__":
    if IS_VERBOSE:
        logger.info("[orchestrator] Verbose debug logging active (--verbose)")

    if "--test" in sys.argv:
        KEYWORDS[:] = ["golang developer", "backend engineer"]
        logger.info(f"[orchestrator] Running in TEST mode. Keywords reduced to: {KEYWORDS}")

    selected_platform = None
    for arg_index, arg_value in enumerate(sys.argv):
        if arg_value in ("--platform", "--site") and arg_index + 1 < len(sys.argv):
            selected_platform = sys.argv[arg_index + 1].lower()
            logger.info(f"[orchestrator] Target platform filter active: {selected_platform}")
            break

    run_orchestration(target_platform=selected_platform)
