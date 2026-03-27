#!/usr/bin/env python3
"""
Check whether tracked Solana addresses appear in the local whale stablecoin dump.

Example:
  python3 scripts/check_solana_addresses_in_dump.py \
    --parquet-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/solana-data-tmp" \
    --addresses data/candidates/solana_addresses.local.txt
"""

from __future__ import annotations

import argparse
from collections import defaultdict
from pathlib import Path
from typing import Dict, List

import pyarrow.dataset as ds


def load_addresses(path: str) -> List[str]:
    out = []
    seen = set()
    with open(path, "r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if line not in seen:
                seen.add(line)
                out.append(line)
    return out


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--parquet-dir", required=True)
    parser.add_argument("--addresses", required=True)
    args = parser.parse_args()

    addresses = load_addresses(args.addresses)
    if not addresses:
        print("No addresses loaded.")
        return 1

    parquet_dir = Path(args.parquet_dir).expanduser().resolve()
    dataset = ds.dataset(str(parquet_dir), format="parquet")

    columns = [
        "block_timestamp",
        "tx_signature",
        "source",
        "destination",
        "authority",
        "token_amount",
        "mint",
    ]

    scanner = dataset.scanner(columns=columns, batch_size=100_000, use_threads=True)

    stats: Dict[str, Dict[str, int]] = {
        a: {"source": 0, "destination": 0, "authority": 0, "any": 0}
        for a in addresses
    }

    mint_counts: Dict[str, Dict[str, int]] = {
        a: defaultdict(int)
        for a in addresses
    }

    for batch in scanner.to_batches():
        pyd = batch.to_pydict()
        size = len(pyd["tx_signature"])
        for i in range(size):
            source = pyd["source"][i]
            destination = pyd["destination"][i]
            authority = pyd["authority"][i]
            mint = pyd["mint"][i]

            for addr in addresses:
                matched = False
                if source == addr:
                    stats[addr]["source"] += 1
                    matched = True
                if destination == addr:
                    stats[addr]["destination"] += 1
                    matched = True
                if authority == addr:
                    stats[addr]["authority"] += 1
                    matched = True
                if matched:
                    stats[addr]["any"] += 1
                    if mint:
                        mint_counts[addr][mint] += 1

    print("\nAddress presence in Solana whale stablecoin dump:\n")
    for addr in addresses:
        s = stats[addr]
        print(addr)
        print(f"  any         : {s['any']}")
        print(f"  source      : {s['source']}")
        print(f"  destination : {s['destination']}")
        print(f"  authority   : {s['authority']}")
        top_mints = sorted(mint_counts[addr].items(), key=lambda x: -x[1])[:5]
        if top_mints:
            print("  top mints   :")
            for mint, count in top_mints:
                print(f"    {mint} -> {count}")
        print()

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
