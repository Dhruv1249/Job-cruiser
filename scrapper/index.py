import json
from pathlib import Path
from datetime import datetime
from config import DATA_DIR

#DATA_DIR = Path("data")

companies = []

for folder in DATA_DIR.iterdir():

    if not folder.is_dir():
        continue

    company_file = (
        folder /
        "company.json"
    )

    if not company_file.exists():
        continue

    with open(
        company_file,
        "r",
        encoding="utf-8"
    ) as f:

        companies.append(
            json.load(f)
        )

index = {

    "generated_at":
        datetime.utcnow()
        .isoformat()
        + "Z",

    "companies":
        companies
}

with open(
    DATA_DIR /
    "index.json",
    "w",
    encoding="utf-8"
) as f:

    json.dump(
        index,
        f,
        indent=2
    )

print(
    "index.json created"
)