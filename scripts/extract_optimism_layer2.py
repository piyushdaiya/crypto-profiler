
# scripts/extract_optimism_layer2.py
#!/usr/bin/env python3
"""
Phase 1 Optimism address-scoped extractor for Crypto Profiler.

This version is shortlist-aware and batched:
- reads a shortlist file
- runs one batched transactions query
- runs one batched decoded-events query
- groups results by profiled address locally
- writes one JSON extract per address

Example:
python3 scripts/extract_optimism_layer2.py \
  --project "$GOOGLE_CLOUD_PROJECT" \
  --window-start 2025-03-16T00:00:00Z \
  --window-end 2025-06-17T00:00:00Z \
  --addresses-file data/candidates/optimism_phase1_shortlist.txt \
  --out-dir "/Volumes/Extreme SSD/crypto-profiler/optimism/extracted"
"""

from __future__ import annotations

import argparse
import json
import os
from collections import Counter, defaultdict
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterable

from google.cloud import bigquery

DEFAULT_DATASET = "bigquery-public-data.goog_blockchain_optimism_mainnet_us"
DEFAULT_MAX_BYTES_BILLED = 120 * 1024 * 1024 * 1024  # 120 GiB
DEFAULT_TOP_N = 25
DEFAULT_SAMPLE_LIMIT = 100


@dataclass
class TransferRow:
    profiled_address: str
    transaction_hash: str
    block_timestamp: str
    from_address: str
    to_address: str
    value_wei: str
    direction: str
    counterparty: str


@dataclass
class EventRow:
    profiled_address: str
    transaction_hash: str
    block_timestamp: str
    contract_address: str
    event_signature: str
    event_hash: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Extract Optimism address-scoped summaries from BigQuery")
    parser.add_argument("--project", default=os.getenv("GOOGLE_CLOUD_PROJECT", ""))
    parser.add_argument("--dataset", default=DEFAULT_DATASET)
    parser.add_argument("--window-start", required=True)
    parser.add_argument("--window-end", required=True)
    parser.add_argument("--addresses-file", required=True, help="Text file with one lowercase address per line")
    parser.add_argument("--out-dir", required=True)
    parser.add_argument("--max-bytes-billed", type=int, default=DEFAULT_MAX_BYTES_BILLED)
    parser.add_argument("--top-n", type=int, default=DEFAULT_TOP_N)
    parser.add_argument("--sample-limit", type=int, default=DEFAULT_SAMPLE_LIMIT)
    return parser.parse_args()


def load_addresses(path: Path) -> list[str]:
    addresses: list[str] = []
    for raw in path.read_text(encoding="utf-8").splitlines():
        value = raw.strip().lower()
        if value:
            addresses.append(value)
    unique = sorted(set(addresses))
    if not unique:
        raise SystemExit(f"No addresses found in {path}")
    return unique


def top_n_counter(counter: Counter[str], n: int) -> list[dict[str, Any]]:
    return [{"key": key, "count": count} for key, count in counter.most_common(n)]


def query_batched_transfers(
    client: bigquery.Client,
    *,
    dataset: str,
    addresses: list[str],
    window_start: str,
    window_end: str,
    max_bytes_billed: int,
) -> list[TransferRow]:
    query = f"""
WITH shortlist AS (
  SELECT address
  FROM UNNEST(@addresses) AS address
),
txs AS (
  SELECT
    transaction_hash,
    block_timestamp,
    LOWER(from_address) AS from_address,
    LOWER(to_address) AS to_address,
    value
  FROM `{dataset}.transactions`
  WHERE block_timestamp >= TIMESTAMP(@window_start)
    AND block_timestamp < TIMESTAMP(@window_end)
    AND from_address IS NOT NULL
    AND to_address IS NOT NULL
),
profiled_from AS (
  SELECT
    s.address AS profiled_address,
    t.transaction_hash,
    CAST(t.block_timestamp AS STRING) AS block_timestamp,
    t.from_address,
    t.to_address,
    COALESCE(t.value.string_value, CAST(t.value.bignumeric_value AS STRING), "0") AS value_wei,
    "outbound" AS direction,
    t.to_address AS counterparty
  FROM txs t
  JOIN shortlist s
    ON t.from_address = s.address
  WHERE t.from_address != t.to_address
),
profiled_to AS (
  SELECT
    s.address AS profiled_address,
    t.transaction_hash,
    CAST(t.block_timestamp AS STRING) AS block_timestamp,
    t.from_address,
    t.to_address,
    COALESCE(t.value.string_value, CAST(t.value.bignumeric_value AS STRING), "0") AS value_wei,
    "inbound" AS direction,
    t.from_address AS counterparty
  FROM txs t
  JOIN shortlist s
    ON t.to_address = s.address
  WHERE t.from_address != t.to_address
)
SELECT * FROM profiled_from
UNION ALL
SELECT * FROM profiled_to
ORDER BY profiled_address, block_timestamp DESC
"""
    job_config = bigquery.QueryJobConfig(
        query_parameters=[
            bigquery.ArrayQueryParameter("addresses", "STRING", addresses),
            bigquery.ScalarQueryParameter("window_start", "STRING", window_start),
            bigquery.ScalarQueryParameter("window_end", "STRING", window_end),
        ],
        maximum_bytes_billed=max_bytes_billed,
        use_query_cache=True,
    )
    result = client.query(query, job_config=job_config).result()

    rows: list[TransferRow] = []
    for row in result:
        rows.append(
            TransferRow(
                profiled_address=row["profiled_address"],
                transaction_hash=row["transaction_hash"],
                block_timestamp=row["block_timestamp"],
                from_address=row["from_address"],
                to_address=row["to_address"],
                value_wei=row["value_wei"],
                direction=row["direction"],
                counterparty=row["counterparty"],
            )
        )
    return rows


