#!/usr/bin/env python3
"""
Mine behavior-driven ERC-20 candidate addresses from local Blockchair ERC-20 shards.

This script performs:
- a broad first pass to find highly active addresses
- a focused second pass to score broad-surface, repeated-counterparty, and token-heavy behavior

Example:
  python3 scripts/mine_erc20_candidates.py \
    --tx-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/blockchair/erc-20" \
    --start 2025-03-16 \
    --end 2025-06-17 \
    --labels data/labels/bootstrap_entities.json \
    --top 30
"""

from __future__ import annotations

import argparse
import collections
import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Set


DATE_RE = re.compile(r"blockchair_erc-20_transactions_(\d{8})\.tsv(?:\.gz)?$")
ZERO_ADDRESS = "0x0000000000000000000000000000000000000000"


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


def is_candidate_address(addr: str) -> bool:
    return bool(addr) and addr != ZERO_ADDRESS


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
    import gzip

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
        max_idx = max(index_map[col] for col in wanted_columns)

        for line in fh:
            line = line.rstrip("\n").rstrip("\r")
            parts = line.split("\t", max_idx + 1)
            row = {}
            for col in wanted_columns:
                idx = index_map[col]
                row[col] = parts[idx] if idx < len(parts) else ""
            yield row


def load_label_names(path: Optional[str]) -> Dict[str, str]:
    if not path:
        return {}
    raw = json.loads(Path(path).read_text())
    out: Dict[str, str] = {}
    for addr, meta in raw.items():
        norm = normalize_evm_address(addr)
        if not norm:
            continue
        if isinstance(meta, dict):
            name = str(meta.get("name", "")).strip()
            category = str(meta.get("category", "")).strip()
            out[norm] = f"{category}: {name}".strip(": ").strip() if category or name else ""
        elif isinstance(meta, str):
            out[norm] = meta.strip()
    return out


@dataclass
class CandidateDetail:
    address: str
    inbound_count: int = 0
    outbound_count: int = 0
    counterparties: Set[str] = field(default_factory=set)
    tokens: Set[str] = field(default_factory=set)
    counterparty_counts: collections.Counter = field(default_factory=collections.Counter)

    @property
    def total_count(self) -> int:
        return self.inbound_count + self.outbound_count

    @property
    def dominant_direction(self) -> str:
        if self.outbound_count > self.inbound_count:
            return "outbound"
        if self.inbound_count > self.outbound_count:
            return "inbound"
        return "balanced"

    @property
    def unique_counterparties(self) -> int:
        return len(self.counterparties)

    @property
    def unique_tokens(self) -> int:
        return len(self.tokens)

    @property
    def max_counterparty_interactions(self) -> int:
        return max(self.counterparty_counts.values(), default=0)


