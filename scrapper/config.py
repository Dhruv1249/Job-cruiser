"""
Configuration settings for the job board scraper pipeline.
"""

import os
from pathlib import Path

env_file_path = Path(__file__).resolve().parent / ".env"
if env_file_path.exists():
    with open(env_file_path, "r", encoding="utf-8") as env_stream:
        for line in env_stream:
            stripped_line = line.strip()
            if stripped_line and not stripped_line.startswith("#"):
                key, val = stripped_line.split("=", 1)
                os.environ[key.strip()] = val.strip()

BASE_URL = "https://boards-api.greenhouse.io/v1/boards"
DATA_DIR = Path(__file__).resolve().parent / "data"
MAX_WORKERS = 20
REQUEST_TIMEOUT = 300
REQUEST_DELAY = 0.25
USER_AGENT = "JobCruiser/1.0"
RETRY_COUNT = 5
BACKEND_API_URL = os.environ.get("BACKEND_API_URL", "http://localhost:8080/api")
INGEST_API_KEY = os.environ.get("INGEST_API_KEY") or os.environ.get("INGEST_KEY", "dev-ingest-key-12345")


def parse_proxy_configuration(raw_proxy_string: str) -> list[str]:
    """
    Parses a comma-separated proxy configuration string into formatted proxy URLs.
    """
    if not raw_proxy_string:
        return []
    parsed_proxies = []
    for proxy_item in raw_proxy_string.split(","):
        cleaned_item = proxy_item.strip()
        if not cleaned_item:
            continue
        proxy_components = cleaned_item.split(":")
        if len(proxy_components) == 4:
            host, port, user, password = proxy_components
            parsed_proxies.append(f"{user}:{password}@{host}:{port}")
        else:
            parsed_proxies.append(cleaned_item)
    return parsed_proxies


PROXIES = parse_proxy_configuration(os.environ.get("PROXY_LIST", ""))