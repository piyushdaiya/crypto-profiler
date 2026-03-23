#!/usr/bin/env python3
"""
Extract address-scoped Bitcoin Layer 1 summaries from local Blockchair inputs/outputs TSV.gz files.

This script uses:
- outputs rows where recipient == tracked address  -> inbound receipts
- inputs rows where recipient == tracked address and spending_transaction_hash exists -> outbound spends
- transaction-level context:
  - inbound counterparties are approximated from input recipients of the same tx
  - outbound counterparties are approximated from output recipients of the spending tx

Recommended first window with your current local files:
  --start 2025-03-16 --end 2025-06-17

Example:
  python3 scripts/extract_bitcoin_layer1.py \
    --inputs-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/blockchair/bitcoin/input" \
    --outputs-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/blockchair/bitcoin/output" \
    --addresses data/candidates/bitcoin_addresses.local.txt \
    --out data/cases/extracted-bitcoin \
    --start 2025-03-16 \
    --end 2025-06-17
"""

from __future__ import annotations

import argparse
import datetime as dt
import gzip
import json
import os
import re
from dataclasses import dataclass, field
from decimal import Decimal, InvalidOperation
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Set


DATE_RE = re.compile(r"(\d{8})\.tsv\.gz$")


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


def parse_date_arg(value: Optional[str]) -> Optional[int]:
    if not value:
        return None
    return int(value.replace("-", ""))


def extract_file_date(path: Path) -> Optional[int]:
    m = DATE_RE.search(path.name)
    if not m:
        return None
    return int(m.group(1))


def list_tsv_gz_files(directory: str, start_date: Optional[int], end_date: Optional[int]) -> List[Path]:
    base = Path(directory).expanduser().resolve()
    files = sorted(p for p in base.glob("*.tsv.gz") if p.is_file() and not p.name.startswith("._"))

    filtered: List[Path] = []
    for path in files:
        file_date = extract_file_date(path)
        if file_date is None:
            continue
        if start_date is not None and file_date < start_date:
            continue
        if end_date is not None and file_date > end_date:
            continue
        filtered.append(path)

    return filtered


def decimal_or_zero(value: Optional[str]) -> Decimal:
    if value is None:
        return Decimal("0")
    s = str(value).strip()
    if s == "":
        return Decimal("0")
    try:
        return Decimal(s)
    except (InvalidOperation, ValueError):
        return Decimal("0")


def int_or_zero(value: Optional[str]) -> int:
    if value is None:
        return 0
    s = str(value).strip()
    if s == "":
        return 0
    try:
        return int(s)
    except ValueError:
        return int(decimal_or_zero(s))


def iso_or_none(value: Optional[str]) -> Optional[str]:
    if value is None:
        return None
    s = str(value).strip()
    return s or None


def sats_to_btc_str(sats: int) -> str:
    return str(Decimal(sats) / Decimal(100_000_000))


def looks_like_btc_address(value: Optional[str]) -> bool:
    if value is None:
        return False
    s = str(value).strip()
    if not s:
        return False
    if s.startswith("d-"):
        return False
    return s.startswith("1") or s.startswith("3") or s.startswith("bc1")


def open_tsv_gz(path: Path):
    return gzip.open(path, "rt", encoding="utf-8", newline="")


def iter_selected_tsv_gz(path: Path, wanted_columns: List[str]) -> Iterable[Dict[str, str]]:
    """
    Fast TSV reader that only splits up to the highest needed column index.
    This avoids fully parsing giant trailing fields like witness/signature blobs.
    """
    with open_tsv_gz(path) as fh:
        header_line = fh.readline()
        if not header_line:
            return

        header = header_line.rstrip("\n").rstrip("\r").split("\t")
        index_map = {name: idx for idx, name in enumerate(header)}

        missing = [col for col in wanted_columns if col not in index_map]
        if missing:
            raise ValueError(f"{path} missing required columns: {missing}")

        max_idx = max(index_map[col] for col in wanted_columns)

        for line in fh:
            line = line.rstrip("\n").rstrip("\r")
            parts = line.split("\t", max_idx + 1)

            row: Dict[str, str] = {}
            for col in wanted_columns:
                idx = index_map[col]
                row[col] = parts[idx] if idx < len(parts) else ""
            yield row


