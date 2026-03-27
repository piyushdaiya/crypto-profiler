#!/usr/bin/env python3
"""
Extract address-scoped ERC-20 Layer 1 summaries from local Blockchair ERC-20 TSV/TSV.gz files.

This script:
- scans a dated window of Blockchair ERC-20 transfer shards
- filters rows where sender or recipient matches tracked addresses
- writes per-address compressed NDJSON raw subsets
- writes per-address summary JSON artifacts

Example:
  python3 scripts/extract_erc20_layer1.py \
    --tx-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/blockchair/erc-20" \
    --tokens "/Volumes/Extreme SSD/thinkpad-backup/media-Data/blockchair/erc-20/blockchair_erc-20_tokens_latest.tsv.gz" \
    --addresses data/candidates/erc20_addresses.local.txt \
    --out data/cases/extracted-erc20 \
    --start 2025-03-16 \
    --end 2025-06-17
"""

from __future__ import annotations

import argparse
import collections
import datetime as dt
import gzip
import json
import os
import re
from dataclasses import dataclass, field
from decimal import Decimal, InvalidOperation
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Set


DATE_RE = re.compile(r"blockchair_erc-20_transactions_(\d{8})\.tsv(?:\.gz)?$")


def now_iso() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def normalize_evm_address(value: Optional[str]) -> str:
    if value is None:
        return ""
    s = str(value).strip().lower()
    if not s or s in {"\\n", "\\N"}:
        return ""
    s = s.removeprefix("0x")
    if len(s) == 40 and all(ch in "0123456789abcdef" for ch in s):
        return "0x" + s
    return ""


def decimal_or_zero(value: Optional[str]) -> Decimal:
    if value is None:
        return Decimal("0")
    s = str(value).strip()
    if not s:
        return Decimal("0")
    try:
        return Decimal(s)
    except (InvalidOperation, ValueError):
        return Decimal("0")


def int_or_zero(value: Optional[str]) -> int:
    return int(decimal_or_zero(value))


