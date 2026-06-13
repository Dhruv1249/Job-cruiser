from pathlib import Path

BASE_URL = "https://boards-api.greenhouse.io/v1/boards"

DATA_DIR = Path("scrapper/data")

MAX_WORKERS = 10

REQUEST_TIMEOUT = 60

REQUEST_DELAY = 0.25

USER_AGENT = "JobCruiser/1.0"

RETRY_COUNT = 5

COMPANIES_FILE = "companies.txt"