@dataclass
class CounterpartySummary:
    address: str
    interactions: int = 0
    inbound_count: int = 0
    outbound_count: int = 0

    def to_json(self) -> dict:
        return {
            "address": self.address,
            "interactions": self.interactions,
            "inbound_count": self.inbound_count,
            "outbound_count": self.outbound_count,
        }


@dataclass
class BitcoinEvent:
    direction: str
    event_time: Optional[str]
    tx_hash: str
    linked_tx_hash: Optional[str]
    value_sats: int
    value_btc: str
    value_usd: str
    address: str
    script_type: str
    counterparties: List[str] = field(default_factory=list)

    def to_json(self) -> dict:
        return {
            "direction": self.direction,
            "event_time": self.event_time,
            "tx_hash": self.tx_hash,
            "linked_tx_hash": self.linked_tx_hash,
            "value_sats": self.value_sats,
            "value_btc": self.value_btc,
            "value_usd": self.value_usd,
            "address": self.address,
            "script_type": self.script_type,
            "counterparties": self.counterparties,
        }


@dataclass
class BitcoinDatasetBuilder:
    address: str
    out_dir: str
    sample_limit: int
    top_limit: int

    generated_at: str = field(default_factory=now_iso)

    first_seen: Optional[str] = None
    last_seen: Optional[str] = None

    inbound_receipt_count: int = 0
    outbound_spend_count: int = 0
    inbound_value_sats: int = 0
    outbound_value_sats: int = 0

    unique_counterparties: Set[str] = field(default_factory=set)
    counterparties: Dict[str, CounterpartySummary] = field(default_factory=dict)

    mintless_sample_events: List[dict] = field(default_factory=list)
    events: List[BitcoinEvent] = field(default_factory=list)

    inbound_tx_to_event_idxs: Dict[str, List[int]] = field(default_factory=dict)
    outbound_spend_tx_to_event_idxs: Dict[str, List[int]] = field(default_factory=dict)

    source_row_count: int = 0

    def __post_init__(self) -> None:
        os.makedirs(self.out_dir, exist_ok=True)
        self.raw_filename = f"{self.address}.bitcoin.ndjson.gz"
        self.raw_path = os.path.join(self.out_dir, self.raw_filename)

    def _touch_time(self, event_time: Optional[str]) -> None:
        if not event_time:
            return
        if self.first_seen is None or event_time < self.first_seen:
            self.first_seen = event_time
        if self.last_seen is None or event_time > self.last_seen:
            self.last_seen = event_time

    def add_inbound(self, row: dict) -> None:
        tx_hash = row["transaction_hash"]
        event_time = iso_or_none(row.get("time"))
        value_sats = int_or_zero(row.get("value"))
        value_usd = str(decimal_or_zero(row.get("value_usd")))

        event = BitcoinEvent(
            direction="inbound",
            event_time=event_time,
            tx_hash=tx_hash,
            linked_tx_hash=None,
            value_sats=value_sats,
            value_btc=sats_to_btc_str(value_sats),
            value_usd=value_usd,
            address=self.address,
            script_type=str(row.get("type") or ""),
        )

        idx = len(self.events)
        self.events.append(event)
        self.inbound_tx_to_event_idxs.setdefault(tx_hash, []).append(idx)

        self.source_row_count += 1
        self.inbound_receipt_count += 1
        self.inbound_value_sats += value_sats
        self._touch_time(event_time)

    def add_outbound(self, row: dict) -> None:
        spend_tx_hash = str(row.get("spending_transaction_hash") or "").strip()
        if not spend_tx_hash:
            return

        event_time = iso_or_none(row.get("spending_time"))
        value_sats = int_or_zero(row.get("value"))
        value_usd = str(decimal_or_zero(row.get("spending_value_usd")))

        event = BitcoinEvent(
            direction="outbound",
            event_time=event_time,
            tx_hash=spend_tx_hash,
            linked_tx_hash=str(row.get("transaction_hash") or "").strip() or None,
            value_sats=value_sats,
            value_btc=sats_to_btc_str(value_sats),
            value_usd=value_usd,
            address=self.address,
            script_type=str(row.get("type") or ""),
        )

        idx = len(self.events)
        self.events.append(event)
        self.outbound_spend_tx_to_event_idxs.setdefault(spend_tx_hash, []).append(idx)

        self.source_row_count += 1
        self.outbound_spend_count += 1
        self.outbound_value_sats += value_sats
        self._touch_time(event_time)

    def _record_counterparty(self, cp_address: str, direction: str) -> None:
        if not looks_like_btc_address(cp_address):
            return
        if cp_address == self.address:
            return

        self.unique_counterparties.add(cp_address)
        cp = self.counterparties.get(cp_address)
        if cp is None:
            cp = CounterpartySummary(address=cp_address)
            self.counterparties[cp_address] = cp

        cp.interactions += 1
        if direction == "inbound":
            cp.inbound_count += 1
        elif direction == "outbound":
            cp.outbound_count += 1

    def attach_inbound_sources(self, tx_hash: str, sources: Set[str]) -> None:
        for idx in self.inbound_tx_to_event_idxs.get(tx_hash, []):
            counterparties = sorted(cp for cp in sources if looks_like_btc_address(cp) and cp != self.address)
            self.events[idx].counterparties = counterparties
            for cp in counterparties:
                self._record_counterparty(cp, "inbound")

    def attach_outbound_destinations(self, tx_hash: str, destinations: Set[str]) -> None:
        for idx in self.outbound_spend_tx_to_event_idxs.get(tx_hash, []):
            counterparties = sorted(cp for cp in destinations if looks_like_btc_address(cp) and cp != self.address)
            self.events[idx].counterparties = counterparties
            for cp in counterparties:
                self._record_counterparty(cp, "outbound")

    def write_outputs(self) -> None:
        with gzip.open(self.raw_path, "wt", encoding="utf-8") as fh:
            for event in self.events:
                row = event.to_json()
                if len(self.mintless_sample_events) < self.sample_limit:
                    self.mintless_sample_events.append(row)
                fh.write(json.dumps(row, separators=(",", ":")) + "\n")

    def metadata(self) -> dict:
        top_counterparties = sorted(
            self.counterparties.values(),
            key=lambda x: (x.interactions, x.inbound_count + x.outbound_count),
            reverse=True,
        )[: self.top_limit]

        return {
            "address": self.address,
            "chain": "BITCOIN",
            "dataset_type": "utxo_flow_layer1",
            "generated_at": self.generated_at,
            "summary": {
                "first_seen": self.first_seen,
                "last_seen": self.last_seen,
                "inbound_receipt_count": self.inbound_receipt_count,
                "outbound_spend_count": self.outbound_spend_count,
                "inbound_value_sats": self.inbound_value_sats,
                "outbound_value_sats": self.outbound_value_sats,
                "inbound_value_btc": sats_to_btc_str(self.inbound_value_sats),
                "outbound_value_btc": sats_to_btc_str(self.outbound_value_sats),
                "unique_counterparties": len(self.unique_counterparties),
                "counterparty_resolution_mode": "inbound=tx_input_recipients outbound=tx_output_recipients",
            },
            "top_counterparties": [cp.to_json() for cp in top_counterparties],
            "sample_events": self.mintless_sample_events,
            "source_row_count": self.source_row_count,
            "raw_flow_file": self.raw_filename,
        }


