"""
Workday ATS API client.
"""

import requests
from bs4 import BeautifulSoup

class WorkdayClient:
    """
    Client for interacting with the Workday careers site JSON API.
    """

    def __init__(self, session=None):
        """
        Initialize the Workday client.
        """
        if session is not None:
            self.session = session
        else:
            self.session = requests.Session()
            self.session.headers.update({
                "User-Agent": "JobCruiser/1.0",
                "Accept": "application/json"
            })

    def board_exists(self, company: str) -> bool:
        """
        Check if a Workday job board exists for the given company subdomain.
        """
        url = f"https://{company}.wd5.myworkdaysite.com/wday/cxs/{company}/External/jobs"
        payload = {"limit": 1, "offset": 0, "searchText": ""}
        try:
            response = self.session.post(url, json=payload, timeout=20)
            return response.status_code == 200
        except Exception:
            return False

    def get_jobs(self, company: str) -> list:
        """
        Fetch and normalize all jobs from the Workday board for the company.
        """
        url = f"https://{company}.wd5.myworkdaysite.com/wday/cxs/{company}/External/jobs"
        payload = {"limit": 100, "offset": 0, "searchText": ""}
        try:
            response = self.session.post(url, json=payload, timeout=60)
            if response.status_code != 200:
                return []
            data = response.json()
        except Exception:
            return []

        jobs = []
        for posting in data.get("jobPostings", []):
            path = posting.get("externalPath")
            if not path:
                continue

            job_id = ""
            description_text = ""
            detail_url = f"https://{company}.wd5.myworkdaysite.com/wday/cxs/{company}/External{path}"
            try:
                detail_resp = self.session.get(detail_url, timeout=20)
                if detail_resp.status_code == 200:
                    detail_data = detail_resp.json()
                    info = detail_data.get("jobPostingInfo", {})
                    job_id = info.get("id", "")
                    desc_html = info.get("jobDescription", "")
                    if desc_html:
                        soup = BeautifulSoup(desc_html, "html.parser")
                        description_text = soup.get_text(separator=" ", strip=True)
            except Exception:
                pass

            if not job_id:
                job_id = path.split("_")[-1] if "_" in path else path.split("/")[-1]

            location = posting.get("locationsText", "")
            offices = [location] if location else []

            jobs.append({
                "job_id": job_id,
                "title": posting.get("title"),
                "updated_at": posting.get("postedOn"),
                "absolute_url": f"https://{company}.wd5.myworkdaysite.com/en-US/{company}/details/{job_id}" if job_id else f"https://{company}.wd5.myworkdaysite.com/en-US/{company}/details{path}",
                "location": location,
                "departments": [],
                "offices": offices,
                "description_text": description_text
            })
        return jobs
