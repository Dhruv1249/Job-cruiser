import json
import time
from pathlib import Path
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor, as_completed

import requests
from bs4 import BeautifulSoup
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

from config import *

# ==========================================================
# SESSION
# ==========================================================

session = requests.Session()

session.headers.update({
    "User-Agent": USER_AGENT,
    "Accept": "application/json"
})

retry_strategy = Retry(
    total=RETRY_COUNT,
    backoff_factor=2,
    status_forcelist=[429, 500, 502, 503, 504]
)

adapter = HTTPAdapter(
    max_retries=retry_strategy
)

session.mount("https://", adapter)

# ==========================================================
# UTILS
# ==========================================================

def ensure_dir(path):
    Path(path).mkdir(
        parents=True,
        exist_ok=True
    )


def save_json(data, file_path):

    with open(
        file_path,
        "w",
        encoding="utf-8"
    ) as f:

        json.dump(
            data,
            f,
            indent=2,
            ensure_ascii=False
        )


def html_to_text(html):

    if not html:
        return ""

    soup = BeautifulSoup(
        html,
        "html.parser"
    )

    return soup.get_text(
        separator=" ",
        strip=True
    )

# ==========================================================
# API
# ==========================================================

def api_get(url):

    try:

        time.sleep(
            REQUEST_DELAY
        )

        response = session.get(
            url,
            timeout=REQUEST_TIMEOUT
        )

        if response.status_code == 404:
            return None

        response.raise_for_status()

        return response.json()

    except Exception as e:

        print(
            f"ERROR {url}"
        )

        print(e)

        return None

# ==========================================================
# VALIDATE TOKEN
# ==========================================================

def board_exists(company):

    url = f"{BASE_URL}/{company}"

    try:

        response = session.get(
            url,
            timeout=20
        )

        return response.status_code == 200

    except:
        return False

# ==========================================================
# FETCHERS
# ==========================================================

def get_jobs(company):

    url = (
        f"{BASE_URL}/{company}"
        "/jobs?content=true"
    )

    data = api_get(url)

    if not data:
        return []

    jobs = []

    for job in data.get(
        "jobs",
        []
    ):

        jobs.append({

            "job_id":
                job.get("id"),

            "title":
                job.get("title"),

            "updated_at":
                job.get("updated_at"),

            "absolute_url":
                job.get("absolute_url"),

            "language":
                job.get("language"),

            "location":
                job.get(
                    "location",
                    {}
                ).get(
                    "name",
                    ""
                ),

            "departments": [
                d.get("name")
                for d in job.get(
                    "departments",
                    []
                )
            ],

            "offices": [
                o.get("name")
                for o in job.get(
                    "offices",
                    []
                )
            ],

            "description_text":
                html_to_text(
                    job.get(
                        "content",
                        ""
                    )
                )
        })

    return jobs


def get_offices(company):

    url = (
        f"{BASE_URL}/{company}"
        "/offices"
    )

    data = api_get(url)

    if not data:
        return {"offices": []}

    return data


def get_departments(company):

    url = (
        f"{BASE_URL}/{company}"
        "/departments"
    )

    data = api_get(url)

    if not data:
        return {"departments": []}

    return data

# ==========================================================
# COUNTERS
# ==========================================================

def count_nodes(nodes):

    total = 0

    for node in nodes:

        total += 1

        total += count_nodes(
            node.get(
                "children",
                []
            )
        )

    return total

# ==========================================================
# PROCESS COMPANY
# ==========================================================

def process_company(company):

    if not board_exists(company):

        return {
            "company": company,
            "status": "invalid"
        }

    print(
        f"Processing {company}"
    )

    company_dir = (
        DATA_DIR / company
    )

    ensure_dir(company_dir)

    jobs = get_jobs(company)

    offices = get_offices(company)

    departments = get_departments(company)

    metadata = {

        "company":
            company,

        "scraped_at":
            datetime.utcnow()
            .isoformat()
            + "Z",

        "job_count":
            len(jobs),

        "office_count":
            count_nodes(
                offices.get(
                    "offices",
                    []
                )
            ),

        "department_count":
            count_nodes(
                departments.get(
                    "departments",
                    []
                )
            ),

        "status":
            "success"
    }

    save_json(
        metadata,
        company_dir /
        "company.json"
    )

    save_json(
        jobs,
        company_dir /
        "jobs_flat.json"
    )

    save_json(
        offices,
        company_dir /
        "offices_hierarchy.json"
    )

    save_json(
        departments,
        company_dir /
        "departments_hierarchy.json"
    )

    return metadata

# ==========================================================
# MAIN
# ==========================================================

def main():

    ensure_dir(DATA_DIR)

    companies_file = Path(__file__).with_name(COMPANIES_FILE)

    with open(
        companies_file,
        "r",
        encoding="utf-8"
    ) as f:

        companies = [
            c.strip()
            for c in f
            if c.strip()
        ]

    results = []

    with ThreadPoolExecutor(
        max_workers=
        MAX_WORKERS
    ) as executor:

        futures = {

            executor.submit(
                process_company,
                company
            ): company

            for company
            in companies
        }

        for future in as_completed(
            futures
        ):

            results.append(
                future.result()
            )

    save_json(
        results,
        DATA_DIR /
        "manifest.json"
    )

    print(
        "\nScraping Complete"
    )

if __name__ == "__main__":
    main()