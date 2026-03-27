#!/usr/bin/env python3
"""
Extract address-scoped Solana stablecoin summaries from a local Parquet dump.

Input:
- local Parquet directory exported from BigQuery whale stablecoin flows
- text file with one Solana address per line

Output per tracked address:
- <address>.stablecoin.ndjson.gz
- <address>.json

Example:
  python3 scripts/extract_solana_stablecoin.py \
    --parquet-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/solana-data-tmp/whale-stablecoin-flows" \
    --addresses data/candidates/solana_addresses.local.txt \
    --out data/cases/extracted-solana-stablecoin
"""

from __future__ import annotations

import argparse
import datetime as dt
import gzip
import json
import os
from collections import Counter
from dataclasses import dataclass, field
from decimal import Decimal, InvalidOperation
from pathlib import Path
from typing import Dict, List, Optional, Set

import pyarrow.dataset as ds


USDC_MINT = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
USDT_MINT = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"


def now_iso() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def load_addresses(path: str) -> List[str]:
    out: List[str] = []
    seen: Set[str] = set()

    with open(path, "r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if line not in seen:
                seen.add(line)
                out.append(line)

    return out


def decimal_or_zero(value) -> Decimal:
    if value is None:
        return Decimal("0")
    try:
        return Decimal(str(value))
    except (InvalidOperation, ValueError):
        return Decimal("0")


def ui_amount(raw_amount: Decimal, decimals_value) -> str:
    decimals = int(decimal_or_zero(decimals_value))
    if decimals < 0:
        decimals = 0
    scale = Decimal(10) ** decimals
    if scale == 0:
        return str(raw_amount)
    return str(raw_amount / scale)


def iso_or_none(value) -> Optional[str]:
    if value is None:
        return None
    if isinstance(value, dt.datetime):
        if value.tzinfo is None:
            return value.isoformat() + "Z"
        return value.astimezone(dt.timezone.utc).isoformat().replace("+00:00", "Z")
    return str(value)


@dataclass
class CounterpartySummary:
    address: str
    interactions: int = 0
    inbound_count: int = 0
    outbound_count: int = 0
    authority_count: int = 0
    total_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))
    inbound_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))
    outbound_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))
    authority_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))
    mint_counts: Counter = field(default_factory=Counter)

    def to_json(self) -> dict:
        return {
            "address": self.address,
            "interactions": self.interactions,
            "inbound_count": self.inbound_count,
            "outbound_count": self.outbound_count,
            "authority_count": self.authority_count,
            "total_value_raw": str(self.total_value_raw),
            "inbound_value_raw": str(self.inbound_value_raw),
            "outbound_value_raw": str(self.outbound_value_raw),
            "authority_value_raw": str(self.authority_value_raw),
            "top_mints": [
                {"mint": mint, "count": count}
                for mint, count in self.mint_counts.most_common(3)
            ],
        }


@dataclass
class AuthorityPairSummary:
    key: str
    interactions: int = 0
    total_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))

    def to_json(self) -> dict:
        source, destination = self.key.split("->", 1)
        return {
            "source": source,
            "destination": destination,
            "interactions": self.interactions,
            "total_value_raw": str(self.total_value_raw),
        }


