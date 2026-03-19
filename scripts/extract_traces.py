#!/usr/bin/env python3
import argparse
import datetime as dt
import gzip
import json
import os
import sys
from collections import defaultdict
from dataclasses import dataclass, field
from decimal import Decimal, InvalidOperation
from typing import Dict, List, Optional

import pyarrow.dataset as ds


def normalize_evm_address(value: Optional[str]) -> str:
    if value is None:
        return ""
    s = str(value).strip().lower()
    if not s:
        return ""
    if s.startswith("0x"):
        return s
    if len(s) == 40 and all(ch in "0123456789abcdef" for ch in s):
        return "0x" + s
    return s


def is_positive_decimal(value: Optional[str]) -> bool:
    if value is None:
        return False
    s = str(value).strip()
    if not s:
        return False
    try:
        return Decimal(s) > 0
    except (InvalidOperation, ValueError):
        return False


def isoformat_or_none(value):
    if value is None:
        return None
    if isinstance(value, dt.datetime):
        if value.tzinfo is None:
            return value.isoformat() + "Z"
        return value.astimezone(dt.timezone.utc).isoformat().replace("+00:00", "Z")
    return str(value)


def json_trace_row(row: dict) -> dict:
    out = {
        "block_id": int(row["block_id"]) if row.get("block_id") is not None else None,
        "time": isoformat_or_none(row.get("time")),
        "transaction_hash": row.get("transaction_hash"),
        "transaction_index": int(row["transaction_index"]) if row.get("transaction_index") is not None else None,
        "trace_path": row.get("trace_path") or "",
        "depth": int(row["depth"]) if row.get("depth") is not None else 0,
        "trace_type": row.get("trace_type") or "",
        "call_type": row.get("call_type") or "",
        "sender": normalize_evm_address(row.get("sender")),
        "recipient": normalize_evm_address(row.get("recipient")),
        "value": str(row.get("value")) if row.get("value") is not None else "0",
        "gas": int(row["gas"]) if row.get("gas") is not None else None,
        "gas_used": int(row["gas_used"]) if row.get("gas_used") is not None else None,
        "child_call_count": int(row["child_call_count"]) if row.get("child_call_count") is not None else 0,
        "status": int(row["status"]) if row.get("status") is not None else None,
        "failed": bool(row["failed"]) if row.get("failed") is not None else False,
        "fail_reason": row.get("fail_reason") or "",
    }
    return out


@dataclass
class CounterpartySummary:
    address: str
    interactions: int = 0
    inbound_count: int = 0
    outbound_count: int = 0
    value_transfer_count: int = 0
    failed_count: int = 0


