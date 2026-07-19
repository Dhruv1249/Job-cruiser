"""
Lever ATS API client.
"""

import datetime
import requests
from bs4 import BeautifulSoup

class LeverClient:
    """
    Client for interacting with the Lever Postings API.
    """

    def __init__(self, session=None):
        """
        Initialize the Lever client with an optional requests session.
        """
        if session is not None:
            self.session = session
        else:
            self.session = requests.Session()
            self.session.headers.update({
                "User-Agent": "JobCruiser/1.0",
                "Accept": "application/json"
            })
        self.base_url = "https://api.lever.co/v0/postings"

    def board_exists(self, company: str) -> bool:
        """
        Check if a Lever job board exists for the given company slug.
        """
        url = f"{self.base_url}/{company}?limit=1"
        try:
            response = self.session.get(url, timeout=20)
            return response.status_code == 200
        except Exception:
            return False

    def get_jobs(self, company: str) -> list:
        """
        Fetch and normalize all active jobs from the Lever postings API.
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
        for job_data in data:
            description_html = job_data.get("description", "")
            description_text = ""
            if description_html:
                soup = BeautifulSoup(description_html, "html.parser")
                description_text = soup.get_text(separator=" ", strip=True)

            list_contents = []
            for item in job_data.get("lists", []):
                item_title = item.get("text", "")
                item_html = item.get("content", "")
                if item_html:
                    soup_item = BeautifulSoup(item_html, "html.parser")
                    list_contents.append(f"{item_title}\n{soup_item.get_text(separator=' ', strip=True)}")

            if list_contents:
                description_text += "\n\n" + "\n\n".join(list_contents)

            created_at_ms = job_data.get("createdAt")
            updated_at = ""
            if created_at_ms:
                updated_at = datetime.datetime.fromtimestamp(
                    created_at_ms / 1000.0,
                    datetime.timezone.utc
                ).isoformat().replace("+00:00", "Z")

            categories = job_data.get("categories", {})
            location = categories.get("location", "")
            
            departments = []
            dept = categories.get("department")
            if dept:
                departments.append(dept)
            team = categories.get("team")
            if team:
                departments.append(team)

            offices = []
            if location:
                offices.append(location)

            jobs.append({
                "job_id": job_data.get("id"),
                "title": job_data.get("title"),
                "updated_at": updated_at,
                "absolute_url": job_data.get("hostedUrl"),
                "location": location,
                "departments": departments,
                "offices": offices,
                "description_text": description_text
            })
        return jobs