def query_batched_events(
    client: bigquery.Client,
    *,
    dataset: str,
    addresses: list[str],
    window_start: str,
    window_end: str,
    max_bytes_billed: int,
) -> list[EventRow]:
    query = f"""
WITH shortlist AS (
  SELECT address
  FROM UNNEST(@addresses) AS address
),
tx_hashes AS (
  SELECT DISTINCT
    s.address AS profiled_address,
    t.transaction_hash
  FROM `{dataset}.transactions` t
  JOIN shortlist s
    ON LOWER(t.from_address) = s.address
    OR LOWER(t.to_address) = s.address
  WHERE t.block_timestamp >= TIMESTAMP(@window_start)
    AND t.block_timestamp < TIMESTAMP(@window_end)
    AND t.from_address IS NOT NULL
    AND t.to_address IS NOT NULL
),
events AS (
  SELECT
    th.profiled_address,
    e.transaction_hash,
    CAST(e.block_timestamp AS STRING) AS block_timestamp,
    LOWER(e.address) AS contract_address,
    COALESCE(e.event_signature, "") AS event_signature,
    COALESCE(e.event_hash, "") AS event_hash
  FROM `{dataset}.decoded_events` e
  JOIN tx_hashes th
    ON e.transaction_hash = th.transaction_hash
  WHERE e.block_timestamp >= TIMESTAMP(@window_start)
    AND e.block_timestamp < TIMESTAMP(@window_end)
)
SELECT *
FROM events
ORDER BY profiled_address, block_timestamp DESC
"""
    job_config = bigquery.QueryJobConfig(
        query_parameters=[
            bigquery.ArrayQueryParameter("addresses", "STRING", addresses),
            bigquery.ScalarQueryParameter("window_start", "STRING", window_start),
            bigquery.ScalarQueryParameter("window_end", "STRING", window_end),
        ],
        maximum_bytes_billed=max_bytes_billed,
        use_query_cache=True,
    )
    result = client.query(query, job_config=job_config).result()

    rows: list[EventRow] = []
    for row in result:
        rows.append(
            EventRow(
                profiled_address=row["profiled_address"],
                transaction_hash=row["transaction_hash"],
                block_timestamp=row["block_timestamp"],
                contract_address=row["contract_address"],
                event_signature=row["event_signature"],
                event_hash=row["event_hash"],
            )
        )
    return rows


def group_by_address[T](rows: Iterable[T], key_name: str) -> dict[str, list[T]]:
    grouped: dict[str, list[T]] = defaultdict(list)
    for row in rows:
        grouped[getattr(row, key_name)].append(row)
    return grouped


def detect_bridge_signatures(event_signature: str) -> bool:
    sig = (event_signature or "").lower()
    bridge_hints = (
        "deposit",
        "withdraw",
        "relay",
        "bridge",
        "send",
        "receive",
        "finalize",
    )
    return any(hint in sig for hint in bridge_hints)


def detect_stablecoin_signatures(event_signature: str) -> bool:
    sig = (event_signature or "").lower()
    stablecoin_hints = (
        "transfer",
        "approval",
        "mint",
        "burn",
        "usdc",
        "usdt",
        "dai",
    )
    return any(hint in sig for hint in stablecoin_hints)