@dataclass
class StablecoinDatasetBuilder:
    address: str
    out_dir: str
    sample_limit: int
    top_limit: int

    generated_at: str = field(default_factory=now_iso)

    first_seen: Optional[str] = None
    last_seen: Optional[str] = None

    source_transfer_count: int = 0
    destination_transfer_count: int = 0
    authority_transfer_count: int = 0
    source_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))
    destination_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))
    authority_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))

    unique_counterparties: Set[str] = field(default_factory=set)
    source_counterparties: Set[str] = field(default_factory=set)
    destination_counterparties: Set[str] = field(default_factory=set)
    authority_counterparties: Set[str] = field(default_factory=set)

    mint_counts: Counter = field(default_factory=Counter)
    transfer_type_counts: Counter = field(default_factory=Counter)

    counterparties: Dict[str, CounterpartySummary] = field(default_factory=dict)
    authority_pairs: Dict[str, AuthorityPairSummary] = field(default_factory=dict)

    sample_transfers: List[dict] = field(default_factory=list)
    source_row_count: int = 0

    def __post_init__(self) -> None:
        os.makedirs(self.out_dir, exist_ok=True)
        self.raw_filename = f"{self.address}.stablecoin.ndjson.gz"
        self.raw_path = os.path.join(self.out_dir, self.raw_filename)
        self._fh = gzip.open(self.raw_path, "wt", encoding="utf-8")

    def close(self) -> None:
        self._fh.close()

    def _touch_time(self, block_timestamp) -> None:
        iso = iso_or_none(block_timestamp)
        if iso is None:
            return
        if self.first_seen is None or iso < self.first_seen:
            self.first_seen = iso
        if self.last_seen is None or iso > self.last_seen:
            self.last_seen = iso

    def _get_cp(self, address: str) -> CounterpartySummary:
        cp = self.counterparties.get(address)
        if cp is None:
            cp = CounterpartySummary(address=address)
            self.counterparties[address] = cp
        return cp

    def _record_counterparty(self, counterparty: str, role: str, value_raw: Decimal, mint: str) -> None:
        if not counterparty or counterparty == self.address:
            return

        self.unique_counterparties.add(counterparty)
        cp = self._get_cp(counterparty)
        cp.interactions += 1
        cp.total_value_raw += value_raw
        cp.mint_counts[mint] += 1

        if role == "source":
            cp.outbound_count += 1
            cp.outbound_value_raw += value_raw
            self.source_counterparties.add(counterparty)
        elif role == "destination":
            cp.inbound_count += 1
            cp.inbound_value_raw += value_raw
            self.destination_counterparties.add(counterparty)
        elif role == "authority":
            cp.authority_count += 1
            cp.authority_value_raw += value_raw
            self.authority_counterparties.add(counterparty)

    def _record_authority_pair(self, source: str, destination: str, value_raw: Decimal) -> None:
        key = f"{source}->{destination}"
        pair = self.authority_pairs.get(key)
        if pair is None:
            pair = AuthorityPairSummary(key=key)
            self.authority_pairs[key] = pair
        pair.interactions += 1
        pair.total_value_raw += value_raw

    def add_row(self, row: dict) -> None:
        self.source_row_count += 1
        self._touch_time(row.get("block_timestamp"))

        mint = row.get("mint") or ""
        transfer_type = row.get("transfer_type") or ""
        value_raw = decimal_or_zero(row.get("token_amount"))

        self.mint_counts[mint] += 1
        self.transfer_type_counts[transfer_type] += 1

        source = row.get("source") or ""
        destination = row.get("destination") or ""
        authority = row.get("authority") or ""

        matched = False

        if source == self.address:
            matched = True
            self.source_transfer_count += 1
            self.source_value_raw += value_raw
            self._record_counterparty(destination, "source", value_raw, mint)

        if destination == self.address:
            matched = True
            self.destination_transfer_count += 1
            self.destination_value_raw += value_raw
            self._record_counterparty(source, "destination", value_raw, mint)

        if authority == self.address:
            matched = True
            self.authority_transfer_count += 1
            self.authority_value_raw += value_raw
            self._record_counterparty(source, "authority", value_raw, mint)
            self._record_counterparty(destination, "authority", value_raw, mint)
            if source or destination:
                self._record_authority_pair(source, destination, value_raw)

        if not matched:
            return

        normalized = {
            "block_timestamp": iso_or_none(row.get("block_timestamp")),
            "tx_signature": row.get("tx_signature"),
            "source": source,
            "destination": destination,
            "authority": authority,
            "token_amount_raw": str(value_raw),
            "token_amount_ui": ui_amount(value_raw, row.get("decimals")),
            "decimals": int(decimal_or_zero(row.get("decimals"))),
            "mint": mint,
            "transfer_type": transfer_type,
            "matched_roles": {
                "source": source == self.address,
                "destination": destination == self.address,
                "authority": authority == self.address,
            },
        }

        if len(self.sample_transfers) < self.sample_limit:
            self.sample_transfers.append(normalized)

        self._fh.write(json.dumps(normalized, separators=(",", ":")) + "\n")

    def metadata(self) -> dict:
        top_counterparties = sorted(
            self.counterparties.values(),
            key=lambda x: (x.interactions, x.total_value_raw),
            reverse=True,
        )[: self.top_limit]

        top_authority_pairs = sorted(
            self.authority_pairs.values(),
            key=lambda x: (x.interactions, x.total_value_raw),
            reverse=True,
        )[: self.top_limit]

        return {
            "address": self.address,
            "chain": "SOLANA",
            "dataset_type": "stablecoin_whale_flows",
            "generated_at": self.generated_at,
            "summary": {
                "first_seen": self.first_seen,
                "last_seen": self.last_seen,
                "source_transfer_count": self.source_transfer_count,
                "destination_transfer_count": self.destination_transfer_count,
                "authority_transfer_count": self.authority_transfer_count,
                "source_value_raw": str(self.source_value_raw),
                "destination_value_raw": str(self.destination_value_raw),
                "authority_value_raw": str(self.authority_value_raw),
                "unique_counterparties": len(self.unique_counterparties),
                "source_counterparties": len(self.source_counterparties),
                "destination_counterparties": len(self.destination_counterparties),
                "authority_counterparties": len(self.authority_counterparties),
            },
            "mint_breakdown": [
                {"mint": mint, "count": count}
                for mint, count in self.mint_counts.most_common()
            ],
            "transfer_type_breakdown": [
                {"transfer_type": ttype, "count": count}
                for ttype, count in self.transfer_type_counts.most_common()
            ],
            "top_counterparties": [cp.to_json() for cp in top_counterparties],
            "top_authority_pairs": [pair.to_json() for pair in top_authority_pairs],
            "sample_transfers": self.sample_transfers,
            "source_row_count": self.source_row_count,
            "raw_flow_file": self.raw_filename,
        }