def load_addresses(path: str) -> List[str]:
    out: List[str] = []
    seen: Set[str] = set()

    with open(path, "r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            addr = normalize_evm_address(line)
            if not addr or addr in seen:
                continue
            seen.add(addr)
            out.append(addr)

    return out


def load_label_names(path: Optional[str]) -> Dict[str, str]:
    if not path:
        return {}

    with open(path, "r", encoding="utf-8") as fh:
        raw = json.load(fh)

    labels: Dict[str, str] = {}
    for addr, meta in raw.items():
        norm = normalize_evm_address(addr)
        if not norm:
            continue

        if isinstance(meta, str):
            label = meta.strip()
        elif isinstance(meta, dict):
            name = str(meta.get("name", "")).strip()
            category = str(meta.get("category", "")).strip()
            label = f"{category}: {name}".strip(": ").strip() if category or name else ""
        else:
            label = ""

        if label:
            labels[norm] = label

    return labels


def parse_date_arg(value: Optional[str]) -> Optional[int]:
    if not value:
        return None
    return int(value.replace("-", ""))


def extract_file_date(path: Path) -> Optional[int]:
    m = DATE_RE.match(path.name)
    if not m:
        return None
    return int(m.group(1))


def list_tx_files(directory: str, start_date: Optional[int], end_date: Optional[int]) -> List[Path]:
    base = Path(directory).expanduser().resolve()
    files = sorted(
        p for p in base.glob("blockchair_erc-20_transactions_*.tsv*")
        if p.is_file() and not p.name.startswith("._")
    )

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


def open_maybe_gzip(path: Path):
    if path.suffix == ".gz":
        return gzip.open(path, "rt", encoding="utf-8", newline="")
    return path.open("r", encoding="utf-8", newline="")


def iter_selected_tsv(path: Path, wanted_columns: List[str]) -> Iterable[Dict[str, str]]:
    with open_maybe_gzip(path) as fh:
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


def load_token_metadata(path: Optional[str], wanted_tokens: Set[str]) -> Dict[str, dict]:
    if not path or not wanted_tokens:
        return {}

    token_meta: Dict[str, dict] = {}
    for row in iter_selected_tsv(
        Path(path).expanduser().resolve(),
        [
            "address",
            "name",
            "symbol",
            "decimals",
        ],
    ):
        token_address = normalize_evm_address(row.get("address"))
        if token_address not in wanted_tokens:
            continue
        token_meta[token_address] = {
            "name": (row.get("name") or "").strip(),
            "symbol": (row.get("symbol") or "").strip(),
            "decimals": int_or_zero(row.get("decimals")),
        }

    return token_meta


def value_display(value_raw: str, decimals: int) -> str:
    raw = decimal_or_zero(value_raw)
    if decimals < 0:
        decimals = 0
    scale = Decimal(10) ** decimals
    if scale == 0:
        return str(raw)
    return str(raw / scale)


@dataclass
class TokenCount:
    token_address: str
    symbol: str
    count: int

    def to_json(self) -> dict:
        return {
            "token_address": self.token_address,
            "symbol": self.symbol,
            "count": self.count,
        }


@dataclass
class CounterpartySummary:
    address: str
    label: str = ""
    interactions: int = 0
    inbound_count: int = 0
    outbound_count: int = 0
    unique_tokens: Set[str] = field(default_factory=set)
    token_counts: collections.Counter = field(default_factory=collections.Counter)

    def to_json(self, token_symbols: Optional[Dict[str, str]] = None) -> dict:
        token_symbols = token_symbols or {}
        top_tokens = sorted(
            (
                TokenCount(
                    token_address=token,
                    symbol=token_symbols.get(token, ""),
                    count=count,
                )
                for token, count in self.token_counts.items()
            ),
            key=lambda item: (-item.count, item.token_address),
        )[:5]
        return {
            "address": self.address,
            "label": self.label,
            "interactions": self.interactions,
            "inbound_count": self.inbound_count,
            "outbound_count": self.outbound_count,
            "unique_tokens": len(self.unique_tokens),
            "top_tokens": [item.to_json() for item in top_tokens],
        }


@dataclass
class TokenSummary:
    token_address: str
    token_name: str = ""
    token_symbol: str = ""
    token_decimals: int = 0
    transfer_count: int = 0
    inbound_count: int = 0
    outbound_count: int = 0
    inbound_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))
    outbound_value_raw: Decimal = field(default_factory=lambda: Decimal("0"))
    counterparties: Set[str] = field(default_factory=set)

    def to_json(self) -> dict:
        return {
            "token_address": self.token_address,
            "token_name": self.token_name,
            "token_symbol": self.token_symbol,
            "token_decimals": self.token_decimals,
            "transfer_count": self.transfer_count,
            "inbound_count": self.inbound_count,
            "outbound_count": self.outbound_count,
            "inbound_value_raw": str(self.inbound_value_raw),
            "outbound_value_raw": str(self.outbound_value_raw),
            "unique_counterparties": len(self.counterparties),
        }


@dataclass
class SampleTransfer:
    block_timestamp: Optional[str]
    tx_hash: str
    sender: str
    recipient: str
    direction: str
    counterparty: str
    token_address: str
    token_name: str
    token_symbol: str
    token_decimals: int
    value_raw: str
    value_display: str
    label_sender: str = ""
    label_recipient: str = ""

    def to_json(self) -> dict:
        return {
            "block_timestamp": self.block_timestamp,
            "tx_hash": self.tx_hash,
            "sender": self.sender,
            "recipient": self.recipient,
            "direction": self.direction,
            "counterparty": self.counterparty,
            "token_address": self.token_address,
            "token_name": self.token_name,
            "token_symbol": self.token_symbol,
            "token_decimals": self.token_decimals,
            "value_raw": self.value_raw,
            "value_display": self.value_display,
            "label_sender": self.label_sender,
            "label_recipient": self.label_recipient,
        }


