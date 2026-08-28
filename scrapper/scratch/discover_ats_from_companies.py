"""
Automated career page ATS discovery scanner that probes database companies for ATS board integrations.
"""

import os
import re
import time
import requests
from concurrent.futures import ThreadPoolExecutor, as_completed

BACKEND_API_URL = os.environ.get("BACKEND_API_URL", "http://localhost:8080/api")
INGEST_API_KEY = os.environ.get("INGEST_API_KEY", "dev-ingest-key-12345")

ATS_PATTERNS = [
    (re.compile(r"(?:boards|job-boards|boards\.eu)\.greenhouse\.io/([^/?#]+)"), "greenhouse"),
    (re.compile(r"jobs\.lever\.co/([^/?#]+)"), "lever"),
    (re.compile(r"jobs\.ashbyhq\.com/([^/?#]+)"), "ashby"),
    (re.compile(r"jobs\.smartrecruiters\.com/([^/?#]+)"), "smartrecruiters"),
    (re.compile(r"([a-z0-9-]+)\.wd\d+\.myworkdaysite\.com"), "workday"),
]

CAREER_PATHS = ["/jobs", "/careers", "/career", "/work-with-us", "/join-us", "/openings"]
CAREER_SUBDOMAINS = ["careers", "jobs"]

HTTP_SESSION = requests.Session()
HTTP_SESSION.headers.update(
    {
        "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    }
)
HTTP_SESSION.max_redirects = 5


def generate_company_career_url_candidates(company_name: str) -> list[str]:
    """
    Generates candidate career page URLs based on standard company naming patterns.
    """
    sanitized_name = re.sub(
        r"\b(inc\.?|llc\.?|ltd\.?|corp\.?|co\.?|plc\.?|technologies|solutions|services|group)\b",
        "",
        company_name,
        flags=re.IGNORECASE,
    )
    url_slug = re.sub(r"[^a-z0-9]", "", sanitized_name.lower().strip())
    if not url_slug or len(url_slug) < 3:
        return []

    candidate_urls = []
    for path in CAREER_PATHS:
        candidate_urls.append(f"https://{url_slug}.com{path}")
    for subdomain in CAREER_SUBDOMAINS:
        candidate_urls.append(f"https://{subdomain}.{url_slug}.com")

    return candidate_urls


def extract_ats_details_from_response(final_url: str, html_body: str) -> tuple[str, str] | None:
    """
    Extracts ATS platform name and company slug from a response URL and HTML body.
    """
    for regex_pattern, platform_name in ATS_PATTERNS:
        url_match = regex_pattern.search(final_url)
        if url_match:
            return platform_name, url_match.group(1).lower().strip("/")
        if html_body:
            html_match = regex_pattern.search(html_body)
            if html_match:
                return platform_name, html_match.group(1).lower().strip("/")
    return None


def probe_single_company(company_name: str) -> tuple[str, str] | None:
    """
    Probes candidate career URLs for a single company name and returns ATS details if found.
    """
    for candidate_url in generate_company_career_url_candidates(company_name):
        try:
            http_response = HTTP_SESSION.get(candidate_url, timeout=8, allow_redirects=True)
            discovered_details = extract_ats_details_from_response(
                http_response.url,
                http_response.text if http_response.status_code == 200 else "",
            )
            if discovered_details:
                return discovered_details
        except Exception:
            continue
    return None


def register_ats_slug_remotely(platform_name: str, company_slug: str) -> None:
    """
    Sends a discovered ATS platform slug to the backend API for persistent storage.
    """
    request_headers = {
        "X-Ingest-Key": INGEST_API_KEY,
        "Content-Type": "application/json",
    }
    request_payload = {
        "platform": platform_name,
        "slug": company_slug,
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


def execute_company_probing_scan(worker_concurrency: int = 20) -> None:
    """
    Fetches all distinct companies from the backend and probes their career pages for ATS configurations.
    """
    request_headers = {
        "X-Ingest-Key": INGEST_API_KEY,
    }
    api_response = requests.get(
        f"{BACKEND_API_URL}/scraper/companies",
        headers=request_headers,
        timeout=15,
    )
    company_names_list = api_response.json().get("data", [])
    total_companies = len(company_names_list)
    print(f"Beginning career page ATS probe for {total_companies} companies with {worker_concurrency} workers...")

    discovered_slug_count = 0

    with ThreadPoolExecutor(max_workers=worker_concurrency) as probe_executor:
        future_map = {
            probe_executor.submit(probe_single_company, company_name): company_name
            for company_name in company_names_list
        }
        for index, completed_future in enumerate(as_completed(future_map)):
            company_name = future_map[completed_future]
            try:
                probe_result = completed_future.result()
                if probe_result:
                    platform_name, company_slug = probe_result
                    register_ats_slug_remotely(platform_name, company_slug)
                    discovered_slug_count += 1
                    print(f"  ✓ {company_name} -> {platform_name}/{company_slug}")
            except Exception:
                pass

            if index > 0 and index % 200 == 0:
                print(f"  Progress: {index}/{total_companies} probed, {discovered_slug_count} ATS boards identified.")

    print(f"Discovery scan complete. Identified and registered {discovered_slug_count} ATS boards.")


if __name__ == "__main__":
    execute_company_probing_scan()