def build_extract(
    *,
    address: str,
    window_start: str,
    window_end: str,
    transfers: list[TransferRow],
    events: list[EventRow],
    top_n: int,
    sample_limit: int,
) -> dict[str, Any]:
    counterparties = Counter[str]()
    contracts = Counter[str]()
    signatures = Counter[str]()
    event_hashes = Counter[str]()

    inbound_count = 0
    outbound_count = 0

    timestamps: list[str] = []

    for transfer in transfers:
        counterparties[transfer.counterparty] += 1
        timestamps.append(transfer.block_timestamp)
        if transfer.direction == "inbound":
            inbound_count += 1
        elif transfer.direction == "outbound":
            outbound_count += 1

    for event in events:
        contracts[event.contract_address] += 1
        if event.event_signature:
            signatures[event.event_signature] += 1
        if event.event_hash:
            event_hashes[event.event_hash] += 1

    first_seen = min(timestamps) if timestamps else ""
    last_seen = max(timestamps) if timestamps else ""

    bridge_breakdown = [
        {"key": key, "count": count}
        for key, count in signatures.most_common(top_n)
        if detect_bridge_signatures(key)
    ]
    stablecoin_breakdown = [
        {"key": key, "count": count}
        for key, count in signatures.most_common(top_n)
        if detect_stablecoin_signatures(key)
    ]

    dominant_direction = "mixed"
    if outbound_count > inbound_count:
        dominant_direction = "outbound"
    elif inbound_count > outbound_count:
        dominant_direction = "inbound"

    total_transfer_rows = max(len(transfers), 1)
    total_event_rows = max(len(events), 1)

    dominant_counterparty_share = 0.0
    if counterparties:
        dominant_counterparty_share = counterparties.most_common(1)[0][1] / total_transfer_rows

    dominant_contract_share = 0.0
    if contracts:
        dominant_contract_share = contracts.most_common(1)[0][1] / total_event_rows

    return {
        "chain": "OPTIMISM",
        "address": address,
        "window_start": window_start,
        "window_end": window_end,
        "summary": {
            "first_seen": first_seen,
            "last_seen": last_seen,
            "tx_count": len(transfers),
            "inbound_count": inbound_count,
            "outbound_count": outbound_count,
            "unique_counterparties": len(counterparties),
            "unique_contracts": len(contracts),
            "protocol_event_count": len(events),
            "bridge_event_count": sum(item["count"] for item in bridge_breakdown),
            "stablecoin_event_count": sum(item["count"] for item in stablecoin_breakdown),
            "dominant_direction": dominant_direction,
            "dominant_counterparty_share": round(dominant_counterparty_share * 100, 2),
            "dominant_contract_share": round(dominant_contract_share * 100, 2),
        },
        "top_counterparties": top_n_counter(counterparties, top_n),
        "contract_breakdown": top_n_counter(contracts, top_n),
        "protocol_breakdown": top_n_counter(signatures, top_n),
        "bridge_breakdown": bridge_breakdown,
        "stablecoin_breakdown": stablecoin_breakdown,
        "sample_transfers": [asdict(item) for item in transfers[:sample_limit]],
        "sample_events": [asdict(item) for item in events[:sample_limit]],
        "dataset_notes": {
            "source": "Google Cloud Blockchain Analytics / BigQuery",
            "window_name": "canonical_90d",
            "address_normalization": "lowercase",
            "extraction_mode": "batched_shortlist",
        },
    }


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2), encoding="utf-8")


def main() -> None:
    args = parse_args()
    if not args.project:
        raise SystemExit("Missing --project and GOOGLE_CLOUD_PROJECT is not set")

    addresses = load_addresses(Path(args.addresses_file))
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    client = bigquery.Client(project=args.project)

    print(f"Loaded {len(addresses)} shortlisted addresses")
    transfers = query_batched_transfers(
        client,
        dataset=args.dataset,
        addresses=addresses,
        window_start=args.window_start,
        window_end=args.window_end,
        max_bytes_billed=args.max_bytes_billed,
    )
    print(f"Fetched {len(transfers)} batched transfer rows")

    events = query_batched_events(
        client,
        dataset=args.dataset,
        addresses=addresses,
        window_start=args.window_start,
        window_end=args.window_end,
        max_bytes_billed=args.max_bytes_billed,
    )
    print(f"Fetched {len(events)} batched event rows")

    transfers_by_address = group_by_address(transfers, "profiled_address")
    events_by_address = group_by_address(events, "profiled_address")

    manifest: dict[str, Any] = {
        "script": "extract_optimism_layer2.py",
        "project": args.project,
        "dataset": args.dataset,
        "window_start": args.window_start,
        "window_end": args.window_end,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "max_bytes_billed": args.max_bytes_billed,
        "addresses": [],
    }

    for address in addresses:
        payload = build_extract(
            address=address,
            window_start=args.window_start,
            window_end=args.window_end,
            transfers=transfers_by_address.get(address, []),
            events=events_by_address.get(address, []),
            top_n=args.top_n,
            sample_limit=args.sample_limit,
        )
        out_path = out_dir / f"{address}.json"
        write_json(out_path, payload)

        manifest["addresses"].append(
            {
                "address": address,
                "extract_path": str(out_path),
                "transfer_rows": len(transfers_by_address.get(address, [])),
                "event_rows": len(events_by_address.get(address, [])),
            }
        )
        print(f"Wrote extract for {address} -> {out_path}")

    manifest_path = out_dir / "extract_manifest.json"
    write_json(manifest_path, manifest)
    print(f"Wrote manifest -> {manifest_path}")


if __name__ == "__main__":
    main()
