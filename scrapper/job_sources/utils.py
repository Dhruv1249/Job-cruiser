"""
Utility functions for the job scraper, including custom configuration parsers.
"""

def load_yaml_config(file_path: str) -> dict:
    """
    Load and parse a basic YAML file containing key-value lists of strings.
    """
    config = {}
    current_key = None
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                if line.endswith(":"):
                    current_key = line[:-1].strip()
                    config[current_key] = []
                elif line.startswith("-") and current_key is not None:
                    value = line[1:].strip().strip('"').strip("'")
                    config[current_key].append(value)
    except Exception:
        return {}
    return config
