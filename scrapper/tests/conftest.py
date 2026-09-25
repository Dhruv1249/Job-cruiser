"""
Pytest configuration and test environment initialization for scraper tests.
"""

import os


def pytest_configure(config):
    """
    Initializes mock environment variables for unit testing if not present in the environment.
    """
    os.environ.setdefault("BACKEND_API_URL", "http://localhost:8080/api")
    os.environ.setdefault("INGEST_API_KEY", "mock-ingest-key-for-unit-testing")
    os.environ.setdefault("GEMINI_API_KEY", "mock-gemini-key-for-unit-testing")