@dataclass
class ERC20DatasetBuilder:
    address: str
    out_dir: str
    label_names: Dict[str, str]
    sample_limit: int
    top_limit: int

    generated_at: str = field(default_factory=now_iso)
    first_seen: Optional[str] = None
    last_seen: Optional[str] = None

    inbound_transfer_count: int = 0
    outbound_transfer_count: int = 0
    self_transfer_count: int = 0

    inbound_counterparties: Set[str] = field(default_factory=set)
    outbound_counterparties: Set[str] = field(default_factory=set)
    counterparties: Dict[str, CounterpartySummary] = field(default_factory=dict)
    token_breakdown: Dict[str, TokenSummary] = field(default_factory=dict)
    seen_tokens: Set[str] = field(default_factory=set)

    source_row_count: int = 0
    _samples_head: List[SampleTransfer] = field(default_factory=list)
    _samples_tail: collections.deque = field(default_factory=collections.deque)

    def __post_init__(self) -> None:
        os.makedirs(self.out_dir, exist_ok=True)
        self.raw_filename = f"{self.address[2:]}.erc20.ndjson.gz"
        self.raw_path = os.path.join(self.out_dir, self.raw_filename)
        self._raw_fh = gzip.open(self.raw_path, "wt", encoding="utf-8")
        self._tail_max = max(1, self.sample_limit // 2)

    def close(self) -> None:
        self._raw_fh.close()

    def _touch_time(self, block_timestamp: Optional[str]) -> None:
        if not block_timestamp:
            return
        if self.first_seen is None or block_timestamp < self.first_seen:
            self.first_seen = block_timestamp
        if self.last_seen is None or block_timestamp > self.last_seen:
            self.last_seen = block_timestamp

    def _counterparty(self, address: str) -> CounterpartySummary:
        if address not in self.counterparties:
            self.counterparties[address] = CounterpartySummary(
                address=address,
                label=self.label_names.get(address, ""),
            )
        return self.counterparties[address]

    def _token(self, token_address: str, token_name: str, token_symbol: str, token_decimals: int) -> TokenSummary:
        if token_address not in self.token_breakdown:
            self.token_breakdown[token_address] = TokenSummary(
                token_address=token_address,
                token_name=token_name,
                token_symbol=token_symbol,
                token_decimals=token_decimals,
            )
        token = self.token_breakdown[token_address]
        if not token.token_name and token_name:
            token.token_name = token_name
        if not token.token_symbol and token_symbol:
            token.token_symbol = token_symbol
        if token.token_decimals == 0 and token_decimals:
            token.token_decimals = token_decimals
        return token

    def add(self, row: Dict[str, str]) -> None:
        sender = normalize_evm_address(row.get("sender"))
        recipient = normalize_evm_address(row.get("recipient"))
        if sender != self.address and recipient != self.address:
            return

        tx_hash = (row.get("transaction_hash") or "").strip()
        block_timestamp = (row.get("time") or "").strip() or None
        token_address = normalize_evm_address(row.get("token_address"))
        token_name = (row.get("token_name") or "").strip()
        token_symbol = (row.get("token_symbol") or "").strip()
        token_decimals = int_or_zero(row.get("token_decimals"))
        raw_value = (row.get("value") or "").strip() or "0"

        if not tx_hash or not token_address:
            return

        self.source_row_count += 1
        self.seen_tokens.add(token_address)
        self._touch_time(block_timestamp)

        if sender == self.address and recipient == self.address:
            direction = "self"
            counterparty = self.address
            self.self_transfer_count += 1
        elif recipient == self.address:
            direction = "inbound"
            counterparty = sender
            self.inbound_transfer_count += 1
            if counterparty:
                self.inbound_counterparties.add(counterparty)
        else:
            direction = "outbound"
            counterparty = recipient
            self.outbound_transfer_count += 1
            if counterparty:
                self.outbound_counterparties.add(counterparty)

        if counterparty and counterparty != self.address:
            cp = self._counterparty(counterparty)
            cp.interactions += 1
            if direction == "inbound":
                cp.inbound_count += 1
            elif direction == "outbound":
                cp.outbound_count += 1
            cp.unique_tokens.add(token_address)
            cp.token_counts[token_address] += 1

        token = self._token(token_address, token_name, token_symbol, token_decimals)
        token.transfer_count += 1
        if direction == "inbound":
            token.inbound_count += 1
            token.inbound_value_raw += decimal_or_zero(raw_value)
        elif direction == "outbound":
            token.outbound_count += 1
            token.outbound_value_raw += decimal_or_zero(raw_value)
        if counterparty and counterparty != self.address:
            token.counterparties.add(counterparty)

        sample = SampleTransfer(
            block_timestamp=block_timestamp,
            tx_hash=tx_hash,
            sender=sender,
            recipient=recipient,
            direction=direction,
            counterparty=counterparty,
            token_address=token_address,
            token_name=token_name,
            token_symbol=token_symbol,
            token_decimals=token_decimals,
            value_raw=raw_value,
            value_display=value_display(raw_value, token_decimals),
            label_sender=self.label_names.get(sender, ""),
            label_recipient=self.label_names.get(recipient, ""),
        )

        raw_line = sample.to_json()
        self._raw_fh.write(json.dumps(raw_line, separators=(",", ":")) + "\n")

        if len(self._samples_head) < self._tail_max:
            self._samples_head.append(sample)
        else:
            self._samples_tail.append(sample)
            if len(self._samples_tail) > self._tail_max:
                self._samples_tail.popleft()

    def finalize_metadata(self, token_meta: Dict[str, dict]) -> None:
        for token_address, token in self.token_breakdown.items():
            meta = token_meta.get(token_address)
            if not meta:
                continue
            if not token.token_name:
                token.token_name = meta.get("name", "")
            if not token.token_symbol:
                token.token_symbol = meta.get("symbol", "")
            if token.token_decimals == 0 and meta.get("decimals") is not None:
                token.token_decimals = int(meta["decimals"])

    def write_summary(self) -> str:
        total_transfers = self.inbound_transfer_count + self.outbound_transfer_count
        if self.outbound_transfer_count > self.inbound_transfer_count:
            dominant_direction = "outbound"
        elif self.inbound_transfer_count > self.outbound_transfer_count:
            dominant_direction = "inbound"
        else:
            dominant_direction = "balanced"

        token_rows = sorted(
            self.token_breakdown.values(),
            key=lambda item: (-item.transfer_count, item.token_address),
        )
        dominant_token = token_rows[0] if token_rows else None
        dominant_share = 0.0
        if dominant_token and total_transfers > 0:
            dominant_share = round((dominant_token.transfer_count / total_transfers) * 100.0, 2)

        top_counterparties = sorted(
            self.counterparties.values(),
            key=lambda item: (-item.interactions, item.address),
        )[: self.top_limit]
        token_symbols = {
            token.token_address: token.token_symbol
            for token in self.token_breakdown.values()
            if token.token_symbol
        }

        repeated_counterparties = sum(1 for cp in self.counterparties.values() if cp.interactions >= 3)
        max_counterparty_interactions = max((cp.interactions for cp in self.counterparties.values()), default=0)

        samples = self._samples_head + list(self._samples_tail)
        deduped: Dict[tuple, SampleTransfer] = {}
        for sample in samples:
            key = (sample.tx_hash, sample.direction, sample.token_address, sample.counterparty)
            deduped[key] = sample
        sample_rows = sorted(
            deduped.values(),
            key=lambda item: ((item.block_timestamp or ""), item.tx_hash, item.direction),
        )[: self.sample_limit]

        summary_path = os.path.join(self.out_dir, f"{self.address[2:]}.json")
        with open(summary_path, "w", encoding="utf-8") as fh:
            json.dump(
                {
                    "address": self.address,
                    "chain": "EVM",
                    "dataset_type": "erc20_layer1",
                    "generated_at": self.generated_at,
                    "source_row_count": self.source_row_count,
                    "raw_subset_file": self.raw_filename,
                    "summary": {
                        "first_seen": self.first_seen,
                        "last_seen": self.last_seen,
                        "inbound_transfer_count": self.inbound_transfer_count,
                        "outbound_transfer_count": self.outbound_transfer_count,
                        "self_transfer_count": self.self_transfer_count,
                        "inbound_counterparties": len(self.inbound_counterparties),
                        "outbound_counterparties": len(self.outbound_counterparties),
                        "unique_counterparties": len(self.counterparties),
                        "unique_token_contracts": len(self.token_breakdown),
                        "repeated_counterparties": repeated_counterparties,
                        "max_counterparty_interactions": max_counterparty_interactions,
                        "dominant_direction": dominant_direction,
                        "dominant_token_address": dominant_token.token_address if dominant_token else "",
                        "dominant_token_symbol": dominant_token.token_symbol if dominant_token else "",
                        "dominant_token_transfer_share_pct": dominant_share,
                    },
                    "token_breakdown": [token.to_json() for token in token_rows[: self.top_limit]],
                    "top_counterparties": [cp.to_json(token_symbols) for cp in top_counterparties],
                    "sample_transfers": [sample.to_json() for sample in sample_rows],
                },
                fh,
                indent=2,
            )
            fh.write("\n")

        return summary_path


def main() -> int:
    parser = argparse.ArgumentParser(description="Extract address-scoped ERC-20 Layer 1 summaries")
    parser.add_argument("--tx-dir", required=True, help="Directory containing Blockchair ERC-20 transaction shards")
    parser.add_argument("--tokens", default="", help="Optional path to blockchair_erc-20_tokens_latest TSV/TSV.GZ")
    parser.add_argument("--addresses", required=True, help="Tracked address list")
    parser.add_argument("--out", required=True, help="Output directory for extracted ERC-20 artifacts")
    parser.add_argument("--labels", default="data/labels/bootstrap_entities.json", help="Optional label JSON for known names")
    parser.add_argument("--start", default="2025-03-16", help="Inclusive start date (YYYY-MM-DD)")
    parser.add_argument("--end", default="2025-06-17", help="Inclusive end date (YYYY-MM-DD)")
    parser.add_argument("--sample", type=int, default=50, help="Sample transfer count to retain")
    parser.add_argument("--top", type=int, default=20, help="Top counterparties/token rows to retain")
    args = parser.parse_args()

    addresses = load_addresses(args.addresses)
    if not addresses:
        raise SystemExit("no addresses loaded")

    start_date = parse_date_arg(args.start)
    end_date = parse_date_arg(args.end)
    tx_files = list_tx_files(args.tx_dir, start_date, end_date)
    if not tx_files:
        raise SystemExit("no ERC-20 transaction files matched the requested window")

    label_names = load_label_names(args.labels)
    builders = {
        address: ERC20DatasetBuilder(
            address=address,
            out_dir=str(Path(args.out).expanduser().resolve()),
            label_names=label_names,
            sample_limit=args.sample,
            top_limit=args.top,
        )
        for address in addresses
    }

    wanted_columns = [
        "transaction_hash",
        "time",
        "token_address",
        "token_name",
        "token_symbol",
        "token_decimals",
        "sender",
        "recipient",
        "value",
    ]

    try:
        for path in tx_files:
            print(f"scanning {path.name} ...")
            for row in iter_selected_tsv(path, wanted_columns):
                sender = normalize_evm_address(row.get("sender"))
                recipient = normalize_evm_address(row.get("recipient"))
                if sender in builders:
                    builders[sender].add(row)
                if recipient in builders and recipient != sender:
                    builders[recipient].add(row)
    finally:
        for builder in builders.values():
            builder.close()

    wanted_tokens: Set[str] = set()
    for builder in builders.values():
        wanted_tokens.update(builder.seen_tokens)

    token_meta = load_token_metadata(args.tokens, wanted_tokens)

    for address, builder in builders.items():
        builder.finalize_metadata(token_meta)
        summary_path = builder.write_summary()
        print(
            f"wrote {summary_path} "
            f"(rows={builder.source_row_count}, counterparties={len(builder.counterparties)}, tokens={len(builder.token_breakdown)})"
        )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