def pass1_collect_tracked_events(
    output_files: List[Path],
    input_files: List[Path],
    builders: Dict[str, BitcoinDatasetBuilder],
) -> tuple[Set[str], Set[str]]:
    inbound_tx_hashes: Set[str] = set()
    outbound_spend_tx_hashes: Set[str] = set()
    tracked = set(builders.keys())

    for path in output_files:
        for row in iter_selected_tsv_gz(
            path,
            ["transaction_hash", "time", "value", "value_usd", "recipient", "type"],
        ):
            recipient = str(row.get("recipient") or "").strip()
            if recipient in tracked:
                builders[recipient].add_inbound(row)
                inbound_tx_hashes.add(str(row.get("transaction_hash") or "").strip())

    for path in input_files:
        for row in iter_selected_tsv_gz(
            path,
            [
                "transaction_hash",
                "value",
                "recipient",
                "type",
                "spending_transaction_hash",
                "spending_time",
                "spending_value_usd",
            ],
        ):
            recipient = str(row.get("recipient") or "").strip()
            if recipient in tracked:
                spend_tx = str(row.get("spending_transaction_hash") or "").strip()
                if spend_tx:
                    builders[recipient].add_outbound(row)
                    outbound_spend_tx_hashes.add(spend_tx)

    return inbound_tx_hashes, outbound_spend_tx_hashes


