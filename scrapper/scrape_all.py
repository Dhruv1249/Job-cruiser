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
    USER_AGENT,
    GEMINI_API_KEY,
    GEMMA_MOE_MODEL,
    GEMMA_DENSE_MODEL,
    DISABLE_AI_EXTRACTION
)
from job_sources.utils import load_yaml_config

from jobspy import scrape_jobs
from jobspy.model import Site

KEYWORDS = ["software", "developer", "engineer", "tech", "systems"]
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
    Site.CRYPTO_JOBS
]

companies_yaml_lock = threading.Lock()
gemma_limits = {
    GEMMA_MOE_MODEL: {"lock": threading.Lock(), "last_called": 0.0},
    GEMMA_DENSE_MODEL: {"lock": threading.Lock(), "last_called": 0.0}
}

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

def call_gemma_model(model_name: str, payload: dict) -> dict:
    """
    Call the specified Gemma 4 model while respecting the 30 RPM limit.
    """
    limit_info = gemma_limits[model_name]
    with limit_info["lock"]:
        elapsed = time.time() - limit_info["last_called"]
        if elapsed < 2.0:
            time.sleep(2.0 - elapsed)
        limit_info["last_called"] = time.time()
        
    url = f"https://generativelanguage.googleapis.com/v1beta/models/{model_name}:generateContent?key={GEMINI_API_KEY}"
    response = requests.post(url, json=payload, timeout=45)
    return response.json() if response.status_code == 200 else {}

