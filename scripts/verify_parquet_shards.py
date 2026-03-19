#!/usr/bin/env python3
"""
Verify Parquet shard integrity and optionally delete corrupted files.

Typical usage:
  python3 scripts/verify_parquet_shards.py \
    --dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/eth-data-tmp"

Dry run only:
  python3 scripts/verify_parquet_shards.py --dir /path/to/parquet --dry-run

Delete corrupted shards:
  python3 scripts/verify_parquet_shards.py --dir /path/to/parquet --delete-bad

Dependencies:
  python3 -m pip install pyarrow tqdm
"""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

from pyarrow.parquet import read_metadata
from tqdm import tqdm


def iter_parquet_files(directory: Path) -> list[Path]:
    files = sorted(directory.glob("*.parquet"))
    return [f for f in files if not f.name.startswith("._") and f.is_file()]


def verify_file(file_path: Path) -> tuple[bool, str]:
    try:
        read_metadata(file_path)
        return True, ""
    except Exception as exc:  # noqa: BLE001
        return False, str(exc)


def main() -> int:
    parser = argparse.ArgumentParser(description="Verify Parquet shard integrity.")
    parser.add_argument(
        "--dir",
        required=True,
        help="Directory containing Parquet shards.",
    )
    parser.add_argument(
        "--delete-bad",
        action="store_true",
        help="Delete corrupted shards after verification.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Report corrupted shards without deleting them.",
    )
    args = parser.parse_args()

    directory = Path(args.dir).expanduser().resolve()
    if not directory.exists() or not directory.is_dir():
        print(f"error: directory not found: {directory}", file=sys.stderr)
        return 1

    files = iter_parquet_files(directory)
    if not files:
        print(f"error: no parquet files found in {directory}", file=sys.stderr)
        return 1

    print(f"Scanning {len(files)} shards in: {directory}")

    healthy_count = 0
    corrupted: list[tuple[Path, str]] = []

    for file_path in tqdm(files, desc="Checking integrity"):
        ok, error_message = verify_file(file_path)
        if ok:
            healthy_count += 1
        else:
            corrupted.append((file_path, error_message))

    deleted_count = 0
    if corrupted and args.delete_bad and not args.dry_run:
        for file_path, _ in corrupted:
            try:
                os.remove(file_path)
                deleted_count += 1
            except OSError as exc:
                print(f"warning: failed to delete {file_path}: {exc}", file=sys.stderr)

    print("\n" + "-" * 48)
    print(f"Healthy shards   : {healthy_count}")
    print(f"Corrupted shards : {len(corrupted)}")
    print(f"Deleted shards   : {deleted_count}")
    print("-" * 48)

    if corrupted:
        print("\nCorrupted files:")
        for file_path, error_message in corrupted[:20]:
            print(f"- {file_path.name}: {error_message}")
        if len(corrupted) > 20:
            print(f"... and {len(corrupted) - 20} more")

    if corrupted and not args.delete_bad:
        print("\nUse --delete-bad to remove corrupted shards.")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
