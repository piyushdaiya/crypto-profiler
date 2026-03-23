#!/usr/bin/env python3
"""
Mine Bitcoin candidate addresses from local Blockchair input/output TSV.gz files.

This version is optimized to avoid parsing giant trailing fields by only
splitting up to the highest needed column index.

Example:
  python3 scripts/mine_bitcoin_candidates.py \
    --inputs-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/blockchair/bitcoin/input" \
    --outputs-dir "/Volumes/Extreme SSD/thinkpad-backup/media-Data/blockchair/bitcoin/output" \
    --start 2025-03-16 \
    --end 2025-06-17 \
    --top 30
"""

from __future__ import annotations

import argparse
import gzip
import re
from dataclasses import dataclass
from decimal import Decimal, InvalidOperation
from pathlib import Path
from typing import Dict, Iterable, List, Optional


DATE_RE = re.compile(r"(\d{8})\.tsv\.gz$")

def looks_like_btc_address(value: Optional[str]) -> bool:
    if value is None:
        return False
    s = str(value).strip()
    if not s:
        return False
    if s.startswith("d-"):
        return False
    return (
        s.startswith("1")
        or s.startswith("3")
        or s.startswith("bc1")
    )
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

    out: List[Path] = []
    for path in files:
        d = extract_file_date(path)
        if d is None:
            continue
        if start_date is not None and d < start_date:
            continue
        if end_date is not None and d > end_date:
            continue
        out.append(path)
    return out


def open_tsv_gz(path: Path):
    return gzip.open(path, "rt", encoding="utf-8", newline="")


def is_meaningful_address(value: Optional[str]) -> bool:
    if value is None:
        return False
    s = str(value).strip()
    if not s:
        return False
    if s.startswith("d-"):
        return False
    return True


def int_or_zero(value: Optional[str]) -> int:
    if value is None:
        return 0
    s = str(value).strip()
    if not s:
        return 0
    try:
        return int(s)
    except ValueError:
        try:
            return int(Decimal(s))
        except (InvalidOperation, ValueError):
            return 0


def sats_to_btc_str(sats: int) -> str:
    return str(Decimal(sats) / Decimal(100_000_000))


def iter_selected_tsv_gz(path: Path, wanted_columns: List[str]) -> Iterable[Dict[str, str]]:
    """
    Fast TSV reader that only splits up to the max needed column index.
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
class AddrStats:
    address: str
    receive_count: int = 0
    spend_count: int = 0
    receive_value_sats: int = 0
    spend_value_sats: int = 0

    @property
    def total_count(self) -> int:
        return self.receive_count + self.spend_count

    @property
    def balance_delta(self) -> int:
        return abs(self.receive_count - self.spend_count)


def get_stats(stats: Dict[str, AddrStats], address: str) -> AddrStats:
    if address not in stats:
        stats[address] = AddrStats(address=address)
    return stats[address]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--inputs-dir", required=True)
    parser.add_argument("--outputs-dir", required=True)
    parser.add_argument("--start")
    parser.add_argument("--end")
    parser.add_argument("--top", type=int, default=50)
    args = parser.parse_args()

    start_date = parse_date_arg(args.start)
    end_date = parse_date_arg(args.end)

    input_files = list_tsv_gz_files(args.inputs_dir, start_date, end_date)
    output_files = list_tsv_gz_files(args.outputs_dir, start_date, end_date)

    stats: Dict[str, AddrStats] = {}

    print(f"Using {len(output_files)} output shards and {len(input_files)} input shards")

    # Outputs: only need recipient + value
    for path in output_files:
        for row in iter_selected_tsv_gz(path, ["recipient", "value"]):
            recipient = row["recipient"].strip()
            if not looks_like_btc_address(recipient):
                continue

            s = get_stats(stats, recipient)
            s.receive_count += 1
            s.receive_value_sats += int_or_zero(row["value"])

    # Inputs: only need recipient + value + spending_transaction_hash
    for path in input_files:
        for row in iter_selected_tsv_gz(path, ["recipient", "value", "spending_transaction_hash"]):
            recipient = row["recipient"].strip()
            if not looks_like_btc_address(recipient):
                continue

            spend_tx = row["spending_transaction_hash"].strip()
            if not spend_tx:
                continue

            s = get_stats(stats, recipient)
            s.spend_count += 1
            s.spend_value_sats += int_or_zero(row["value"])

    ranked_total = sorted(
        stats.values(),
        key=lambda x: (x.total_count, x.receive_value_sats + x.spend_value_sats),
        reverse=True,
    )

    ranked_balanced = sorted(
        [x for x in stats.values() if x.receive_count >= 100 and x.spend_count >= 100],
        key=lambda x: (x.balance_delta, -x.total_count),
    )

    ranked_receive_heavy = sorted(
        [x for x in stats.values() if x.receive_count >= 100],
        key=lambda x: (x.receive_count, x.receive_value_sats),
        reverse=True,
    )

    ranked_spend_heavy = sorted(
        [x for x in stats.values() if x.spend_count >= 100],
        key=lambda x: (x.spend_count, x.spend_value_sats),
        reverse=True,
    )

    print("\nTop overall candidates:\n")
    for row in ranked_total[: args.top]:
        print(
            f"{row.address} | total={row.total_count} "
            f"| recv={row.receive_count} spend={row.spend_count} "
            f"| recv_btc={sats_to_btc_str(row.receive_value_sats)} "
            f"| spend_btc={sats_to_btc_str(row.spend_value_sats)}"
        )

    print("\nTop balanced hubs:\n")
    for row in ranked_balanced[: args.top]:
        print(
            f"{row.address} | total={row.total_count} "
            f"| recv={row.receive_count} spend={row.spend_count} "
            f"| delta={row.balance_delta}"
        )

    print("\nTop receive-heavy candidates:\n")
    for row in ranked_receive_heavy[: args.top]:
        print(
            f"{row.address} | recv={row.receive_count} "
            f"| spend={row.spend_count} "
            f"| recv_btc={sats_to_btc_str(row.receive_value_sats)}"
        )

    print("\nTop spend-heavy candidates:\n")
    for row in ranked_spend_heavy[: args.top]:
        print(
            f"{row.address} | spend={row.spend_count} "
            f"| recv={row.receive_count} "
            f"| spend_btc={sats_to_btc_str(row.spend_value_sats)}"
        )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())