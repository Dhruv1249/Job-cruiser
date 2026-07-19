"""
Unified scraper orchestrator running both custom ATS clients and JobSpy scrapers.
"""

import json
from pathlib import Path
from datetime import datetime, timezone
from concurrent.futures import ThreadPoolExecutor, as_completed

import requests
from bs4 import BeautifulSoup
from config import (
    DATA_DIR,
    MAX_WORKERS,
    BACKEND_API_URL,
    INGEST_API_KEY,
    USER_AGENT,
    GEMINI_API_KEY
)
from job_sources.greenhouse import GreenhouseClient
from job_sources.lever import LeverClient
from job_sources.ashby import AshbyClient
from job_sources.smartrecruiters import SmartRecruitersClient
from job_sources.workday import WorkdayClient
from job_sources.utils import load_yaml_config

from jobspy import scrape_jobs
from jobspy.model import Site

KEYWORDS = ["software engineer", "developer", "backend", "frontend", "full stack"]
JOBSPY_SITES = [Site.REMOTEOK, Site.WEWORKREMOTELY, Site.HN_HIRING, Site.THE_MUSE]

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
    Save the given data to a JSON file.
    """
    with open(file_path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)

def ensure_dir(path: Path):
    """
    Ensure that the given directory path exists.
    """
    path.mkdir(parents=True, exist_ok=True)

def sanitize_company_name(name: str) -> str:
    """
    Sanitize and limit the length of a company name for filesystem safety.
    """
    if not name:
        return "Unknown"
    
    soup = BeautifulSoup(name, "html.parser")
    clean_name = soup.get_text()
    
    if "http://" in clean_name or "https://" in clean_name:
        clean_name = clean_name.split("http")[0].strip()
        
    for char in ["/", "\\", ":", "*", "?", '"', "<", ">", "|"]:
        clean_name = clean_name.replace(char, " ")
        
    clean_name = clean_name.strip()[:50].strip()
    return clean_name if clean_name else "Unknown"

existing_jobs_cache = {}

def load_existing_jobs_cache() -> None:
    """
    Load previously scraped jobs from jobs_flat.json to cache AI extractions.
    """
    global existing_jobs_cache
    jobs_flat_path = DATA_DIR / "jobs_flat.json"
    if not jobs_flat_path.exists():
        return
    try:
        with open(jobs_flat_path, "r") as f:
            for job in json.load(f):
                url = job.get("absolute_url")
                if url:
                    existing_jobs_cache[url] = {
                        "seniority": job.get("seniority", "Unknown"),
                        "summary": job.get("summary", ""),
                        "tech_stack": job.get("tech_stack", []),
                        "salary_min": job.get("salary_min", 0),
                        "salary_max": job.get("salary_max", 0),
                        "currency": job.get("currency", "USD")
                    }
    except Exception:
        pass

def extract_job_details_with_gemini(title: str, description: str) -> dict:
    """
    Extract structured job details using Gemini API.
    """
    default_result = {
        "seniority": "Unknown",
        "summary": "",
        "tech_stack": [],
        "salary_min": 0,
        "salary_max": 0,
        "currency": "USD"
    }
    if not GEMINI_API_KEY:
        return default_result

    url = f"https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-flash-lite:generateContent?key={GEMINI_API_KEY}"
    prompt = f"Analyze the following job posting.\nTitle: {title}\nDescription: {description}\n\nExtract the following structured details:\n1. Seniority (e.g. Junior, Mid, Senior, Lead, Executive, or Unknown)\n2. A short summary of the role (1-2 sentences)\n3. Tech stack (list of languages/frameworks/tools)\n4. Minimum salary (integer, 0 if not mentioned)\n5. Maximum salary (integer, 0 if not mentioned)\n6. Currency (e.g. USD, EUR, GBP)"

    payload = {
        "contents": [
            {
                "parts": [
                    {"text": prompt}
                ]
            }
        ],
        "generationConfig": {
            "responseMimeType": "application/json",
            "responseSchema": {
                "type": "OBJECT",
                "properties": {
                    "seniority": {"type": "STRING"},
                    "summary": {"type": "STRING"},
                    "tech_stack": {"type": "ARRAY", "items": {"type": "STRING"}},
                    "salary_min": {"type": "INTEGER"},
                    "salary_max": {"type": "INTEGER"},
                    "currency": {"type": "STRING"}
                },
                "required": ["seniority", "summary", "tech_stack"]
            }
        }
    }

    try:
        response = requests.post(url, json=payload, timeout=30)
        if response.status_code == 200:
            res_json = response.json()
            candidates = res_json.get("candidates", [])
            if candidates:
                text_content = candidates[0].get("content", {}).get("parts", [{}])[0].get("text", "")
                if text_content:
                    return json.loads(text_content)
    except Exception:
        pass

    return default_result

def normalize_job_post(job, source: str) -> dict:
    """
    Normalize a job post from any source format into the standard schema.
    """
    if isinstance(job, dict):
        job_id = job.get("job_id") or job.get("id") or ""
        title = job.get("title") or ""
        company = job.get("company") or job.get("company_name") or ""
        url = job.get("absolute_url") or job.get("job_url") or ""
        location = job.get("location") or ""
        description = job.get("description_text") or job.get("description") or ""
        updated_at = job.get("updated_at") or job.get("date_posted") or ""
        departments = job.get("departments") or []
        offices = job.get("offices") or []
    else:
        job_id = getattr(job, "id", "") or getattr(job, "job_id", "") or ""
        title = getattr(job, "title", "")
        company = getattr(job, "company_name", "") or getattr(job, "company", "") or ""
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

    if url in existing_jobs_cache:
        details = existing_jobs_cache[url]
    else:
        details = extract_job_details_with_gemini(title, description)

    return {
        "job_id": job_id,
        "title": title,
        "updated_at": updated_at,
        "absolute_url": url,
        "location": location,
        "departments": departments,
        "offices": offices,
        "description_text": description,
        "company": sanitize_company_name(company),
        "source": source,
        "seniority": details.get("seniority", "Unknown"),
        "summary": details.get("summary", ""),
        "tech_stack": details.get("tech_stack", []),
        "salary_min": details.get("salary_min", 0),
        "salary_max": details.get("salary_max", 0),
        "currency": details.get("currency", "USD")
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
    Fetch and normalize jobs for a company from an ATS platform.
    """
    session = requests.Session()
    session.headers.update({
        "User-Agent": USER_AGENT,
        "Accept": "application/json"
    })

    if platform == "greenhouse":
        client = GreenhouseClient(session=session)
    elif platform == "lever":
        client = LeverClient(session=session)
    elif platform == "ashby":
        client = AshbyClient(session=session)
    elif platform == "smartrecruiters":
        client = SmartRecruitersClient(session=session)
    elif platform == "workday":
        client = WorkdayClient(session=session)
    else:
        return {
            "company": company,
            "platform": platform,
            "status": "invalid_platform"
        }

    if not client.board_exists(company):
        return {
            "company": company,
            "platform": platform,
            "status": "invalid"
        }

    jobs = client.get_jobs(company)
    normalized_jobs = [normalize_job_post(j, platform) for j in jobs]

    return {
        "company": company,
        "platform": platform,
        "jobs": normalized_jobs,
        "status": "success"
    }

