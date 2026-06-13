import json
from pathlib import Path
from config import DATA_DIR

#DATA_DIR = Path("data")

keyword = input(
    "Enter keyword: "
).lower()

matches = []

for company_dir in DATA_DIR.iterdir():

    if not company_dir.is_dir():
        continue

    jobs_file = (
        company_dir /
        "jobs_flat.json"
    )

    if not jobs_file.exists():
        continue

    with open(
        jobs_file,
        "r",
        encoding="utf-8"
    ) as f:

        jobs = json.load(f)

    for job in jobs:

        text = (
            job.get(
                "title",
                ""
            )
            + " "
            + job.get(
                "description_text",
                ""
            )
        ).lower()

        if keyword in text:

            matches.append({

                "company":
                    company_dir.name,

                "title":
                    job["title"],

                "location":
                    job["location"],

                "url":
                    job[
                        "absolute_url"
                    ]
            })

print(
    f"\nFound "
    f"{len(matches)} jobs\n"
)

for item in matches[:50]:

    print(
        f"{item['company']} | "
        f"{item['title']} | "
        f"{item['location']}"
    )

    print(item["url"])

    print("-" * 50)