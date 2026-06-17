import os
from pathlib import Path

BASE_URL = "https://boards-api.greenhouse.io/v1/boards"

DATA_DIR = Path(__file__).resolve().parent / "data"

MAX_WORKERS = 10

REQUEST_TIMEOUT = 60

REQUEST_DELAY = 0.25

USER_AGENT = "JobCruiser/1.0"

RETRY_COUNT = 5

COMPANIES_FILE = "companies.txt"

# Ingestion configuration for serverless deployment
BACKEND_API_URL = os.environ.get("BACKEND_API_URL", "http://localhost:8080/api")
INGEST_API_KEY = os.environ.get("INGEST_API_KEY", "dev-ingest-key-12345")