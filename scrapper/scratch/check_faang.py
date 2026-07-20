"""
Check presence of FAANG and tech giant slugs in companies.yaml and add missing ones.
"""

from pathlib import Path
from job_sources.utils import load_yaml_config

TECH_GIANTS = {
    "greenhouse": [
        "meta", "airbnb", "stripe", "coinbase", "door-dash", "instacart", "robinhood",
        "reddit", "discord", "figma", "databricks", "snowflake", "palantir", "cloudflare",
        "datadog", "okta", "twilio", "zendesk", "square", "block", "uber"
    ],
    "lever": [
        "spotify", "netflix", "palantir", "asana", "atlassian", "lyft", "box"
    ],
    "ashby": [
        "notion", "linear", "ramp", "retool", "sentry", "vercel", "perplexity"
    ],
    "smartrecruiters": [
        "visa", "square", "bosch", "mcdonalds"
    ],
    "workday": [
        "salesforce", "workday", "adobe", "cisco", "nvidia", "walmart"
    ]
}

def main():
    """
    Ensure all tech giant slugs are present in companies.yaml.
    """
    config_path = Path(__file__).resolve().parent.parent / "companies.yaml"
    config = load_yaml_config(str(config_path))
    
    added_count = 0
    for platform, slugs in TECH_GIANTS.items():
        if platform not in config:
            config[platform] = []
        for slug in slugs:
            if slug not in config[platform]:
                config[platform].append(slug)
                added_count += 1

    with open(config_path, "w", encoding="utf-8") as f:
        for platform in ["greenhouse", "lever", "ashby", "smartrecruiters", "workday"]:
            f.write(f"{platform}:\n")
            slugs = sorted(list(set(config.get(platform, []))))
            for s in slugs:
                if s:
                    f.write(f"  - {s}\n")
            f.write("\n")

    print(f"Added {added_count} tech giant slugs to companies.yaml.")

if __name__ == "__main__":
    main()
