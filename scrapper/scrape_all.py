"""
Unified scraper orchestrator running custom ATS clients and JobSpy scrapers with twin Gemma 4 AI parallel evaluation and self-growing ATS discovery.
"""

import json
import os
import re
import threading
import time
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed

import requests
from bs4 import BeautifulSoup
from config import (
    DATA_DIR,
    MAX_WORKERS,
    BACKEND_API_URL,
    INGEST_API_KEY,
    USER_AGENT
)
from job_sources.utils import load_yaml_config

from jobspy import scrape_jobs
from jobspy.model import Site

THROTTLE_SENSITIVE_KEYWORD = "software engineer"
DEFAULT_KEYWORDS = [
    "backend engineer",
    "backend developer",
    "software engineer backend",
    "backend intern",
    "junior backend engineer",
    "node.js developer",
    "express.js developer",
    "api developer",
    "rest api developer",
    "microservices engineer",
    "distributed systems engineer",
    "cloud backend engineer",
    "server-side developer",
    "platform backend engineer",
    "full stack engineer",
    "full stack developer",
    "fullstack developer",
    "mern stack developer",
    "next.js developer",
    "react developer",
    "frontend engineer",
    "javascript developer",
    "typescript developer",
    "software developer",
    "web developer",
    "cloud engineer",
    "cloud developer",
    "cloud infrastructure engineer",
    "cloud software engineer",
    "aws engineer",
    "aws developer",
    "gcp engineer",
    "google cloud engineer",
    "cloud platform engineer",
    "devops engineer",
    "devops intern",
    "junior devops engineer",
    "platform engineer",
    "infrastructure engineer",
    "site reliability engineer",
    "sre intern",
    "build engineer",
    "release engineer",
    "ci/cd engineer",
    "kubernetes engineer",
    "docker engineer",
    "container platform engineer",
    "cloud native engineer",
    "kubernetes developer",
    "systems engineer",
    "systems programmer",
    "systems software engineer",
    "kernel engineer",
    "kernel developer",
    "operating systems engineer",
    "embedded systems engineer",
    "low level software engineer",
    "firmware engineer",
    "rust developer",
    "rust systems engineer",
    "rust backend engineer",
    "rust systems developer",
    "c developer",
    "c systems programmer",
    "systems research engineer",
    "ai infrastructure engineer",
    "ai platform engineer",
    "ml infrastructure engineer",
    "mlops engineer",
    "ml platform engineer",
    "ai backend engineer",
    "ai systems engineer",
    "genai infrastructure engineer",
    "python developer",
    "python backend engineer",
    "fastapi developer",
    "python software engineer",
    "observability engineer",
    "monitoring engineer",
    "reliability engineer",
    "automation engineer",
    "infrastructure automation engineer",
    "devops automation engineer",
    "forward deployed engineer",
    "forward deployed software engineer",
    "founding engineer",
    "founding software engineer",
    "founding backend engineer",
    "founding full stack engineer",
    "early stage engineer",
    "startup software engineer",
    "startup backend engineer",
    "software engineer i",
    "graduate software engineer",
    "new grad software engineer",
    "software engineer intern",
    "software development engineer",
    "sde i",
    "graduate backend engineer",
    "graduate cloud engineer",
    "graduate devops engineer",
    "entry level software engineer",
    "infrastructure software engineer",
    "platform software engineer",
    "cloud platform developer",
    "infrastructure developer",
    "reliability platform engineer",
    "backend engineer new grad",
    "cloud engineer entry level",
    "devops engineer graduate",
    "platform engineer new grad",
    "sre new grad",
    "kubernetes engineer junior",
    "aws backend engineer",
    "software engineer cloud",
    "software engineer infrastructure",
    "software engineer",
]

def fetch_master_keywords():
    try:
        resp = requests.get(f"{BACKEND_API_URL}/api/keywords", timeout=5)
        if resp.status_code == 200:
            data = resp.json().get("data", [])
            if data:
                return data
    except Exception as e:
        pass
    return DEFAULT_KEYWORDS

