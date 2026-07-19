"""
SmartRecruiters ATS API client.
"""

import requests
from bs4 import BeautifulSoup

class SmartRecruitersClient:
    """
    Client for interacting with the SmartRecruiters Job Board API.
    """

    def __init__(self, session=None):
        """
        Initialize the SmartRecruiters client.
        """
        if session is not None:
            self.session = session
        else:
            self.session = requests.Session()
            self.session.headers.update({
                "User-Agent": "JobCruiser/1.0",
                "Accept": "application/json"
            })
        self.base_url = "https://api.smartrecruiters.com/v1/companies"

    def board_exists(self, company: str) -> bool:
        """
        Check if a SmartRecruiters job board exists for the given company ID.
        """
        url = f"{self.base_url}/{company}/postings?limit=1"
        try:
            response = self.session.get(url, timeout=20)
            return response.status_code == 200
        except Exception:
            return False

    def get_jobs(self, company: str) -> list:
        """
        Fetch and normalize all jobs from the SmartRecruiters board for the company.
        """
        url = f"{self.base_url}/{company}/postings"
        try:
            response = self.session.get(url, timeout=60)
            if response.status_code != 200:
                return []
            data = response.json()
        except Exception:
            return []

        jobs = []
        for posting in data.get("content", []):
            posting_id = posting.get("id")
            if not posting_id:
                continue

            description_text = ""
            detail_url = f"{self.base_url}/{company}/postings/{posting_id}"
            try:
                detail_resp = self.session.get(detail_url, timeout=20)
                if detail_resp.status_code == 200:
                    detail_data = detail_resp.json()
                    desc_html = detail_data.get("jobAd", {}).get("sections", {}).get("jobDescription", {}).get("text", "")
                    if desc_html:
                        soup = BeautifulSoup(desc_html, "html.parser")
                        description_text = soup.get_text(separator=" ", strip=True)
            except Exception:
                pass

            loc = posting.get("location", {})
            loc_parts = []
            for k in ["city", "region", "country"]:
                val = loc.get(k)
                if val:
                    loc_parts.append(val)
            location = ", ".join(loc_parts)

            dept_name = posting.get("department", {}).get("name")
            departments = [dept_name] if dept_name else []

            jobs.append({
                "job_id": posting_id,
                "title": posting.get("name"),
                "updated_at": posting.get("releasedDate"),
                "absolute_url": f"https://jobs.smartrecruiters.com/{company}/{posting_id}",
                "location": location,
                "departments": departments,
                "offices": [location] if location else [],
                "description_text": description_text
            })
        return jobs
