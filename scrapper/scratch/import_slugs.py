"""
Seed companies.yaml with 12,000+ companies from outscal/OpenJobs database.
"""

import json
from pathlib import Path
import requests
from job_sources.utils import load_yaml_config

def extract_ats_slug(url: str) -> tuple[str, str] | None:
    """
    Extract ATS platform and company slug from a job board or career page URL.
    """
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

def main():
    """
    Download OpenJobs dataset and parse/merge company slugs into companies.yaml.
    """
    url = "https://raw.githubusercontent.com/outscal/OpenJobs/main/data/companies_v2.json"
    print(f"Downloading OpenJobs dataset from {url}...")
    try:
        resp = requests.get(url, timeout=60)
        if resp.status_code != 200:
            print(f"Failed to download OpenJobs dataset: HTTP {resp.status_code}")
            return
        companies_data = resp.json()
    except Exception as e:
        print(f"Error downloading OpenJobs dataset: {e}")
        return

    print(f"Downloaded {len(companies_data)} company entries. Parsing ATS links...")

    # Load existing config
    config_path = Path(__file__).resolve().parent.parent / "companies.yaml"
    existing_config = load_yaml_config(str(config_path))
    
    # Initialize keys if missing
    for k in ["greenhouse", "lever", "ashby", "smartrecruiters", "workday"]:
        if k not in existing_config:
            existing_config[k] = []
            
    added_counts = {"greenhouse": 0, "lever": 0, "ashby": 0, "smartrecruiters": 0, "workday": 0}

    for item in companies_data:
        ats_links = item.get("ats_links") or item.get("list_urls") or []
        for link in ats_links:
            res = extract_ats_slug(link)
            if res:
                platform, slug = res
                if slug and slug not in existing_config[platform]:
                    existing_config[platform].append(slug)
                    added_counts[platform] += 1

    # Save back to companies.yaml in standard format
    with open(config_path, "w", encoding="utf-8") as f:
        for platform in ["greenhouse", "lever", "ashby", "smartrecruiters", "workday"]:
            f.write(f"{platform}:\n")
            # Sort slugs alphabetically
            slugs = sorted(list(set(existing_config[platform])))
            for slug in slugs:
                if slug:
                    f.write(f"  - {slug}\n")
            f.write("\n")

    print("Successfully merged company slugs into companies.yaml:")
    for k, v in added_counts.items():
        print(f"  - {k}: added {v} slugs")

if __name__ == "__main__":
    main()