KEYWORDS = fetch_master_keywords()
JOBSPY_SITES = [
    Site.LINKEDIN,
    Site.INDEED,
    Site.GLASSDOOR,
    Site.GOOGLE,
    Site.NAUKRI,
    Site.REMOTEOK,
    Site.WEWORKREMOTELY,
    Site.HN_HIRING,
    Site.THE_MUSE,
    Site.HIMALAYAS,
    Site.JOBSPRESSO,
    Site.RUST_CAREERS,
    Site.WORKING_NOMADS,
    Site.WEB3_CAREER,
    Site.CRYPTO_JOBS,
    Site.WELLFOUND,
    Site.DICE,
    Site.BUILTIN,
    Site.SIMPLYHIRED,
    Site.OTTA,
    Site.LEVELSFYI,
    Site.CORD
]

companies_yaml_lock = threading.Lock()


def start_run() -> str:
    """
    Register a new scraper run with the backend telemetry.
    """
    headers = {
        "X-Ingest-Key": INGEST_API_KEY,
        "Content-Type": "application/json"
    }
    try:
        url = f"{BACKEND_API_URL}/scraper/start"
        response = requests.post(url, headers=headers, timeout=20)
        if response.status_code == 200:
            return response.json().get("run_id")
    except Exception:
        pass
    return None

def finish_run(run_id: str, status: str, error_message: str = None):
    """
    Notify the backend telemetry that the scraper run has finished.
    """
    headers = {
        "X-Ingest-Key": INGEST_API_KEY,
        "Content-Type": "application/json"
    }
    payload = {
        "run_id": run_id,
        "status": status,
        "error_message": error_message
    }
    try:
        url = f"{BACKEND_API_URL}/scraper/finish"
        requests.post(url, json=payload, headers=headers, timeout=20)
    except Exception:
        pass

def save_json(data, file_path: Path):
    """
    Save the given data to a JSON file (unindented for fast serialization).
    """
    with open(file_path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False)

def ensure_dir(path: Path):
    """
    Ensure that the given directory path exists.
    """
    path.mkdir(parents=True, exist_ok=True)

def sanitize_company_name(name) -> str:
    """
    Sanitize and limit the length of a company name for filesystem safety.
    """
    if not name or not isinstance(name, (str, bytes)):
        return "Unknown"
    
    name_str = name.strip()
    if not name_str or name_str.lower() == "nan":
        return "Unknown"
    
    soup = BeautifulSoup(name_str, "html.parser")
    clean_name = soup.get_text()
    
    if "http://" in clean_name or "https://" in clean_name:
        clean_name = clean_name.split("http")[0].strip()
        
    for char in ["/", "\\", ":", "*", "?", '"', "<", ">", "|"]:
        clean_name = clean_name.replace(char, " ")
        
    clean_name = clean_name.strip()[:50].strip()
    return clean_name if clean_name else "Unknown"

def extract_ats_slug(url: str) -> tuple[str, str] | None:
    """
    Extract ATS platform and company slug from a job board or career page URL.
    """
    if not url:
        return None
        
    url_lower = url.lower()
    
    if "boards.greenhouse.io/" in url_lower or "boards-api.greenhouse.io/" in url_lower:
        parts = [p for p in url.split("/") if p]
        for i, part in enumerate(parts):
            if "greenhouse.io" in part and i + 1 < len(parts):
                return "greenhouse", parts[i+1].split("?")[0].split("#")[0].strip()
                
    if "jobs.lever.co/" in url_lower:
        parts = [p for p in url.split("/") if p]
        for i, part in enumerate(parts):
            if "lever.co" in part and i + 1 < len(parts):
                return "lever", parts[i+1].split("?")[0].split("#")[0].strip()
                
    if "jobs.ashbyhq.com/" in url_lower:
        parts = [p for p in url.split("/") if p]
        for i, part in enumerate(parts):
            if "ashbyhq.com" in part and i + 1 < len(parts):
                return "ashby", parts[i+1].split("?")[0].split("#")[0].strip()
                
    if "jobs.smartrecruiters.com/" in url_lower:
        parts = [p for p in url.split("/") if p]
        for i, part in enumerate(parts):
            if "smartrecruiters.com" in part and i + 1 < len(parts):
                return "smartrecruiters", parts[i+1].split("?")[0].split("#")[0].strip()
                
    if ".myworkdaysite.com/" in url_lower:
        domain = url.split("://")[-1].split("/")[0]
        if "myworkdaysite.com" in domain:
            return "workday", domain.split(".")[0].strip()
            
    return None