def pass2_build_inbound_sources(input_files: List[Path], relevant_spend_txs: Set[str]) -> Dict[str, Set[str]]:
    inbound_sources: Dict[str, Set[str]] = {}

    if not relevant_spend_txs:
        return inbound_sources

    for path in input_files:
        for row in iter_selected_tsv_gz(path, ["recipient", "spending_transaction_hash"]):
            spend_tx = str(row.get("spending_transaction_hash") or "").strip()
            if spend_tx in relevant_spend_txs:
                recipient = str(row.get("recipient") or "").strip()
                if looks_like_btc_address(recipient):
                    inbound_sources.setdefault(spend_tx, set()).add(recipient)

    return inbound_sources


def pass3_build_outbound_destinations(output_files: List[Path], relevant_txs: Set[str]) -> Dict[str, Set[str]]:
    outbound_destinations: Dict[str, Set[str]] = {}

    if not relevant_txs:
        return outbound_destinations

    for path in output_files:
        for row in iter_selected_tsv_gz(path, ["transaction_hash", "recipient"]):
            tx_hash = str(row.get("transaction_hash") or "").strip()
            if tx_hash in relevant_txs:
                recipient = str(row.get("recipient") or "").strip()
                if looks_like_btc_address(recipient):
                    outbound_destinations.setdefault(tx_hash, set()).add(recipient)

    return outbound_destinations


def main() -> int:
    parser = argparse.ArgumentParser(description="Extract Bitcoin Layer 1 address-scoped summaries")
    parser.add_argument("--inputs-dir", required=True, help="Blockchair bitcoin input directory")
    parser.add_argument("--outputs-dir", required=True, help="Blockchair bitcoin output directory")
    parser.add_argument("--addresses", required=True, help="Text file with one Bitcoin address per line")
    parser.add_argument("--out", required=True, help="Output directory")
    parser.add_argument("--sample", type=int, default=200, help="Sample events to keep per address")
    parser.add_argument("--top", type=int, default=20, help="Top counterparties to keep")
    parser.add_argument("--start", help="Start date YYYY-MM-DD")
    parser.add_argument("--end", help="End date YYYY-MM-DD")
    args = parser.parse_args()

    addresses = load_addresses(args.addresses)
    if not addresses:
        print("error: no addresses loaded")
        return 1

    start_date = parse_date_arg(args.start)
    end_date = parse_date_arg(args.end)

    output_files = list_tsv_gz_files(args.outputs_dir, start_date, end_date)
    input_files = list_tsv_gz_files(args.inputs_dir, start_date, end_date)

    if not output_files:
        print("error: no output files matched the requested window")
        return 1
    if not input_files:
        print("error: no input files matched the requested window")
        return 1

    builders = {
        address: BitcoinDatasetBuilder(address, args.out, args.sample, args.top)
        for address in addresses
    }

    print(f"Using {len(output_files)} output shards and {len(input_files)} input shards")

    inbound_tx_hashes, outbound_spend_tx_hashes = pass1_collect_tracked_events(output_files, input_files, builders)
    print(f"tracked inbound tx hashes   : {len(inbound_tx_hashes)}")
    print(f"tracked outbound spend txs : {len(outbound_spend_tx_hashes)}")

    inbound_sources = pass2_build_inbound_sources(input_files, inbound_tx_hashes)
    outbound_destinations = pass3_build_outbound_destinations(output_files, outbound_spend_tx_hashes)

    for builder in builders.values():
        for tx_hash in builder.inbound_tx_to_event_idxs.keys():
            builder.attach_inbound_sources(tx_hash, inbound_sources.get(tx_hash, set()))
        for tx_hash in builder.outbound_spend_tx_to_event_idxs.keys():
            builder.attach_outbound_destinations(tx_hash, outbound_destinations.get(tx_hash, set()))

        builder.write_outputs()

        out_path = os.path.join(args.out, f"{builder.address}.json")
        with open(out_path, "w", encoding="utf-8") as fh:
            json.dump(builder.metadata(), fh, indent=2)
            fh.write("\n")

        print(
            f"wrote {out_path} "
            f"(events={len(builder.events)}, counterparties={len(builder.unique_counterparties)})"
        )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())