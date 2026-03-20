#!/usr/bin/env python3
"""
Validate Solana stablecoin-flow Parquet exports.

What it checks:
- shard readability
- required columns present
- schema consistency
- total rows
- min/max block_timestamp
- null counts for key fields
- top mints
- sample rows

Example:
  python3 scripts/validate_solana_parquet.py \
    --dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/solana-data-tmp"

Dependencies:
  python3 -m pip install pyarrow
"""

from __future__ import annotations

import argparse
import collections
import math
import os
import sys
from pathlib import Path
from typing import Dict, List

import pyarrow as pa
import pyarrow.dataset as ds
import pyarrow.parquet as pq


REQUIRED_COLUMNS = [
    "block_timestamp",
    "tx_signature",
    "source",
    "destination",
    "authority",
    "token_amount",
    "decimals",
    "mint",
    "transfer_type",
]


def list_parquet_files(directory: Path) -> List[Path]:
    files = sorted(directory.glob("*.parquet"))
    return [f for f in files if f.is_file() and not f.name.startswith("._")]


def schema_signature(schema: pa.Schema) -> tuple:
    return tuple((field.name, str(field.type)) for field in schema)


def validate_files(files: List[Path]) -> Dict:
    schema_counts = collections.Counter()
    missing_columns_by_file = {}
    bad_files = []

    for file_path in files:
        try:
            pf = pq.ParquetFile(file_path)
            schema = pf.schema_arrow
            schema_counts[schema_signature(schema)] += 1

            cols = set(schema.names)
            missing = [c for c in REQUIRED_COLUMNS if c not in cols]
            if missing:
                missing_columns_by_file[str(file_path)] = missing

        except Exception as exc:  # noqa: BLE001
            bad_files.append((str(file_path), str(exc)))

    return {
        "schema_counts": schema_counts,
        "missing_columns_by_file": missing_columns_by_file,
        "bad_files": bad_files,
    }


def stream_stats(dataset: ds.Dataset) -> Dict:
    total_rows = 0
    min_ts = None
    max_ts = None
    mint_counter = collections.Counter()
    transfer_type_counter = collections.Counter()

    null_counts = collections.Counter()

    columns = [
        "block_timestamp",
        "tx_signature",
        "source",
        "destination",
        "authority",
        "token_amount",
        "decimals",
        "mint",
        "transfer_type",
    ]

    scanner = dataset.scanner(columns=columns, batch_size=50_000, use_threads=True)

    for batch in scanner.to_batches():
        pyd = batch.to_pydict()
        if not pyd:
            continue

        batch_size = len(pyd["block_timestamp"])
        total_rows += batch_size

        for i in range(batch_size):
            ts = pyd["block_timestamp"][i]
            mint = pyd["mint"][i]
            transfer_type = pyd["transfer_type"][i]

            if ts is not None:
                if min_ts is None or ts < min_ts:
                    min_ts = ts
                if max_ts is None or ts > max_ts:
                    max_ts = ts

            if mint:
                mint_counter[str(mint)] += 1
            if transfer_type:
                transfer_type_counter[str(transfer_type)] += 1

            for col in columns:
                if pyd[col][i] is None:
                    null_counts[col] += 1

    return {
        "total_rows": total_rows,
        "min_ts": min_ts,
        "max_ts": max_ts,
        "top_mints": mint_counter.most_common(10),
        "top_transfer_types": transfer_type_counter.most_common(10),
        "null_counts": dict(null_counts),
    }


def sample_rows(dataset: ds.Dataset, limit: int) -> List[Dict]:
    table = dataset.head(
        limit,
        columns=[
            "block_timestamp",
            "tx_signature",
            "source",
            "destination",
            "authority",
            "token_amount",
            "decimals",
            "mint",
            "transfer_type",
        ],
    )
    return table.to_pylist()


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate Solana Parquet export")
    parser.add_argument("--dir", required=True, help="Directory containing Parquet files")
    parser.add_argument("--sample", type=int, default=5, help="Number of sample rows to print")
    args = parser.parse_args()

    directory = Path(args.dir).expanduser().resolve()
    if not directory.exists() or not directory.is_dir():
        print(f"error: directory not found: {directory}", file=sys.stderr)
        return 1

    files = list_parquet_files(directory)
    if not files:
        print(f"error: no parquet files found in {directory}", file=sys.stderr)
        return 1

    print(f"Directory      : {directory}")
    print(f"Shard count    : {len(files)}")

    file_validation = validate_files(files)

    if file_validation["bad_files"]:
        print("\nUnreadable / corrupted files:")
        for path, err in file_validation["bad_files"][:20]:
            print(f"- {path}: {err}")
        if len(file_validation["bad_files"]) > 20:
            print(f"... and {len(file_validation['bad_files']) - 20} more")
        return 1

    if file_validation["missing_columns_by_file"]:
        print("\nFiles missing required columns:")
        for path, missing in list(file_validation["missing_columns_by_file"].items())[:20]:
            print(f"- {path}: missing {missing}")
        if len(file_validation["missing_columns_by_file"]) > 20:
            print(f"... and {len(file_validation['missing_columns_by_file']) - 20} more")
        return 1

    print("\nSchema variants:")
    for idx, (sig, count) in enumerate(file_validation["schema_counts"].items(), start=1):
        print(f"  Variant {idx}: {count} file(s)")
        for name, dtype in sig:
            print(f"    - {name}: {dtype}")

    parquet_files = [str(f) for f in files]
    dataset = ds.dataset(parquet_files, format="parquet")

    missing_dataset_cols = [c for c in REQUIRED_COLUMNS if c not in dataset.schema.names]
    if missing_dataset_cols:
        print(f"\nerror: dataset missing required columns: {missing_dataset_cols}", file=sys.stderr)
        return 1

    stats = stream_stats(dataset)

    print("\nSummary stats:")
    print(f"  Total rows           : {stats['total_rows']:,}")
    print(f"  Min block_timestamp  : {stats['min_ts']}")
    print(f"  Max block_timestamp  : {stats['max_ts']}")

    print("\nNull counts:")
    for col in REQUIRED_COLUMNS:
        count = stats["null_counts"].get(col, 0)
        pct = 0.0 if stats["total_rows"] == 0 else (count / stats["total_rows"]) * 100.0
        print(f"  {col:15s} {count:>12,}  ({pct:6.2f}%)")

    print("\nTop mints:")
    for mint, count in stats["top_mints"]:
        print(f"  {mint:44s} {count:>12,}")

    print("\nTop transfer types:")
    for transfer_type, count in stats["top_transfer_types"]:
        print(f"  {transfer_type:20s} {count:>12,}")

    rows = sample_rows(dataset, args.sample)
    print(f"\nSample rows ({len(rows)}):")
    for row in rows:
        print(row)

    print("\nVerdict:")
    print("  ✅ Files are readable")
    print("  ✅ Required columns are present")
    print("  ✅ Dataset is ready for Solana stablecoin-flow extraction work")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())