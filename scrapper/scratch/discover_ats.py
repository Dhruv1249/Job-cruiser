"""
Probes and discovers ATS slugs for target companies using API checks and DuckDuckGo HTML search fallback.
"""

import sys
import urllib.parse
from pathlib import Path
import requests
from bs4 import BeautifulSoup
from job_sources.utils import load_yaml_config

COMPANIES = [
    "Aptos Labs", "Astral", "Ava Labs", "Buoyant", "Canonical", "Chainguard", "Chainlink Labs",
    "Cloudflare", "Datadog", "Deno", "Docker", "Ferrous Systems", "Flashbots", "Fly.io", "GitHub",
    "Grafana Labs", "Greptime", "HashiCorp", "Isovalent", "Kubecost", "Loft Labs", "Materialize",
    "Matter Labs", "Mirantis", "Nutanix", "Oxide Computer", "Parity Technologies", "Pulumi",
    "Red Hat", "Solo.io", "SUSE", "TigerBeetle", "Zed Industries", "Neon", "PingCAP", "PostHog",
    "Tailscale", "ClickHouse", "Cockroach Labs", "Confluent", "Elastic", "Ethereum Foundation",
    "Kraken", "MongoDB", "Mysten Labs", "Offchain Labs", "PlanetScale", "ScyllaDB", "Supabase",
    "Turso", "Qdrant", "Temporal.io", "Together AI", "Aiven", "Akuity", "DigitalOcean", "Fastly",
    "GitLab", "Harness", "Humanitec", "JetBrains", "Netlify", "Sourcegraph", "Vercel"
]

def extract_slug_from_url(url: str) -> tuple[str, str] | None:
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

def check_ats_validity(platform: str, slug: str, session: requests.Session) -> bool:
    """
    Validate if a given company slug is active on the specified ATS platform.
    """
    if not slug:
        return False
        
    if platform == "greenhouse":
        url = f"https://boards-api.greenhouse.io/v1/boards/{slug}"
        try:
            return session.get(url, timeout=10).status_code == 200
        except Exception:
            return False
            
    if platform == "lever":
        url = f"https://api.lever.co/v0/postings/{slug}?limit=1"
        try:
            return session.get(url, timeout=10).status_code == 200
        except Exception:
            return False
            
    if platform == "ashby":
        url = f"https://api.ashbyhq.com/posting-api/job-board/{slug}"
        try:
            return session.get(url, timeout=10).status_code == 200
        except Exception:
            return False
            
    if platform == "smartrecruiters":
        url = f"https://api.smartrecruiters.com/v1/companies/{slug}/postings?limit=1"
        try:
            return session.get(url, timeout=10).status_code == 200
        except Exception:
            return False
            
    if platform == "workday":
        url = f"https://{slug}.wd5.myworkdaysite.com/wday/cxs/{slug}/External/jobs"
        payload = {"limit": 1, "offset": 0, "searchText": ""}
        try:
            return session.post(url, json=payload, timeout=10).status_code == 200
        except Exception:
            return False
            
    return False

def search_duckduckgo_for_ats(company: str, session: requests.Session) -> tuple[str, str] | None:
    """
    Search DuckDuckGo HTML for Greenhouse, Lever, Ashby, or Workday URLs for the company.
    """
    queries = [
        f"{company} careers greenhouse",
        f"{company} careers lever",
        f"{company} careers ashby",
        f"{company} careers workday"
    ]
    
    headers = {
        "User-Agent": "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0"
    }

    for query in queries:
        url = f"https://html.duckduckgo.com/html/?q={urllib.parse.quote(query)}"
        try:
            resp = session.get(url, headers=headers, timeout=15)
            if resp.status_code != 200:
                continue
                
            soup = BeautifulSoup(resp.content, "html.parser")
            for link in soup.find_all("a", href=True):
                href = link["href"]
                # Handle DuckDuckGo redirect link wrapping
                if "uddg=" in href:
                    parsed = urllib.parse.urlparse(href)
                    query_params = urllib.parse.parse_qs(parsed.query)
                    actual_url = query_params.get("uddg", [""])[0]
                else:
                    actual_url = href

                if actual_url:
                    res = extract_slug_from_url(actual_url)
                    if res:
                        platform, slug = res
                        if check_ats_validity(platform, slug, session):
                            return platform, slug
        except Exception:
            pass
            
    return None

def main():
    """
    Resolve ATS slugs for the 64 target companies and write them to companies.yaml.
    """
    session = requests.Session()
    session.headers.update({
        "User-Agent": "JobCruiser/1.0"
    })

    config_path = Path(__file__).resolve().parent.parent / "companies.yaml"
    existing_config = load_yaml_config(str(config_path))
    
    for key in ["greenhouse", "lever", "ashby", "smartrecruiters", "workday"]:
        if key not in existing_config:
            existing_config[key] = []

    resolved = {}
    failed = []

    print(f"Resolving ATS slugs for {len(COMPANIES)} companies...")

    for name in COMPANIES:
        # Check if already resolved in config
        name_clean = name.strip()
        name_lower = name_clean.lower()
        
        found_in_config = False
        for platform in ["greenhouse", "lever", "ashby", "smartrecruiters", "workday"]:
            # Check if any slug in config matches
            for slug in existing_config[platform]:
                if name_lower in slug.lower() or slug.lower() in name_lower:
                    resolved[name_clean] = (platform, slug)
                    found_in_config = True
                    break
            if found_in_config:
                break
                
        if found_in_config:
            print(f"[{name_clean}] Already registered in companies.yaml -> ({resolved[name_clean][0]}: {resolved[name_clean][1]})")
            continue

        # Try Tier 1: Candidate variations
        slug_candidates = [
            name_lower.replace(" ", "").replace("-", ""),
            name_lower.replace(" ", "-"),
            name_lower.split()[0] if name_lower.split() else ""
        ]
        
        found = False
        for candidate in slug_candidates:
            if not candidate:
                continue
            for platform in ["greenhouse", "lever", "ashby", "smartrecruiters", "workday"]:
                if check_ats_validity(platform, candidate, session):
                    resolved[name_clean] = (platform, candidate)
                    existing_config[platform].append(candidate)
                    found = True
                    break
            if found:
                break
                
        if found:
            print(f"[{name_clean}] Resolved via direct candidate probing -> ({resolved[name_clean][0]}: {resolved[name_clean][1]})")
            continue

        # Try Tier 2: DuckDuckGo fallback
        res = search_duckduckgo_for_ats(name_clean, session)
        if res:
            platform, slug = res
            resolved[name_clean] = (platform, slug)
            existing_config[platform].append(slug)
            print(f"[{name_clean}] Resolved via DuckDuckGo search fallback -> ({platform}: {slug})")
        else:
            failed.append(name_clean)
            print(f"[{name_clean}] FAILED to resolve ATS mapping")

    # Save updated config
    with open(config_path, "w", encoding="utf-8") as f:
        for platform in ["greenhouse", "lever", "ashby", "smartrecruiters", "workday"]:
            f.write(f"{platform}:\n")
            slugs = sorted(list(set(existing_config[platform])))
            for slug in slugs:
                if slug:
                    f.write(f"  - {slug}\n")
            f.write("\n")

    print("\nResolution Summary:")
    print(f"  Total companies: {len(COMPANIES)}")
    print(f"  Resolved: {len(resolved)}")
    print(f"  Failed ({len(failed)}): {failed}")

if __name__ == "__main__":
    main()