def register_discovered_company(platform: str, slug: str):
    """
    Register a discovered company slug in companies.yaml if not already present.
    """
    if not slug:
        return
        
    with companies_yaml_lock:
        config_path = Path(__file__).resolve().parent / "companies.yaml"
        config = load_yaml_config(str(config_path))
        
        if platform not in config:
            config[platform] = []
            
        if slug not in config[platform]:
            config[platform].append(slug)
            with open(config_path, "w", encoding="utf-8") as f:
                for p in ["greenhouse", "lever", "ashby", "smartrecruiters", "workday"]:
                    f.write(f"{p}:\n")
                    slugs = sorted(list(set(config.get(p, []))))
                    for s in slugs:
                        if s:
                            f.write(f"  - {s}\n")
                    f.write("\n")


        
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

NON_INDIA_ONSITE_LOCATIONS = [
    "san francisco", "new york", "austin", "seattle", "boston", "chicago", "los angeles",
    "london", "berlin", "paris", "toronto", "vancouver", "sydney", "singapore", "amsterdam",
    "madrid", "united states", "united kingdom", "germany", "france", "canada", "australia",
    "netherlands", "spain"
]

def is_location_in_scope(location_str: str, is_remote: bool = False) -> bool:
    """
    Classify if a job location is in scope (India or Global Remote).

    Accept only when:
      - location is empty/unknown (benefit of the doubt)
      - location explicitly names an Indian city/state/region
      - job carries a remote/WFH indicator AND is not restricted to a non-India country via
        an explicit phrase (e.g. "US Remote Only"), city names alone do not block remote jobs

    Reject everything else by default, including any non-empty location with no India/remote signal.
    """
    if not location_str:
        return True

    loc_lower = location_str.lower()

    if any(excl in loc_lower for excl in EXPLICIT_NON_INDIA_RESTRICTIONS):
        return False

    for indian_loc in INDIAN_LOCATIONS:
        if indian_loc in loc_lower:
            return True

    has_remote_indicator = is_remote or any(rem in loc_lower for rem in REMOTE_INDICATORS)
    if has_remote_indicator:
        return True

    return False


def normalize_job_post(job, source: str, company_name: str = None) -> dict:
    """
    Normalize a job post from any source format into the standard schema.
    """
    if isinstance(job, dict):
        job_id = job.get("job_id") or job.get("id") or ""
        title = job.get("title") or ""
        company = company_name or job.get("company") or job.get("company_name") or ""
        url = job.get("absolute_url") or job.get("job_url") or ""
        location = job.get("location") or ""
        description = job.get("description_text") or job.get("description") or ""
        updated_at = job.get("updated_at") or job.get("date_posted") or ""
        departments = job.get("departments") or []
        offices = job.get("offices") or []
    else:
        job_id = getattr(job, "id", "") or getattr(job, "job_id", "") or ""
        title = getattr(job, "title", "")
        company = company_name or getattr(job, "company_name", "") or getattr(job, "company", "") or ""
        url = getattr(job, "job_url", "") or getattr(job, "absolute_url", "") or ""
        
        location_obj = getattr(job, "location", None)
        if location_obj and hasattr(location_obj, "display_location"):
            location = location_obj.display_location()
        else:
            location = str(location_obj) if location_obj else ""

        description = getattr(job, "description", "") or getattr(job, "description_text", "") or ""
        
        date_posted_val = getattr(job, "date_posted", None)
        if date_posted_val:
            if hasattr(date_posted_val, "isoformat"):
                updated_at = date_posted_val.isoformat() + "T00:00:00Z"
            else:
                updated_at = str(date_posted_val)
        else:
            updated_at = getattr(job, "updated_at", "") or ""
            
        departments = getattr(job, "departments", [])
        offices = getattr(job, "offices", [])

    return {
        "job_id": str(job_id),
        "title": title,
        "updated_at": updated_at,
        "absolute_url": url,
        "location": location,
        "departments": departments,
        "offices": offices,
        "description_text": description,
        "company": sanitize_company_name(company),
        "source": source
    }

