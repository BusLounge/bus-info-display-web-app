"""Populate NULL route_segments.road_type values from OpenStreetMap Overpass."""

from __future__ import annotations

import argparse
import logging
import math
import os
import time
from collections import Counter
from pathlib import Path
from typing import Any

import pandas as pd
import psycopg2
import requests
from dotenv import load_dotenv
from psycopg2.extras import execute_values

DEFAULT_OVERPASS_URL = "https://overpass-api.de/api/interpreter"
DEFAULT_BATCH_SIZE = 50
DEFAULT_REQUEST_DELAY = 1.1
DEFAULT_HTTP_TIMEOUT = 30

# These are the road_type values accepted by route_segments.road_type.
OSM_TO_APP_ROAD_TYPE = {
    "motorway": "highway",
    "motorway_link": "highway",
    "trunk": "highway",
    "trunk_link": "highway",
    "primary": "highway",
    "primary_link": "highway",
    "secondary": "suburban",
    "secondary_link": "suburban",
    "tertiary": "suburban",
    "tertiary_link": "suburban",
    "residential": "urban",
    "living_street": "urban",
    "service": "urban",
    "pedestrian": "urban",
    "unclassified": "rural",
    "track": "rural",
    "road": "mixed",
}

logger = logging.getLogger("update_route_segment_road_types")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Populate NULL route segment road_type values from Overpass."
    )
    parser.add_argument(
        "--database-url",
        default=os.getenv("DATABASE_URL"),
        help="PostgreSQL connection URL; defaults to DATABASE_URL.",
    )
    parser.add_argument(
        "--overpass-url",
        default=os.getenv("OVERPASS_URL", DEFAULT_OVERPASS_URL),
        help=f"Overpass endpoint (default: {DEFAULT_OVERPASS_URL}).",
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=DEFAULT_BATCH_SIZE,
        help=f"Rows fetched and committed per batch (default: {DEFAULT_BATCH_SIZE}).",
    )
    parser.add_argument(
        "--delay",
        type=float,
        default=DEFAULT_REQUEST_DELAY,
        help=f"Seconds between Overpass requests (default: {DEFAULT_REQUEST_DELAY}).",
    )
    parser.add_argument(
        "--radius-meters",
        type=int,
        default=100,
        help="Search radius around each endpoint (default: 100).",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Query and print proposed changes without updating PostgreSQL.",
    )
    args = parser.parse_args()
    if not args.database_url:
        parser.error("DATABASE_URL or --database-url is required")
    if args.batch_size < 1 or args.delay < 0 or args.radius_meters < 1:
        parser.error("batch size and radius must be positive; delay cannot be negative")
    return args


def validate_coordinate(latitude: Any, longitude: Any) -> bool:
    return (
        pd.notna(latitude)
        and pd.notna(longitude)
        and math.isfinite(float(latitude))
        and math.isfinite(float(longitude))
        and -90 <= float(latitude) <= 90
        and -180 <= float(longitude) <= 180
    )


def map_osm_road_types(tags: list[str]) -> str | None:
    mapped = [OSM_TO_APP_ROAD_TYPE[tag] for tag in tags if tag in OSM_TO_APP_ROAD_TYPE]
    if not mapped:
        return None

    counts = Counter(mapped)
    if len(counts) == 1:
        return mapped[0]
    if "highway" in counts and counts["highway"] >= max(counts.values()):
        return "highway"
    return "mixed"


def query_overpass(
    session: requests.Session,
    endpoint: str,
    start_latitude: float,
    start_longitude: float,
    end_latitude: float,
    end_longitude: float,
    radius_meters: int,
    timeout: int,
) -> str | None:
    query = f"""
[out:json][timeout:20];
(
  way(around:{radius_meters},{start_latitude},{start_longitude})[highway];
  way(around:{radius_meters},{end_latitude},{end_longitude})[highway];
);
out tags;
"""
    response = session.post(
        endpoint,
        data={"data": query},
        headers={"User-Agent": "bus-info-display-road-type-updater/1.0"},
        timeout=timeout,
    )
    response.raise_for_status()
    payload = response.json()
    elements = payload.get("elements", [])
    osm_tags = [
        element.get("tags", {}).get("highway")
        for element in elements
        if element.get("tags", {}).get("highway")
    ]
    return map_osm_road_types(osm_tags)


def fetch_null_segments(connection: Any) -> pd.DataFrame:
    query = """
        SELECT id, start_latitude, start_longitude, end_latitude, end_longitude
        FROM route_segments
        WHERE road_type IS NULL
        ORDER BY id
    """
    return pd.read_sql_query(query, connection)


def update_road_types(connection: Any, updates: list[tuple[str, str]]) -> None:
    if not updates:
        return
    with connection.cursor() as cursor:
        execute_values(
            cursor,
            """
            UPDATE route_segments AS rs
            SET road_type = values.road_type, updated_at = NOW()
                        FROM (VALUES %s) AS incoming(id, road_type)
                        WHERE rs.id = incoming.id::uuid
                            AND rs.road_type IS NULL
            """,
            updates,
            template="(%s::uuid, %s)",
        )


def main() -> int:
    load_dotenv(Path(__file__).resolve().parents[1] / "backend" / ".env")
    args = parse_args()
    logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")

    connection = psycopg2.connect(args.database_url)
    session = requests.Session()
    processed = 0
    updated = 0
    failed = 0

    try:
        all_segments = fetch_null_segments(connection)
        if all_segments.empty:
            logger.info("No route segments with NULL road_type were found")
            return 0

        for batch_start in range(0, len(all_segments), args.batch_size):
            segments = all_segments.iloc[batch_start : batch_start + args.batch_size]
            batch_updates: list[tuple[str, str]] = []
            for row in segments.itertuples(index=False):
                if not all(
                    validate_coordinate(latitude, longitude)
                    for latitude, longitude in (
                        (row.start_latitude, row.start_longitude),
                        (row.end_latitude, row.end_longitude),
                    )
                ):
                    logger.error("Skipping %s: invalid coordinates", row.id)
                    failed += 1
                    continue

                try:
                    road_type = query_overpass(
                        session,
                        args.overpass_url,
                        float(row.start_latitude),
                        float(row.start_longitude),
                        float(row.end_latitude),
                        float(row.end_longitude),
                        args.radius_meters,
                        DEFAULT_HTTP_TIMEOUT,
                    )
                    if road_type is None:
                        logger.warning("No mapped highway found for %s", row.id)
                        failed += 1
                    else:
                        batch_updates.append((str(row.id), road_type))
                        logger.info("%s -> %s", row.id, road_type)
                except requests.RequestException as error:
                    logger.error("Overpass request failed for %s: %s", row.id, error)
                    failed += 1
                except (ValueError, KeyError, TypeError) as error:
                    logger.error("Invalid Overpass response for %s: %s", row.id, error)
                    failed += 1

                processed += 1
                if args.delay:
                    time.sleep(args.delay)

            if args.dry_run:
                connection.rollback()
            else:
                update_road_types(connection, batch_updates)
                connection.commit()

            updated += len(batch_updates)
            logger.info(
                "Batch complete: fetched=%d updated=%d failed=%d",
                len(segments),
                len(batch_updates),
                failed,
            )

        logger.info(
            "Finished: processed=%d updated=%d failed=%d%s",
            processed,
            updated,
            failed,
            " (dry run)" if args.dry_run else "",
        )
        return 0
    finally:
        session.close()
        connection.close()


if __name__ == "__main__":
    raise SystemExit(main())
