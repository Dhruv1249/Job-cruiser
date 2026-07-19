"""
Main entrypoint for the job board scraper pipeline.
"""

import json
import time
from pathlib import Path
from datetime import datetime, timezone
from concurrent.futures import ThreadPoolExecutor, as_completed

import requests
from config import (
    DATA_DIR,
    MAX_WORKERS,
    BACKEND_API_URL,
    INGEST_API_KEY,
    USER_AGENT
)
from job_sources.greenhouse import GreenhouseClient
from job_sources.lever import LeverClient
from job_sources.ashby import AshbyClient
from job_sources.utils import load_yaml_config

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

def process_company(company: str, platform: str, run_id: str = None) -> dict:
    """
    Process jobs for a specific company on a given ATS platform.
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

    company_dir = DATA_DIR / company
    ensure_dir(company_dir)

    jobs = client.get_jobs(company)

    metadata = {
        "company": company,
        "platform": platform,
        "scraped_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "job_count": len(jobs),
        "status": "success"
    }

    save_json(metadata, company_dir / "company.json")
    save_json(jobs, company_dir / "jobs_flat.json")

    if run_id:
        ingest_payload = {
            "run_id": run_id,
            "company": company,
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

    return metadata

def main():
    """
    Main orchestrator logic to parse configuration and run scraper tasks.
    """
    ensure_dir(DATA_DIR)
    config_file_path = Path(__file__).resolve().parent / "companies.yaml"
    config = load_yaml_config(str(config_file_path))

    run_id = start_run()
    results = []

    try:
        with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
            futures = []
            for platform, companies in config.items():
                for company in companies:
                    futures.append(
                        executor.submit(process_company, company, platform, run_id)
                    )
            for future in as_completed(futures):
                results.append(future.result())

        if run_id:
            finish_run(run_id, "success")
    except Exception as e:
        if run_id:
            finish_run(run_id, "failed", str(e))
        raise e

    save_json(results, DATA_DIR / "manifest.json")

if __name__ == "__main__":
    main()