def main() -> int:
    parser = argparse.ArgumentParser(description="Extract address-scoped Solana stablecoin summaries")
    parser.add_argument("--parquet-dir", required=True, help="Directory containing local Parquet shards")
    parser.add_argument("--addresses", required=True, help="Text file with one Solana address per line")
    parser.add_argument("--out", required=True, help="Output directory")
    parser.add_argument("--sample", type=int, default=200, help="Sample transfers to keep per address")
    parser.add_argument("--top", type=int, default=20, help="Top counterparties / pairs to keep")
    args = parser.parse_args()

    addresses = load_addresses(args.addresses)
    if not addresses:
        print("error: no addresses loaded", file=os.sys.stderr)
        return 1

    dataset = ds.dataset(str(Path(args.parquet_dir).expanduser().resolve()), format="parquet")

    builders = {
        address: StablecoinDatasetBuilder(address, args.out, args.sample, args.top)
        for address in addresses
    }

    filter_expr = (
        ds.field("source").isin(addresses)
        | ds.field("destination").isin(addresses)
        | ds.field("authority").isin(addresses)
    )

    scanner = dataset.scanner(
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
        filter=filter_expr,
        batch_size=100_000,
        use_threads=True,
    )

    matched_rows = 0

    try:
        for batch in scanner.to_batches():
            pyd = batch.to_pydict()
            size = len(pyd["tx_signature"])

            for i in range(size):
                row = {
                    "block_timestamp": pyd["block_timestamp"][i],
                    "tx_signature": pyd["tx_signature"][i],
                    "source": pyd["source"][i],
                    "destination": pyd["destination"][i],
                    "authority": pyd["authority"][i],
                    "token_amount": pyd["token_amount"][i],
                    "decimals": pyd["decimals"][i],
                    "mint": pyd["mint"][i],
                    "transfer_type": pyd["transfer_type"][i],
                }

                source = row["source"]
                destination = row["destination"]
                authority = row["authority"]

                for address, builder in builders.items():
                    if source == address or destination == address or authority == address:
                        builder.add_row(row)

                matched_rows += 1
                if matched_rows % 250_000 == 0:
                    print(f"processed {matched_rows} matched rows...", file=os.sys.stderr)
    finally:
        for builder in builders.values():
            builder.close()

    for address, builder in builders.items():
        out_path = os.path.join(args.out, f"{address}.json")
        with open(out_path, "w", encoding="utf-8") as fh:
            json.dump(builder.metadata(), fh, indent=2)
            fh.write("\n")
        print(
            f"wrote {out_path} "
            f"(source_rows={builder.source_row_count}, counterparties={len(builder.unique_counterparties)})"
        )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
