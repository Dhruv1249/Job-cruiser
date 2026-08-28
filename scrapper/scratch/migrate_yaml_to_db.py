"""
Migration script that transfers ATS platform slugs from companies.yaml into PostgreSQL.
"""

import os
from pathlib import Path
import psycopg2
import yaml


def migrate_yaml_slugs_to_database(yaml_file_path: Path, database_url: str) -> int:
    """
    Reads company ATS platform mappings from a YAML configuration file and inserts them into PostgreSQL.
    """
    with open(yaml_file_path, "r", encoding="utf-8") as file_stream:
        platform_mappings = yaml.safe_load(file_stream)

    database_connection = psycopg2.connect(database_url)
    database_cursor = database_connection.cursor()
    total_migrated_records = 0

    for platform_name, company_slugs in platform_mappings.items():
        for company_slug in company_slugs or []:
            if not company_slug or not isinstance(company_slug, str):
                continue
            cleaned_slug = company_slug.strip()
            if not cleaned_slug:
                continue
            database_cursor.execute(
                """
                INSERT INTO company_ats_boards (platform, slug)
                VALUES (%s, %s)
                ON CONFLICT (platform, slug) DO NOTHING
                """,
                (platform_name, cleaned_slug),
            )
            total_migrated_records += 1

    database_connection.commit()
    database_cursor.close()
    database_connection.close()
    return total_migrated_records


if __name__ == "__main__":
    target_yaml_path = Path(__file__).resolve().parent.parent / "companies.yaml"
    target_database_url = os.environ.get("DATABASE_URL")
    if not target_database_url:
        raise ValueError("DATABASE_URL environment variable must be set")
    migrated_count = migrate_yaml_slugs_to_database(target_yaml_path, target_database_url)
    print(f"Successfully processed and migrated {migrated_count} ATS slug entries into PostgreSQL.")
