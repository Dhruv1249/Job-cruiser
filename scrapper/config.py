"""
Configuration settings for the job board scraper pipeline.
"""

import os
from pathlib import Path

env_path = Path(__file__).resolve().parent / ".env"
if env_path.exists():
    with open(env_path, "r") as f:
        for line in f:
            stripped_line = line.strip()
            if stripped_line and not stripped_line.startswith("#"):
                key, val = stripped_line.split("=", 1)
                os.environ[key.strip()] = val.strip()


BASE_URL = "https://boards-api.greenhouse.io/v1/boards"

DATA_DIR = Path(__file__).resolve().parent / "data"

MAX_WORKERS = 10

REQUEST_TIMEOUT = 60

REQUEST_DELAY = 0.25

USER_AGENT = "JobCruiser/1.0"

RETRY_COUNT = 5

COMPANIES_FILE = "companies.txt"

BACKEND_API_URL = os.environ.get("BACKEND_API_URL", "http://localhost:8080/api")
INGEST_API_KEY = os.environ.get("INGEST_API_KEY", "dev-ingest-key-12345")
GEMINI_API_KEY = os.environ.get("GEMINI_API_KEY", "")

GEMMA_MOE_MODEL = "gemma-4-26b-a4b-it"
GEMMA_DENSE_MODEL = "gemma-4-31b-it"
DISABLE_AI_EXTRACTION = os.environ.get("DISABLE_AI_EXTRACTION", "false").lower() == "true"