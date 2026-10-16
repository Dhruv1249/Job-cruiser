"""Calculates the next semantic version for Job Cruiser releases using base-12 patch and base-9 minor rollover."""

import json
import os
import sys
import urllib.request


def parse_semantic_parts(version_string: str) -> tuple[int, int, int] | None:
    """Extracts major, minor, and patch numbers from a valid client version string."""
    cleaned_string = version_string.strip()
    if cleaned_string.startswith("v") or cleaned_string.startswith("V"):
        cleaned_string = cleaned_string[1:]
    cleaned_string = cleaned_string.split("+")[0]
    version_segments = cleaned_string.split(".")
    if len(version_segments) != 3:
        return None
    if not all(segment.isdigit() for segment in version_segments):
        return None
    return int(version_segments[0]), int(version_segments[1]), int(version_segments[2])


def filter_highest_client_release(releases_payload: list[dict]) -> str | None:
    """Filters client releases and returns the tag corresponding to the highest semantic version."""
    candidate_versions: list[tuple[tuple[int, int, int], str]] = []
    for item in releases_payload:
        if not isinstance(item, dict):
            continue
        if item.get("draft", False) or item.get("prerelease", False):
            continue
        tag_name = item.get("tag_name", "")
        if tag_name.startswith("extension-"):
            continue
        parsed_tuple = parse_semantic_parts(tag_name)
        if parsed_tuple is not None:
            candidate_versions.append((parsed_tuple, tag_name))
    if not candidate_versions:
        return None
    candidate_versions.sort(key=lambda element: element[0])
    return candidate_versions[-1][1]


def compute_next_version(previous_version_string: str) -> str:
    """Takes a version string and calculates the next version following rollover rules."""
    parsed_parts = parse_semantic_parts(previous_version_string)
    if parsed_parts is not None:
        major, minor, patch = parsed_parts
    else:
        major, minor, patch = 1, 0, 0

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
    """Fetches the latest published client release tag from GitHub Releases."""
    request_headers = {
        "Accept": "application/vnd.github+json",
        "User-Agent": "JobCruiser-Version-Calculator",
    }
    if authorization_token:
        request_headers["Authorization"] = f"token {authorization_token}"

    api_url = f"https://api.github.com/repos/{repository_name}/releases?per_page=100"
    api_request = urllib.request.Request(api_url, headers=request_headers)

    try:
        with urllib.request.urlopen(api_request, timeout=10) as response_stream:
            releases_payload = json.loads(response_stream.read().decode())
            if not isinstance(releases_payload, list):
                return None
            return filter_highest_client_release(releases_payload)
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
