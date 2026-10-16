"""Unit tests for calculate_next_version rollover logic."""

from calculate_next_version import compute_next_version


def test_patch_increment_within_range():
    assert compute_next_version("1.0.0") == "1.0.1"
    assert compute_next_version("1.0.1") == "1.0.2"
    assert compute_next_version("1.0.11") == "1.0.12"
    assert compute_next_version("v1.0.5") == "1.0.6"


def test_patch_rollover_to_minor():
    assert compute_next_version("1.0.12") == "1.1.0"
    assert compute_next_version("v1.0.12") == "1.1.0"
    assert compute_next_version("1.1.12") == "1.2.0"
    assert compute_next_version("1.8.12") == "1.9.0"


def test_minor_increment_within_range():
    assert compute_next_version("1.1.0") == "1.1.1"
    assert compute_next_version("1.9.0") == "1.9.1"
    assert compute_next_version("1.9.11") == "1.9.12"


def test_major_rollover_from_max_minor_and_patch():
    assert compute_next_version("1.9.12") == "2.0.0"
    assert compute_next_version("v1.9.12") == "2.0.0"
    assert compute_next_version("2.9.12") == "3.0.0"



def test_parse_semantic_parts():
    """Verifies that parse_semantic_parts parses valid version tags and rejects invalid or extension tags."""
    from calculate_next_version import parse_semantic_parts

    assert parse_semantic_parts("1.3.2") == (1, 3, 2)
    assert parse_semantic_parts("v1.3.3") == (1, 3, 3)
    assert parse_semantic_parts("v1.3.3+12") == (1, 3, 3)
    assert parse_semantic_parts("extension-v0.1.2") is None
    assert parse_semantic_parts("v0.1") is None
    assert parse_semantic_parts("random-tag") is None


def test_filter_highest_client_release():
    """Verifies that filter_highest_client_release extracts only client tags and picks the highest semantic version."""
    from calculate_next_version import filter_highest_client_release

    mock_releases = [
        {"tag_name": "extension-v0.1.2", "draft": False, "prerelease": False},
        {"tag_name": "extension-v0.1.1", "draft": False, "prerelease": False},
        {"tag_name": "v1.3.2", "draft": False, "prerelease": False},
        {"tag_name": "v1.3.1", "draft": False, "prerelease": False},
        {"tag_name": "v1.3.0", "draft": False, "prerelease": False},
        {"tag_name": "v1.4.0-draft", "draft": True, "prerelease": False},
    ]

    selected_tag = filter_highest_client_release(mock_releases)
    assert selected_tag == "v1.3.2"


if __name__ == "__main__":
    test_patch_increment_within_range()
    test_patch_rollover_to_minor()
    test_minor_increment_within_range()
    test_major_rollover_from_max_minor_and_patch()
    test_parse_semantic_parts()
    test_filter_highest_client_release()
    print("All calculate_next_version tests passed successfully!")
