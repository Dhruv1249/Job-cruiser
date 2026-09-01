"""Calculates the next semantic version for Job Cruiser releases using base-12 patch and base-9 minor rollover."""

import json
import os
import sys
import urllib.request


def compute_next_version(previous_version_string: str) -> str:
    """Takes a version string and calculates the next version following rollover rules."""
    cleaned_version = previous_version_string.lstrip("vV").split("+")[0]
    segments = [int(segment) if segment.isdigit() else 0 for segment in cleaned_version.split(".")]
    while len(segments) < 3:
        segments.append(0)

    major, minor, patch = segments[0], segments[1], segments[2]

    if patch < 12:
        patch += 1
    elif minor < 9:
        minor += 1
        patch = 0
    else:
        major += 1
        minor = 0
        patch = 0

    return f"{major}.{minor}.{patch}"


def fetch_latest_release_tag(repository_name: str, authorization_token: str = "") -> str | None:
    """Fetches the latest published release tag from the GitHub Releases API."""
    request_headers = {"Accept": "application/vnd.github+json"}
    if authorization_token:
        request_headers["Authorization"] = f"token {authorization_token}"

    api_url = f"https://api.github.com/repos/{repository_name}/releases/latest"
    api_request = urllib.request.Request(api_url, headers=request_headers)

    try:
        with urllib.request.urlopen(api_request, timeout=10) as response_stream:
            release_payload = json.loads(response_stream.read().decode())
            return release_payload.get("tag_name")
    except Exception:
        return None


def read_base_version_from_pubspec(pubspec_file_path: str = "client/pubspec.yaml") -> str:
    """Reads the base version string from client pubspec.yaml."""
    try:
        with open(pubspec_file_path, "r", encoding="utf-8") as file_stream:
            for line_content in file_stream:
                if line_content.startswith("version:"):
                    return line_content.replace("version:", "").strip().split("+")[0]
    except Exception:
        pass
    return "1.0.0"


def main() -> None:
    """Main entrypoint calculating and exporting next release version."""
    target_repository = os.environ.get("GITHUB_REPOSITORY", "Dhruv1249/Job-cruiser")
    github_access_token = os.environ.get("GITHUB_TOKEN", "")

    latest_tag = fetch_latest_release_tag(target_repository, github_access_token)
    if not latest_tag:
        latest_tag = read_base_version_from_pubspec()

    next_version = compute_next_version(latest_tag)
    current_build_number = os.environ.get("GITHUB_RUN_NUMBER", "1")

    github_output_path = os.environ.get("GITHUB_OUTPUT")
    if github_output_path:
        with open(github_output_path, "a", encoding="utf-8") as output_stream:
            output_stream.write(f"name={next_version}\n")
            output_stream.write(f"code={current_build_number}\n")
            output_stream.write(f"tag=v{next_version}\n")

    print(f"Calculated next version: v{next_version} (previous: {latest_tag})")


if __name__ == "__main__":
    main()