def run_orchestration() -> dict:
    """
    Execute the full scraping pipeline and return execution status.
    """
    ensure_dir(DATA_DIR)
    load_existing_jobs_cache()
    config_file_path = Path(__file__).resolve().parent / "companies.yaml"
    config = load_yaml_config(str(config_file_path))

    run_id = start_run()
    all_jobs = []
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
                    all_jobs.extend(res["jobs"])

        for keyword in KEYWORDS:
            try:
                df = scrape_jobs(
                    site_name=JOBSPY_SITES,
                    search_term=keyword,
                    results_wanted=30
                )
                for row in df.itertuples():
                    source = getattr(row, "site", "jobspy")
                    normalized = normalize_job_post(row, source)
                    all_jobs.append(normalized)
            except Exception:
                pass

        deduped_jobs = deduplicate_jobs(all_jobs)
        save_json(deduped_jobs, DATA_DIR / "jobs_flat.json")

        company_jobs = {}
        for job in deduped_jobs:
            comp = job["company"]
            if comp not in company_jobs:
                company_jobs[comp] = []
            company_jobs[comp].append(job)

        for comp, jobs in company_jobs.items():
            if run_id:
                ingest_payload = {
                    "run_id": run_id,
                    "company": comp,
                    "jobs": jobs
                }
                headers = {
                    "X-Ingest-Key": INGEST_API_KEY,
                    "Content-Type": "application/json"
                }
                try:
                    url = f"{BACKEND_API_URL}/scraper/ingest"
                    requests.post(url, json=ingest_payload, headers=headers, timeout=60)
                except Exception:
                    pass

        save_json(manifest, DATA_DIR / "manifest.json")

        if run_id:
            finish_run(run_id, "success")
            
        return {"status": "success", "manifest": manifest}

    except Exception as e:
        if run_id:
            finish_run(run_id, "failed", str(e))
        raise e

if __name__ == "__main__":
    run_orchestration()
