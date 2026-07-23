FROM python:3.11-slim

WORKDIR /app

# Install git for pip git dependencies and ca-certificates
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install all dependencies (including JobSpy fork from git)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy scrapper source files
COPY . .

# Execute scrape_all orchestrator on Cloud Run Job run
CMD ["python", "scrape_all.py"]
