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


if __name__ == "__main__":
    test_patch_increment_within_range()
    test_patch_rollover_to_minor()
    test_minor_increment_within_range()
    test_major_rollover_from_max_minor_and_patch()
    print("All calculate_next_version tests passed successfully!")
