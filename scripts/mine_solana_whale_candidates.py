#!/usr/bin/env python3
"""
Mine candidate Solana addresses from the local whale stablecoin dump.

It ranks addresses by:
- source count / value
- destination count / value
- authority count / value
- unique counterparties
- total transfer activity

Example:
  python3 scripts/mine_solana_whale_candidates.py \
    --parquet-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/solana-data-tmp/whale-stablecoin-flows" \
    --top 50
"""

from __future__ import annotations

import argparse
import collections
from dataclasses import dataclass, field
from decimal import Decimal
from pathlib import Path
from typing import Dict, Set

import pyarrow.dataset as ds


@dataclass
class AddrStats:
    address: str
    source_count: int = 0
    destination_count: int = 0
    authority_count: int = 0
    source_value: Decimal = field(default_factory=lambda: Decimal("0"))
    destination_value: Decimal = field(default_factory=lambda: Decimal("0"))
    authority_value: Decimal = field(default_factory=lambda: Decimal("0"))
    counterparties: Set[str] = field(default_factory=set)
    mints: collections.Counter = field(default_factory=collections.Counter)

    @property
    def total_count(self) -> int:
        return self.source_count + self.destination_count + self.authority_count

    @property
    def total_value(self) -> Decimal:
        return self.source_value + self.destination_value + self.authority_value


def get_stats(stats: Dict[str, AddrStats], address: str) -> AddrStats:
    if address not in stats:
        stats[address] = AddrStats(address=address)
    return stats[address]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--parquet-dir", required=True)
    parser.add_argument("--top", type=int, default=50)
    args = parser.parse_args()

    dataset = ds.dataset(str(Path(args.parquet_dir).expanduser().resolve()), format="parquet")
    scanner = dataset.scanner(
        columns=[
            "source",
            "destination",
            "authority",
            "token_amount",
            "mint",
        ],
        batch_size=100_000,
        use_threads=True,
    )

    stats: Dict[str, AddrStats] = {}

    for batch in scanner.to_batches():
        pyd = batch.to_pydict()
        size = len(pyd["mint"])

        for i in range(size):
            source = pyd["source"][i]
            destination = pyd["destination"][i]
            authority = pyd["authority"][i]
            mint = pyd["mint"][i]
            value = pyd["token_amount"][i]
            value = Decimal(str(value)) if value is not None else Decimal("0")

            if source:
                s = get_stats(stats, source)
                s.source_count += 1
                s.source_value += value
                if destination:
                    s.counterparties.add(destination)
                if mint:
                    s.mints[mint] += 1

            if destination:
                d = get_stats(stats, destination)
                d.destination_count += 1
                d.destination_value += value
                if source:
                    d.counterparties.add(source)
                if mint:
                    d.mints[mint] += 1

            if authority:
                a = get_stats(stats, authority)
                a.authority_count += 1
                a.authority_value += value
                if source:
                    a.counterparties.add(source)
                if destination:
                    a.counterparties.add(destination)
                if mint:
                    a.mints[mint] += 1

    ranked = sorted(
        stats.values(),
        key=lambda x: (x.total_count, x.total_value, len(x.counterparties)),
        reverse=True,
    )

    print("\nTop overall candidates:\n")
    for row in ranked[: args.top]:
        top_mints = ", ".join(f"{mint}:{count}" for mint, count in row.mints.most_common(2))
        print(
            f"{row.address} | total_count={row.total_count} "
            f"| src={row.source_count} dst={row.destination_count} auth={row.authority_count} "
            f"| counterparties={len(row.counterparties)} "
            f"| total_value={row.total_value} "
            f"| mints={top_mints}"
        )

    print("\nTop source-heavy candidates:\n")
    for row in sorted(ranked, key=lambda x: (x.source_count, x.source_value), reverse=True)[: args.top]:
        print(
            f"{row.address} | source_count={row.source_count} "
            f"| source_value={row.source_value} "
            f"| counterparties={len(row.counterparties)}"
        )

    print("\nTop destination-heavy candidates:\n")
    for row in sorted(ranked, key=lambda x: (x.destination_count, x.destination_value), reverse=True)[: args.top]:
        print(
            f"{row.address} | destination_count={row.destination_count} "
            f"| destination_value={row.destination_value} "
            f"| counterparties={len(row.counterparties)}"
        )

    print("\nTop authority-heavy candidates:\n")
    for row in sorted(ranked, key=lambda x: (x.authority_count, x.authority_value), reverse=True)[: args.top]:
        print(
            f"{row.address} | authority_count={row.authority_count} "
            f"| authority_value={row.authority_value} "
            f"| counterparties={len(row.counterparties)}"
        )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())