def extract_and_filter_batch_with_gemma(jobs_batch: list[dict], batch_idx: int) -> list[dict]:
    """
    Evaluate a batch of 5 jobs in parallel using twin Gemma 4 models.
    """
    if DISABLE_AI_EXTRACTION or not GEMINI_API_KEY:
        return [
            {
                **job,
                "seniority": "Unknown",
                "summary": "AI extraction disabled",
                "tech_stack": [],
                "salary_min": 0,
                "salary_max": 0,
                "currency": "USD"
            }
            for job in jobs_batch
        ]

    model_name = GEMMA_MOE_MODEL if batch_idx % 2 == 0 else GEMMA_DENSE_MODEL
    
    prompt = """You are a strict job filter. Analyze the following batch of job postings.

CONDITION 1 — EXPERIENCE LEVEL (STRICT):
Mark a job as matched ONLY if it explicitly targets candidates with 0-3 years of experience.
Indicators of a match: "0-3 years", "entry level", "junior", "fresher", "new grad", "associate", "intern", "graduate".
REJECT the job if the title or description mentions: "Senior", "Staff", "Lead", "Principal", "Director", "Manager", "Head of", "VP", "Executive", "5+ years", "7+ years", or any experience requirement above 3 years.
If no experience level is stated, use the job title to judge — titles without seniority prefixes AND in a technical individual contributor role are acceptable.

CONDITION 2 — LOCATION (STRICT):
Match ONLY if the job is:
- Fully remote with no geographic restriction (Global remote), OR
- Based in India (any city, remote/onsite/hybrid within India), OR
- Remote open to India applicants.
REJECT if it is US-only remote, UK-only, Europe-only, or restricted to a specific non-India country.

Only if BOTH conditions are met, mark "is_matched": true and extract:
- seniority: must be exactly one of: "Junior", "Mid", or "Intern". Never return Senior/Lead/Staff/Executive for a matched job.
- summary: A short 1-2 sentence description of the role.
- tech_stack: List of languages, frameworks, or tools mentioned.
- salary_min: Minimum salary (integer, 0 if not mentioned).
- salary_max: Maximum salary (integer, 0 if not mentioned).
- currency: Currency code (e.g. USD, INR, EUR).

For jobs that do not match both conditions, return "is_matched": false.
"""

    payload = {
        "contents": [
            {
                "parts": [
                    {"text": prompt + "\nJobs to evaluate:\n" + json.dumps([{"job_id": j["job_id"], "title": j["title"], "description": j["description_text"]} for j in jobs_batch])}
                ]
            }
        ],
        "generationConfig": {
            "responseMimeType": "application/json",
            "responseSchema": {
                "type": "OBJECT",
                "properties": {
                    "results": {
                        "type": "ARRAY",
                        "items": {
                            "type": "OBJECT",
                            "properties": {
                                "job_id": {"type": "STRING"},
                                "is_matched": {"type": "BOOLEAN"},
                                "seniority": {"type": "STRING"},
                                "summary": {"type": "STRING"},
                                "tech_stack": {"type": "ARRAY", "items": {"type": "STRING"}},
                                "salary_min": {"type": "INTEGER"},
                                "salary_max": {"type": "INTEGER"},
                                "currency": {"type": "STRING"}
                            },
                            "required": ["job_id", "is_matched", "seniority", "summary", "tech_stack"]
                        }
                    }
                },
                "required": ["results"]
            }
        }
    }

    try:
        res_data = call_gemma_model(model_name, payload)
        candidates = res_data.get("candidates", [])
        if candidates:
            text_content = candidates[0].get("content", {}).get("parts", [{}])[0].get("text", "")
            if text_content:
                results = json.loads(text_content).get("results", [])
                results_map = {r["job_id"]: r for r in results}
                
                merged_jobs = []
                for job in jobs_batch:
                    ai_res = results_map.get(job["job_id"], {})
                    if ai_res.get("is_matched", False):
                        merged_jobs.append({
                            **job,
                            "seniority": ai_res.get("seniority", "Unknown"),
                            "summary": ai_res.get("summary", ""),
                            "tech_stack": ai_res.get("tech_stack", []),
                            "salary_min": ai_res.get("salary_min", 0),
                            "salary_max": ai_res.get("salary_max", 0),
                            "currency": ai_res.get("currency", "USD")
                        })
                return merged_jobs
    except Exception:
        pass
        
    return []

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

        india_pass_sites = [
            Site.LINKEDIN,
            Site.INDEED,
            Site.GOOGLE,
        ]
        remote_pass_sites = [
            Site.LINKEDIN,
            Site.INDEED,
            Site.REMOTEOK,
            Site.WEWORKREMOTELY,
            Site.HN_HIRING,
            Site.THE_MUSE,
            Site.HIMALAYAS,
            Site.JOBSPRESSO,
            Site.RUST_CAREERS,
            Site.WORKING_NOMADS,
            Site.WEB3_CAREER,
            Site.CRYPTO_JOBS
        ]

        board_raw_jobs_lock = threading.Lock()

        def scrape_board_site_keyword(site: Site, keyword: str, location: str | None, is_remote: bool) -> None:
            """
            Scrape a single board site for one keyword and append results to all_raw_jobs.
            """
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
                df = scrape_jobs(**kwargs)
                scraped = []
                for row in df.itertuples():
                    source = getattr(row, "site", site.value)
                    scraped.append(normalize_job_post(row, source))
                if scraped:
                    with board_raw_jobs_lock:
                        all_raw_jobs.extend(scraped)
            except Exception:
                pass

        board_futures = []
        with ThreadPoolExecutor(max_workers=MAX_WORKERS) as board_executor:
            for keyword in KEYWORDS:
                for site in india_pass_sites:
                    board_futures.append(
                        board_executor.submit(scrape_board_site_keyword, site, keyword, "India", False)
                    )
                for site in remote_pass_sites:
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

        new_jobs = []
        cached_matched_jobs = []
        
        for job in unique_raw_jobs:
            url = job["absolute_url"]
            if url in existing_jobs_cache:
                cached_job = existing_jobs_cache[url]
                cached_matched_jobs.append(cached_job)
            else:
                new_jobs.append(job)

        batches = [new_jobs[i:i + 5] for i in range(0, len(new_jobs), 5)]
        processed_new_jobs = []
        
        with ThreadPoolExecutor(max_workers=2) as ai_executor:
            ai_futures = []
            for idx, batch in enumerate(batches):
                ai_futures.append(
                    ai_executor.submit(extract_and_filter_batch_with_gemma, batch, idx)
                )
            for future in as_completed(ai_futures):
                processed_new_jobs.extend(future.result())

        final_jobs = deduplicate_jobs(cached_matched_jobs + processed_new_jobs)
        save_json(final_jobs, DATA_DIR / "jobs_flat.json")

        company_jobs = {}
        for job in final_jobs:
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