def main() -> int:
    parser = argparse.ArgumentParser(description="Mine ERC-20 candidate addresses")
    parser.add_argument("--tx-dir", required=True, help="Directory containing Blockchair ERC-20 transaction shards")
    parser.add_argument("--start", default="2025-03-16", help="Inclusive start date (YYYY-MM-DD)")
    parser.add_argument("--end", default="2025-06-17", help="Inclusive end date (YYYY-MM-DD)")
    parser.add_argument("--top", type=int, default=30, help="Rows to print per category")
    parser.add_argument("--candidate-scan", type=int, default=5000, help="High-activity addresses to examine in detail")
    parser.add_argument("--min-total-count", type=int, default=500, help="Minimum total transfer count for detailed scoring")
    parser.add_argument("--labels", default="data/labels/bootstrap_entities.json", help="Optional label JSON for annotation")
    parser.add_argument("--out", default="", help="Optional output file for mined addresses")
    args = parser.parse_args()

    start_date = parse_date_arg(args.start)
    end_date = parse_date_arg(args.end)
    tx_files = list_tx_files(args.tx_dir, start_date, end_date)
    if not tx_files:
        raise SystemExit("no ERC-20 transaction files matched the requested window")

    labels = load_label_names(args.labels)
    inbound_counts: collections.Counter = collections.Counter()
    outbound_counts: collections.Counter = collections.Counter()

    for path in tx_files:
        print(f"pass1 {path.name} ...", flush=True)
        for row in iter_selected_tsv(path, ["sender", "recipient"]):
            sender = normalize_evm_address(row.get("sender"))
            recipient = normalize_evm_address(row.get("recipient"))

            if is_candidate_address(sender) and sender != recipient:
                outbound_counts[sender] += 1
            if is_candidate_address(recipient):
                inbound_counts[recipient] += 1

    selected: Set[str] = set()
    all_addresses = set(inbound_counts) | set(outbound_counts)
    ranked_totals = sorted(
        (
            (
                address,
                inbound_counts.get(address, 0),
                outbound_counts.get(address, 0),
            )
            for address in all_addresses
        ),
        key=lambda item: (-(item[1] + item[2]), item[0]),
    )

    for address, inbound_count, outbound_count in ranked_totals:
        total_count = inbound_count + outbound_count
        if not is_candidate_address(address):
            continue
        if total_count < args.min_total_count:
            break
        selected.add(address)
        if len(selected) >= args.candidate_scan:
            break

    for addr in labels:
        if is_candidate_address(addr) and inbound_counts.get(addr, 0)+outbound_counts.get(addr, 0) > 0:
            selected.add(addr)

    details = {addr: CandidateDetail(address=addr) for addr in selected}

    for path in tx_files:
        print(f"pass2 {path.name} ...", flush=True)
        for row in iter_selected_tsv(path, ["sender", "recipient", "token_address"]):
            sender = normalize_evm_address(row.get("sender"))
            recipient = normalize_evm_address(row.get("recipient"))
            token_address = normalize_evm_address(row.get("token_address"))

            if sender in details and recipient and recipient != sender:
                item = details[sender]
                item.outbound_count += 1
                item.counterparties.add(recipient)
                if token_address:
                    item.tokens.add(token_address)
                item.counterparty_counts[recipient] += 1

            if recipient in details and sender and sender != recipient:
                item = details[recipient]
                item.inbound_count += 1
                item.counterparties.add(sender)
                if token_address:
                    item.tokens.add(token_address)
                item.counterparty_counts[sender] += 1

    ranked = sorted(details.values(), key=lambda item: (-item.total_count, item.address))
    broad_surface = sorted(details.values(), key=lambda item: (-item.unique_counterparties, -item.total_count, item.address))
    token_heavy = sorted(details.values(), key=lambda item: (-item.unique_tokens, -item.total_count, item.address))
    repeated = sorted(details.values(), key=lambda item: (-item.max_counterparty_interactions, -item.total_count, item.address))
    inbound_heavy = sorted(
        (item for item in details.values() if item.inbound_count > item.outbound_count),
        key=lambda item: (-(item.inbound_count - item.outbound_count), -item.unique_counterparties, item.address),
    )

    def print_rows(title: str, rows_to_print: List[CandidateDetail]) -> None:
        print(f"\n{title}\n", flush=True)
        for item in rows_to_print[: args.top]:
            label = labels.get(item.address, "")
            label_part = f" | label={label}" if label else ""
            print(
                f"{item.address} | total={item.total_count} | in={item.inbound_count} | out={item.outbound_count} "
                f"| counterparties={item.unique_counterparties} | tokens={item.unique_tokens} "
                f"| max_cp_interactions={item.max_counterparty_interactions} | dominant={item.dominant_direction}{label_part}",
                flush=True,
            )

    print_rows("Top overall candidates:", ranked)
    print_rows("Top broad-surface candidates:", broad_surface)
    print_rows("Top token-heavy candidates:", token_heavy)
    print_rows("Top repeated-counterparty candidates:", repeated)
    print_rows("Top inbound-heavy candidates:", inbound_heavy)

    if args.out:
        out_path = Path(args.out).expanduser().resolve()
        out_path.parent.mkdir(parents=True, exist_ok=True)

        written: List[str] = []
        seen: Set[str] = set()
        sections = [
            ("overall", ranked),
            ("broad_surface", broad_surface),
            ("token_heavy", token_heavy),
            ("repeated_counterparty", repeated),
            ("inbound_heavy", inbound_heavy),
        ]

        with out_path.open("w", encoding="utf-8") as fh:
            fh.write("# ERC-20 candidate addresses mined from local Blockchair data\n")
            fh.write(f"# Window: {args.start} to {args.end}\n")
            for section_name, section_rows in sections:
                fh.write(f"\n# {section_name}\n")
                count = 0
                for item in section_rows:
                    if item.address in seen:
                        continue
                    seen.add(item.address)
                    label = labels.get(item.address, "")
                    label_part = f" | {label}" if label else ""
                    fh.write(
                        f"# total={item.total_count} counterparties={item.unique_counterparties} "
                        f"tokens={item.unique_tokens} max_cp={item.max_counterparty_interactions}{label_part}\n"
                    )
                    fh.write(item.address + "\n")
                    written.append(item.address)
                    count += 1
                    if count >= min(10, args.top):
                        break

        print(f"\nwrote {out_path} ({len(written)} addresses)", flush=True)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