def deduplicate_jobs(jobs: list[dict]) -> list[dict]:
    """
    Remove duplicate jobs based on company, title, and location.
    """
    seen = set()
    unique_jobs = []
    for job in jobs:
        key = (
            (job.get("company") or "").strip().lower(),
            (job.get("title") or "").strip().lower(),
            (job.get("location") or "").strip().lower()
        )
        if key not in seen:
            seen.add(key)
            unique_jobs.append(job)
    return unique_jobs

def process_company(company: str, platform: str, run_id: str = None) -> dict:
    """
    Fetch and normalize jobs for a company from an ATS platform using JobSpy.
    """
    try:
        df = scrape_jobs(
            site_name=[platform],
            search_term=company,
            results_wanted=100
        )
        jobs = []
        for row in df.itertuples():
            normalized = normalize_job_post(row, platform, company)
            is_remote_flag = getattr(row, "is_remote", False) or "remote" in normalized["location"].lower()
            if is_location_in_scope(normalized["location"], is_remote_flag):
                jobs.append(normalized)
        status = "success"
    except Exception:
        jobs = []
        status = "failed"

    return {
        "company": company,
        "platform": platform,
        "jobs": jobs,
        "status": status
    }

def run_orchestration() -> dict:
    """
    Execute the full scraping pipeline and return execution status.
    """
    ensure_dir(DATA_DIR)
    
    existing_jobs_cache = {}
    jobs_flat_path = DATA_DIR / "jobs_flat.json"
    if jobs_flat_path.exists():
        try:
            with open(jobs_flat_path, "r") as f:
                for job in json.load(f):
                    url = job.get("absolute_url")
                    if url:
                        existing_jobs_cache[url] = job
        except Exception:
            pass

    config_file_path = Path(__file__).resolve().parent / "companies.yaml"
    config = load_yaml_config(str(config_file_path))

    run_id = start_run()
    all_raw_jobs = []
    manifest = []

    try:
        with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
            futures = []
            for platform, companies in config.items():
                for company in companies:
                    futures.append(
                        executor.submit(process_company, company, platform, run_id)
                    )
            for future in as_completed(futures):
                res = future.result()
                manifest.append({
                    "company": res["company"],
                    "platform": res["platform"],
                    "job_count": len(res.get("jobs", [])),
                    "status": res["status"]
                })
                if res["status"] == "success":
                    all_raw_jobs.extend(res["jobs"])

        throttle_sensitive_india_sites = [
            Site.LINKEDIN,
            Site.GLASSDOOR,
            Site.NAUKRI,
        ]
        throttle_sensitive_remote_sites = [
            Site.LINKEDIN,
            Site.GLASSDOOR,
        ]
        high_volume_india_sites = [
            Site.INDEED,
            Site.GOOGLE,
            Site.AMAZON,
            Site.MICROSOFT,
        ]
        high_volume_remote_sites = [
            Site.INDEED,
            Site.DICE,
            Site.SIMPLYHIRED,
            Site.BUILTIN,
            Site.WELLFOUND,
            Site.LEVELSFYI,
        ]
        niche_remote_sites = [
            Site.REMOTEOK,
            Site.WEWORKREMOTELY,
            Site.HN_HIRING,
            Site.THE_MUSE,
            Site.HIMALAYAS,
            Site.JOBSPRESSO,
            Site.RUST_CAREERS,
            Site.WORKING_NOMADS,
            Site.WEB3_CAREER,
            Site.CRYPTO_JOBS,
            Site.OTTA,
            Site.CORD,
            Site.DIRECT_CAREERS,
        ]

        board_raw_jobs_lock = threading.Lock()

        def scrape_board_site_keyword(site: Site, keyword: str, location: str | None, is_remote: bool) -> None:
            """
            Scrape a single board site for one keyword and append results to all_raw_jobs.
            Enforces a 15-second hard timeout per keyword search without blocking worker shutdown on hanging sockets.
            """
            sub_exec = ThreadPoolExecutor(max_workers=1)
            df = None
            try:
                kwargs = {
                    "site_name": [site],
                    "search_term": keyword,
                    "results_wanted": 200,
                    "hours_old": 24,
                }
                if location:
                    kwargs["location"] = location
                if is_remote:
                    kwargs["is_remote"] = True

                future = sub_exec.submit(scrape_jobs, **kwargs)
                df = future.result(timeout=15)
                sub_exec.shutdown(wait=False)
            except Exception:
                sub_exec.shutdown(wait=False, cancel_futures=True)
                return

            if df is None or df.empty:
                return

            scraped = []
            for row in df.itertuples():
                source = getattr(row, "site", site.value)
                normalized = normalize_job_post(row, source)
                is_remote_flag = is_remote or getattr(row, "is_remote", False) or "remote" in normalized["location"].lower()
                if is_location_in_scope(normalized["location"], is_remote_flag):
                    scraped.append(normalized)

            if scraped:
                with board_raw_jobs_lock:
                    all_raw_jobs.extend(scraped)

        board_futures = []
        with ThreadPoolExecutor(max_workers=MAX_WORKERS) as board_executor:
            for site in throttle_sensitive_india_sites:
                board_futures.append(
                    board_executor.submit(scrape_board_site_keyword, site, THROTTLE_SENSITIVE_KEYWORD, "India", False)
                )
            for site in throttle_sensitive_remote_sites:
                board_futures.append(
                    board_executor.submit(scrape_board_site_keyword, site, THROTTLE_SENSITIVE_KEYWORD, None, True)
                )
            for keyword in KEYWORDS:
                for site in high_volume_india_sites:
                    board_futures.append(
                        board_executor.submit(scrape_board_site_keyword, site, keyword, "India", False)
                    )
                for site in high_volume_remote_sites:
                    board_futures.append(
                        board_executor.submit(scrape_board_site_keyword, site, keyword, None, True)
                    )
                for site in niche_remote_sites:
                    board_futures.append(
                        board_executor.submit(scrape_board_site_keyword, site, keyword, None, True)
                    )
            for future in as_completed(board_futures):
                future.result()


        unique_raw_jobs = deduplicate_jobs(all_raw_jobs)
        
        for job in unique_raw_jobs:
            ats_info = extract_ats_slug(job["absolute_url"])
            if ats_info:
                register_discovered_company(ats_info[0], ats_info[1])

        save_json(unique_raw_jobs, DATA_DIR / "raw_jobs.json")
        print(f"[Scraper] Saved {len(unique_raw_jobs)} raw scraped jobs to raw_jobs.json", flush=True)

        if not run_id:
            run_id = start_run()

        if run_id:
            ingest_headers = {
                "X-Ingest-Key": INGEST_API_KEY,
                "Content-Type": "application/json",
            }
            ingest_url = f"{BACKEND_API_URL}/scraper/ingest-raw"
            chunk_size = 500
            total_added = 0
            for i in range(0, len(unique_raw_jobs), chunk_size):
                chunk = unique_raw_jobs[i:i + chunk_size]
                try:
                    res = requests.post(ingest_url, json={"run_id": run_id, "jobs": chunk}, headers=ingest_headers, timeout=120)
                    if res.status_code == 200:
                        added = res.json().get("jobs_added", 0)
                        total_added += added
                        print(f"[Scraper] Ingested batch {i // chunk_size + 1}/{(len(unique_raw_jobs) + chunk_size - 1) // chunk_size} ({added} jobs)", flush=True)
                    else:
                        print(f"[Scraper] Batch {i // chunk_size + 1} returned {res.status_code}: {res.text[:200]}", flush=True)
                except Exception as ingest_error:
                    print(f"[Scraper] Failed to POST chunk {i // chunk_size + 1}: {ingest_error}", flush=True)
            print(f"[Scraper] Backend raw ingestion complete. Total jobs added: {total_added}", flush=True)

        save_json(manifest, DATA_DIR / "manifest.json")

        if run_id:
            finish_run(run_id, "success")

        return {"status": "success", "manifest": manifest}

    except Exception as e:
        if run_id:
            finish_run(run_id, "failed", str(e))
        raise e

if __name__ == "__main__":
    import sys
    if "--test" in sys.argv:
        KEYWORDS[:] = ["golang developer", "go developer", "backend engineer"]
        print(f"[Scraper] Running in TEST mode. Keywords reduced to: {KEYWORDS}", flush=True)
    run_orchestration()
