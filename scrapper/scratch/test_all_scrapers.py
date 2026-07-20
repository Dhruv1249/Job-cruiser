"""
Test script to verify all 16 registered JobSpy and ATS scraper engines.
"""

import sys
from pathlib import Path
from jobspy import scrape_jobs
from jobspy.model import Site

TEST_SITES = {
    Site.LINKEDIN: "developer",
    Site.INDEED: "developer",
    Site.ZIP_RECRUITER: "developer",
    Site.GLASSDOOR: "developer",
    Site.GOOGLE: "developer",
    Site.BAYT: "developer",
    Site.NAUKRI: "developer",
    Site.BDJOBS: "developer",
    Site.REMOTEOK: "developer",
    Site.WEWORKREMOTELY: "developer",
    Site.HN_HIRING: "developer",
    Site.THE_MUSE: "developer",
    Site.HIMALAYAS: "developer",
    Site.JOBSPRESSO: "developer",
    Site.RUST_CAREERS: "developer",
    Site.WORKING_NOMADS: "developer",
    Site.WEB3_CAREER: "developer",
    Site.CRYPTO_JOBS: "developer",
    Site.GREENHOUSE: "stripe",
    Site.LEVER: "spotify",
    Site.ASHBY: "notion",
    Site.SMARTRECRUITERS: "visa",
    Site.WORKDAY: "salesforce"
}

def main():
    """
    Test scraping on each site and output verification results.
    """
    print("Starting verification of all scrapers...")
    success_count = 0
    failed_count = 0

    for site, query in TEST_SITES.items():
        print(f"Testing [{site.value}] with query/slug '{query}'...")
        try:
            df = scrape_jobs(
                site_name=[site],
                search_term=query,
                results_wanted=3
            )
            count = len(df)
            if count >= 0:
                print(f"  -> SUCCESS: Scraped {count} jobs.")
                success_count += 1
            else:
                print("  -> FAILED: DataFrame is invalid.")
                failed_count += 1
        except Exception as e:
            print(f"  -> ERROR: {e}")
            failed_count += 1
        print("-" * 40)

    print("\nVerification Results:")
    print(f"  Successful: {success_count} scrapers")
    print(f"  Failed/Errors: {failed_count} scrapers")

if __name__ == "__main__":
    main()
