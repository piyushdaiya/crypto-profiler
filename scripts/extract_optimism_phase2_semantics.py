#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

import pyarrow.dataset as ds


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Merge Optimism Phase 2 parquet summaries into extracted JSON artifacts"
    )
    parser.add_argument("--input-dir", required=True, help="Registry-enriched extract directory")
    parser.add_argument("--tx-dir", required=False, help="Optional parquet directory for tx hashes")
    parser.add_argument("--receipt-dir", required=True, help="Parquet directory for receipt summary")
    parser.add_argument("--log-dir", required=True, help="Parquet directory for log summary")
    parser.add_argument("--out-dir", required=True, help="Output directory for final Optimism Phase 2 extracts")
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    raw = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(raw, dict):
        raise ValueError(f"{path} did not contain a JSON object")
    return raw


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.write_text(json.dumps(payload, indent=2, default=str), encoding="utf-8")


def normalize_address(value: Any) -> str:
    if value is None:
        return ""
    s = str(value).strip().lower()
    if s.startswith("0x") and len(s) == 42:
        return s
    return ""


def normalize_key_count_list(items: Any) -> list[dict[str, Any]]:
    out: list[dict[str, Any]] = []
    if not isinstance(items, list):
        return out

    for item in items:
        if not isinstance(item, dict):
            continue
        key = str(item.get("key", "")).strip()
        if not key:
            continue
        try:
            count = int(item.get("count", 0))
        except Exception:
            count = 0
        out.append({"key": key, "count": count})

    return out


def empty_receipt_summary() -> dict[str, Any]:
    return {
        "tx_count": 0,
        "success_count": 0,
        "failure_count": 0,
        "failure_rate_pct": 0.0,
    }


def empty_log_summary() -> dict[str, Any]:
    return {
        "unique_emitters": 0,
        "unique_topic0s": 0,
        "top_emitters": [],
        "top_topic0s": [],
    }


def empty_registry_hits() -> dict[str, Any]:
    return {
        "bridge_contract_hits": [],
        "protocol_contract_hits": [],
        "stablecoin_contract_hits": [],
        "service_contract_hits": [],
        "bridge_topic_hits": [],
        "protocol_topic_hits": [],
        "summary": {
            "observed_address_count": 0,
            "observed_topic0_count": 0,
            "bridge_contract_match_count": 0,
            "protocol_contract_match_count": 0,
            "stablecoin_contract_match_count": 0,
            "service_contract_match_count": 0,
            "bridge_topic_match_count": 0,
            "protocol_topic_match_count": 0,
            "matched_bridge_families": [],
            "matched_protocol_families": [],
            "matched_topic_families": [],
        },
    }


def load_parquet_rows(parquet_dir: Path) -> list[dict[str, Any]]:
    if not parquet_dir.exists():
        return []
    parquet_files = [p for p in parquet_dir.glob("*.parquet") if p.is_file() and not p.name.startswith("._")]
    if not parquet_files:
        return []
    table = ds.dataset(str(parquet_dir), format="parquet").to_table()
    return table.to_pylist()


def load_payload_index(input_dir: Path) -> dict[str, dict[str, Any]]:
    index: dict[str, dict[str, Any]] = {}
    for path in sorted(input_dir.glob("*.json")):
        if path.name.startswith("._"):
            continue
        if path.name.endswith("manifest.json"):
            continue
        payload = load_json(path)
        address = normalize_address(payload.get("address"))
        chain = str(payload.get("chain", "")).strip().upper()
        if not address or chain != "OPTIMISM":
            continue
        index[address] = payload
    return index


def load_receipt_index(parquet_dir: Path) -> dict[str, dict[str, Any]]:
    index: dict[str, dict[str, Any]] = {}
    for row in load_parquet_rows(parquet_dir):
        address = normalize_address(row.get("address"))
        if not address:
            continue
        index[address] = {
            "tx_count": int(row.get("tx_count", 0)),
            "success_count": int(row.get("success_count", 0)),
            "failure_count": int(row.get("failure_count", 0)),
            "failure_rate_pct": round(float(row.get("failure_rate_pct", 0.0)), 4),
        }
    return index