@dataclass
class TraceDatasetBuilder:
    address: str
    out_dir: str
    sample_limit: int
    top_limit: int
    generated_at: str = field(default_factory=lambda: dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z"))
    first_seen: Optional[str] = None
    last_seen: Optional[str] = None
    inbound_trace_count: int = 0
    outbound_trace_count: int = 0
    self_trace_count: int = 0
    failed_trace_count: int = 0
    value_transfer_trace_count: int = 0
    max_depth: int = 0
    source_trace_count: int = 0
    unique_counterparties: set = field(default_factory=set)
    counterparties: Dict[str, CounterpartySummary] = field(default_factory=dict)
    sample_traces: List[dict] = field(default_factory=list)

    def __post_init__(self):
        os.makedirs(self.out_dir, exist_ok=True)
        self.raw_filename = f"{self.address[2:]}.traces.ndjson.gz"
        self.raw_path = os.path.join(self.out_dir, self.raw_filename)
        self._fh = gzip.open(self.raw_path, "wt", encoding="utf-8")

    def close(self):
        self._fh.close()

    def add(self, trace: dict, direction: str):
        self.source_trace_count += 1

        trace_time = trace.get("time")
        if self.first_seen is None or (trace_time and trace_time < self.first_seen):
            self.first_seen = trace_time
        if self.last_seen is None or (trace_time and trace_time > self.last_seen):
            self.last_seen = trace_time

        depth = int(trace.get("depth") or 0)
        if depth > self.max_depth:
            self.max_depth = depth

        if trace.get("failed"):
            self.failed_trace_count += 1

        if is_positive_decimal(trace.get("value")):
            self.value_transfer_trace_count += 1

        counterparty = ""
        if direction == "inbound":
            self.inbound_trace_count += 1
            counterparty = trace.get("sender") or ""
        elif direction == "outbound":
            self.outbound_trace_count += 1
            counterparty = trace.get("recipient") or ""
        else:
            self.self_trace_count += 1

        if counterparty and counterparty != self.address:
            self.unique_counterparties.add(counterparty)
            summary = self.counterparties.get(counterparty)
            if summary is None:
                summary = CounterpartySummary(address=counterparty)
                self.counterparties[counterparty] = summary
            summary.interactions += 1
            if direction == "inbound":
                summary.inbound_count += 1
            elif direction == "outbound":
                summary.outbound_count += 1
            if is_positive_decimal(trace.get("value")):
                summary.value_transfer_count += 1
            if trace.get("failed"):
                summary.failed_count += 1

        if len(self.sample_traces) < self.sample_limit:
            self.sample_traces.append(trace)

        self._fh.write(json.dumps(trace, separators=(",", ":")) + "\n")

    def metadata(self) -> dict:
        top_counterparties = sorted(
            self.counterparties.values(),
            key=lambda x: (-x.interactions, x.address),
        )[: self.top_limit]

        return {
            "address": self.address,
            "chain": "EVM",
            "generated_at": self.generated_at,
            "summary": {
                "first_seen": self.first_seen,
                "last_seen": self.last_seen,
                "inbound_trace_count": self.inbound_trace_count,
                "outbound_trace_count": self.outbound_trace_count,
                "self_trace_count": self.self_trace_count,
                "failed_trace_count": self.failed_trace_count,
                "value_transfer_trace_count": self.value_transfer_trace_count,
                "unique_counterparties": len(self.unique_counterparties),
                "max_depth": self.max_depth,
            },
            "top_counterparties": [
                {
                    "address": cp.address,
                    "interactions": cp.interactions,
                    "inbound_count": cp.inbound_count,
                    "outbound_count": cp.outbound_count,
                    "value_transfer_count": cp.value_transfer_count,
                    "failed_count": cp.failed_count,
                }
                for cp in top_counterparties
            ],
            "sample_traces": self.sample_traces,
            "source_trace_count": self.source_trace_count,
            "raw_trace_file": self.raw_filename,
        }


def load_addresses(path: str) -> List[str]:
    addresses = []
    seen = set()
    with open(path, "r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            addr = normalize_evm_address(line)
            if not addr or addr in seen:
                continue
            seen.add(addr)
            addresses.append(addr)
    return addresses


def main():
    parser = argparse.ArgumentParser(description="Extract address-scoped EVM trace datasets from Parquet export")
    parser.add_argument("--parquet", required=True, help="Directory or glob for exported trace parquet files")
    parser.add_argument("--addresses", required=True, help="Text file with one EVM address per line")
    parser.add_argument("--out", required=True, help="Output directory for extracted trace datasets")
    parser.add_argument("--sample", type=int, default=200, help="Sample trace count to retain in summary JSON")
    parser.add_argument("--top", type=int, default=20, help="Top counterparties to keep in summary JSON")
    args = parser.parse_args()

    addresses = load_addresses(args.addresses)
    if not addresses:
        print("error: no valid addresses loaded", file=sys.stderr)
        sys.exit(1)

    if os.path.isdir(args.parquet):
        parquet_source = os.path.join(args.parquet, "*.parquet")
    else:
        parquet_source = args.parquet

    os.makedirs(args.out, exist_ok=True)

    builders = {addr: TraceDatasetBuilder(addr, args.out, args.sample, args.top) for addr in addresses}

    columns = [
        "block_id",
        "time",
        "transaction_hash",
        "transaction_index",
        "trace_path",
        "depth",
        "trace_type",
        "call_type",
        "sender",
        "recipient",
        "value",
        "gas",
        "gas_used",
        "child_call_count",
        "status",
        "failed",
        "fail_reason",
    ]

    dataset = ds.dataset(parquet_source, format="parquet")
    filter_expr = ds.field("sender").isin(addresses) | ds.field("recipient").isin(addresses)
    scanner = dataset.scanner(columns=columns, filter=filter_expr, batch_size=50000, use_threads=True)

    processed = 0
    try:
        for batch in scanner.to_batches():
            pyd = batch.to_pydict()
            size = len(next(iter(pyd.values()))) if pyd else 0
            for i in range(size):
                raw_row = {k: pyd[k][i] for k in columns}
                trace = json_trace_row(raw_row)
                sender = trace["sender"]
                recipient = trace["recipient"]

                sender_tracked = sender in builders
                recipient_tracked = recipient in builders

                if sender_tracked:
                    direction = "self" if recipient == sender else "outbound"
                    builders[sender].add(trace, direction)

                if recipient_tracked and recipient != sender:
                    builders[recipient].add(trace, "inbound")

                processed += 1
                if processed % 250000 == 0:
                    print(f"processed {processed} matched trace rows...", file=sys.stderr)
    finally:
        for builder in builders.values():
            builder.close()

    for addr, builder in builders.items():
        meta_path = os.path.join(args.out, f"{addr[2:]}.json")
        with open(meta_path, "w", encoding="utf-8") as fh:
            json.dump(builder.metadata(), fh, indent=2)
            fh.write("\n")
        print(
            f"wrote {meta_path} "
            f"(source_traces={builder.source_trace_count}, counterparties={len(builder.unique_counterparties)}, failed={builder.failed_trace_count})"
        )


if __name__ == "__main__":
    main()
