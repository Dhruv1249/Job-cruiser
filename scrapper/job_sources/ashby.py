"""
Ashby ATS API client.
"""

import requests
from bs4 import BeautifulSoup

class AshbyClient:
    """
    Client for interacting with the Ashby Job Board API.
    """

    def __init__(self, session=None):
        """
        Initialize the Ashby client with an optional requests session.
        """
        if session is not None:
            self.session = session
        else:
            self.session = requests.Session()
            self.session.headers.update({
                "User-Agent": "JobCruiser/1.0",
                "Accept": "application/json"
            })
        self.base_url = "https://api.ashbyhq.com/posting-api/job-board"

    def board_exists(self, company: str) -> bool:
        """
        Check if an Ashby job board exists for the given company organization slug.
        """
        url = f"{self.base_url}/{company}"
        try:
            response = self.session.get(url, timeout=20)
            return response.status_code == 200
        except Exception:
            return False

    def get_jobs(self, company: str) -> list:
        """
        Fetch and normalize all active jobs from the Ashby job board.
        """
        url = f"{self.base_url}/{company}"
        try:
            response = self.session.get(url, timeout=60)
            if response.status_code != 200:
                return []
            data = response.json()
        except Exception:
            return []

        jobs = []
        for job_data in data.get("jobs", []):
            description_html = job_data.get("descriptionHtml", "")
            description_text = ""
            if description_html:
                soup = BeautifulSoup(description_html, "html.parser")
                description_text = soup.get_text(separator=" ", strip=True)

            location = job_data.get("location", "")
            
            departments = []
            dept = job_data.get("department")
            if dept:
                departments.append(dept)
            team = job_data.get("team")
            if team:
                departments.append(team)

            offices = []
            if location:
                offices.append(location)

            jobs.append({
                "job_id": job_data.get("id"),
                "title": job_data.get("title"),
                "updated_at": job_data.get("publishedAt"),
                "absolute_url": job_data.get("jobUrl"),
                "location": location,
                "departments": departments,
                "offices": offices,
                "description_text": description_text
            })
        return jobs
