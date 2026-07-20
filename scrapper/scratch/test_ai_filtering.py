"""
Test AI filtering pass using fixed Gemma 4 models on a sample of scraped raw jobs.
"""

import json
from pathlib import Path
from scrape_all import extract_and_filter_batch_with_gemma

def main():
    """
    Test Gemma 4 evaluation on 50 raw jobs from data/raw_jobs.json.
    """
    raw_path = Path("data/raw_jobs.json")
    if not raw_path.exists():
        print("data/raw_jobs.json not found.")
        return

    with open(raw_path, "r", encoding="utf-8") as f:
        jobs = json.load(f)

    print(f"Loaded {len(jobs)} raw jobs from {raw_path}.")
    sample_jobs = jobs[:50]
    
    batches = [sample_jobs[i:i + 5] for i in range(0, len(sample_jobs), 5)]
    print(f"Evaluating sample of {len(sample_jobs)} jobs across {len(batches)} batches using Gemma 4...")

    matched_jobs = []
    for idx, batch in enumerate(batches):
        res = extract_and_filter_batch_with_gemma(batch, idx)
        matched_jobs.extend(res)

    print(f"\nAI Filtering Complete! Matched {len(matched_jobs)} / {len(sample_jobs)} jobs.")
    print("-" * 50)
    for j in matched_jobs:
        print(f"[{j.get('source')}] {j.get('title')} @ {j.get('company')}")
        print(f"  Location: {j.get('location')}")
        print(f"  Seniority: {j.get('seniority')}")
        print(f"  Tech Stack: {', '.join(j.get('tech_stack', []))}")
        print(f"  Summary: {j.get('summary')}")
        print("-" * 50)

if __name__ == "__main__":
    main()
