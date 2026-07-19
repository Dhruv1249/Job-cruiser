"""
Index builder script to compile details of all scraped companies into index.json.
"""

import json
from pathlib import Path
from datetime import datetime, timezone
from config import DATA_DIR

def build_index(data_dir: Path):
    """
    Scan the data directory and compile all company.json metadata.
    """
    companies = []
    for folder in data_dir.iterdir():
        if not folder.is_dir():
            continue
        company_file = folder / "company.json"
        if not company_file.exists():
            continue
        with open(company_file, "r", encoding="utf-8") as f:
            try:
                companies.append(json.load(f))
            except json.JSONDecodeError:
                pass

    index = {
        "generated_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "companies": companies
    }

    with open(data_dir / "index.json", "w", encoding="utf-8") as f:
        json.dump(index, f, indent=2)

if __name__ == "__main__":
    build_index(DATA_DIR)
    print("index.json created")