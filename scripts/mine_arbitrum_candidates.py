#!/usr/bin/env python3
from __future__ import annotations

import argparse
import csv
import json
import os
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
from pathlib import Path

from google.cloud import bigquery

DEFAULT_DATASET = "bigquery-public-data.goog_blockchain_arbitrum_one_us"
DEFAULT_LIMIT = 1000
DEFAULT_MIN_INTERACTIONS = 200
DEFAULT_MIN_COUNTERPARTIES = 25
DEFAULT_MAX_BYTES_BILLED = 12 * 1024 * 1024 * 1024  # 12 GiB


@dataclass
class CandidateRow:
    address: str
    interaction_count: int
    unique_counterparties: int
    inbound_count: int
    outbound_count: int


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Mine Arbitrum candidates from BigQuery")
    parser.add_argument("--project", default=os.getenv("GOOGLE_CLOUD_PROJECT", ""))
    parser.add_argument("--dataset", default=DEFAULT_DATASET)
    parser.add_argument("--window-start", required=True)
    parser.add_argument("--window-end", required=True)
    parser.add_argument("--limit", type=int, default=DEFAULT_LIMIT)
    parser.add_argument("--min-interactions", type=int, default=DEFAULT_MIN_INTERACTIONS)
    parser.add_argument("--min-counterparties", type=int, default=DEFAULT_MIN_COUNTERPARTIES)
    parser.add_argument("--max-bytes-billed", type=int, default=DEFAULT_MAX_BYTES_BILLED)
    parser.add_argument("--out", required=True)
    return parser.parse_args()


def build_query(dataset: str) -> str:
    return f"""
WITH tx_90d AS (
  SELECT
    from_address,
    to_address
  FROM `{dataset}.transactions`
  WHERE block_timestamp >= TIMESTAMP(@window_start)
    AND block_timestamp < TIMESTAMP(@window_end)
    AND from_address IS NOT NULL
    AND to_address IS NOT NULL
),
addr_flows AS (
  SELECT
    from_address AS address,
    to_address AS counterparty,
    "outbound" AS direction
  FROM tx_90d
  WHERE from_address != to_address

  UNION ALL

  SELECT
    to_address AS address,
    from_address AS counterparty,
    "inbound" AS direction
  FROM tx_90d
  WHERE from_address != to_address
),
agg AS (
  SELECT
    address,
    COUNT(*) AS interaction_count,
    COUNT(DISTINCT counterparty) AS unique_counterparties,
    COUNTIF(direction = "inbound") AS inbound_count,
    COUNTIF(direction = "outbound") AS outbound_count
  FROM addr_flows
  WHERE address IS NOT NULL
    AND counterparty IS NOT NULL
  GROUP BY address
)
SELECT
  address,
  interaction_count,
  unique_counterparties,
  inbound_count,
  outbound_count
FROM agg
WHERE interaction_count >= @min_interactions
  AND unique_counterparties >= @min_counterparties
ORDER BY interaction_count DESC
LIMIT @limit
"""


def main() -> None:
    args = parse_args()
    if not args.project:
        raise SystemExit("Missing --project and GOOGLE_CLOUD_PROJECT is not set")

    client = bigquery.Client(project=args.project)
    job_config = bigquery.QueryJobConfig(
        query_parameters=[
            bigquery.ScalarQueryParameter("window_start", "STRING", args.window_start),
            bigquery.ScalarQueryParameter("window_end", "STRING", args.window_end),
            bigquery.ScalarQueryParameter("min_interactions", "INT64", args.min_interactions),
            bigquery.ScalarQueryParameter("min_counterparties", "INT64", args.min_counterparties),
            bigquery.ScalarQueryParameter("limit", "INT64", args.limit),
        ],
        maximum_bytes_billed=args.max_bytes_billed,
        use_query_cache=True,
    )
    rows = client.query(build_query(args.dataset), job_config=job_config).result()

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    parsed = [
        CandidateRow(
            address=row["address"],
            interaction_count=int(row["interaction_count"]),
            unique_counterparties=int(row["unique_counterparties"]),
            inbound_count=int(row["inbound_count"]),
            outbound_count=int(row["outbound_count"]),
        )
        for row in rows
    ]

    with out_path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(
            handle,
            fieldnames=[
                "address",
                "interaction_count",
                "unique_counterparties",
                "inbound_count",
                "outbound_count",
            ],
        )
        writer.writeheader()
        for row in parsed:
            writer.writerow(asdict(row))

    manifest = {
        "script": "mine_arbitrum_candidates.py",
        "dataset": args.dataset,
        "window_start": args.window_start,
        "window_end": args.window_end,
        "row_count": len(parsed),
        "generated_at": datetime.now(timezone.utc).isoformat(),
    }
    out_path.with_suffix(".manifest.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
    print(f"Wrote {len(parsed)} candidates to {out_path}")


if __name__ == "__main__":
    main()