def load_log_index(parquet_dir: Path) -> dict[str, dict[str, Any]]:
    index: dict[str, dict[str, Any]] = {}
    for row in load_parquet_rows(parquet_dir):
        address = normalize_address(row.get("address"))
        if not address:
            continue
        index[address] = {
            "unique_emitters": int(row.get("unique_emitters", 0)),
            "unique_topic0s": int(row.get("unique_topic0s", 0)),
            "top_emitters": normalize_key_count_list(row.get("top_emitters", [])),
            "top_topic0s": normalize_key_count_list(row.get("top_topic0s", [])),
        }
    return index


def coerce_registry_hits(obj: Any) -> dict[str, Any]:
    if not isinstance(obj, dict):
        return empty_registry_hits()

    out = empty_registry_hits()
    for field in (
        "bridge_contract_hits",
        "protocol_contract_hits",
        "stablecoin_contract_hits",
        "service_contract_hits",
        "bridge_topic_hits",
        "protocol_topic_hits",
    ):
        value = obj.get(field, [])
        if isinstance(value, list):
            out[field] = value

    summary = obj.get("summary", {})
    if isinstance(summary, dict):
        merged = dict(out["summary"])
        merged.update(summary)
        out["summary"] = merged

    return out


def enrich_payload(
    base_payload: dict[str, Any],
    receipt_index: dict[str, dict[str, Any]],
    log_index: dict[str, dict[str, Any]],
) -> dict[str, Any]:
    address = normalize_address(base_payload.get("address"))
    if not address:
        raise ValueError("Payload missing valid address")

    enriched = dict(base_payload)
    enriched["schema_version"] = "0.2.0"

    dataset_notes = dict(enriched.get("dataset_notes", {}))
    dataset_notes["extraction_mode"] = "transactions_plus_phase2_semantics"
    dataset_notes["phase2_enrichment_mode"] = "receipts_logs_registry"
    enriched["dataset_notes"] = dataset_notes

    enriched["receipt_summary"] = receipt_index.get(address, empty_receipt_summary())
    enriched["log_summary"] = log_index.get(address, empty_log_summary())
    enriched["registry_hits"] = coerce_registry_hits(enriched.get("registry_hits"))

    return enriched


def main() -> None:
    args = parse_args()

    input_dir = Path(args.input_dir)
    out_dir = Path(args.out_dir)
    receipt_dir = Path(args.receipt_dir)
    log_dir = Path(args.log_dir)
    tx_dir = Path(args.tx_dir) if args.tx_dir else None

    out_dir.mkdir(parents=True, exist_ok=True)

    payload_index = load_payload_index(input_dir)
    receipt_index = load_receipt_index(receipt_dir)
    log_index = load_log_index(log_dir)

    written_files: list[str] = []
    skipped_files: list[str] = []

    for address, payload in payload_index.items():
        try:
            enriched = enrich_payload(payload, receipt_index, log_index)
            out_path = out_dir / f"{address}.json"
            write_json(out_path, enriched)
            written_files.append(out_path.name)
            print(f"Wrote {out_path}")
        except Exception as exc:
            skipped_files.append(f"{address}.json")
            print(f"Skipped {address}: {exc}")

    manifest = {
        "script": "extract_optimism_phase2_semantics.py",
        "input_dir": str(input_dir),
        "tx_dir": str(tx_dir) if tx_dir else "",
        "receipt_dir": str(receipt_dir),
        "log_dir": str(log_dir),
        "out_dir": str(out_dir),
        "written_count": len(written_files),
        "skipped_count": len(skipped_files),
        "written_files": written_files,
        "skipped_files": skipped_files,
    }
    write_json(out_dir / "phase2_extract_manifest.json", manifest)
    print(f"Wrote {out_dir / 'phase2_extract_manifest.json'}")


if __name__ == "__main__":
    main()