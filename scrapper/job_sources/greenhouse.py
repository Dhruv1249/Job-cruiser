"""
Greenhouse ATS API client.
"""

import requests
from bs4 import BeautifulSoup

class GreenhouseClient:
    """
    Client for interacting with the Greenhouse Job Board API.
    """

    def __init__(self, session=None):
        """
        Initialize the Greenhouse client with an optional requests session.
        """
        if session is not None:
            self.session = session
        else:
            self.session = requests.Session()
            self.session.headers.update({
                "User-Agent": "JobCruiser/1.0",
                "Accept": "application/json"
            })
        self.base_url = "https://boards-api.greenhouse.io/v1/boards"

    def board_exists(self, company: str) -> bool:
        """
        Check if a Greenhouse job board exists for the given company.
        """
        url = f"{self.base_url}/{company}"
        try:
            response = self.session.get(url, timeout=20)
            return response.status_code == 200
        except Exception:
            return False

    def get_jobs(self, company: str) -> list:
        """
        Fetch and normalize all jobs from the Greenhouse board for the company.
        """
        url = f"{self.base_url}/{company}/jobs?content=true"
        try:
            response = self.session.get(url, timeout=60)
            if response.status_code != 200:
                return []
            data = response.json()
        except Exception:
            return []

        jobs = []
        for job_data in data.get("jobs", []):
            description_html = job_data.get("content", "")
            description_text = ""
            if description_html:
                soup = BeautifulSoup(description_html, "html.parser")
                description_text = soup.get_text(separator=" ", strip=True)

            jobs.append({
                "job_id": job_data.get("id"),
                "title": job_data.get("title"),
                "updated_at": job_data.get("updated_at"),
                "absolute_url": job_data.get("absolute_url"),
                "location": job_data.get("location", {}).get("name", ""),
                "departments": [
                    dept.get("name")
                    for dept in job_data.get("departments", [])
                    if dept.get("name")
                ],
                "offices": [
                    office.get("name")
                    for office in job_data.get("offices", [])
                    if office.get("name")
                ],
                "description_text": description_text
            })
        return jobs

    def get_offices(self, company: str) -> dict:
        """
        Fetch the hierarchy of offices for the company.
        """
        url = f"{self.base_url}/{company}/offices"
        try:
            response = self.session.get(url, timeout=60)
            if response.status_code != 200:
                return {"offices": []}
            return response.json()
        except Exception:
            return {"offices": []}

    def get_departments(self, company: str) -> dict:
        """
        Fetch the hierarchy of departments for the company.
        """
        url = f"{self.base_url}/{company}/departments"
        try:
            response = self.session.get(url, timeout=60)
            if response.status_code != 200:
                return {"departments": []}
            return response.json()
        except Exception:
            return {"departments": []}
