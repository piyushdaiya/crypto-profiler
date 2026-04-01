#!/usr/bin/env python3
"""
Shared helpers for L2 Phase 2 semantic extract merging.

Expected Phase 2 workflow:
1. Start from extracted Phase 1 JSON artifacts.
2. Run build_l2_phase2_registry_hits.py to add `registry_hits`.
3. On April 1, run BigQuery semantic aggregations and save:
   - receipt summary JSON
   - log summary JSON
4. Use chain-specific Phase 2 scripts to merge everything into final Phase 2 extracts.

Accepted receipt summary input shapes:
- {"0xaddr": {...}, "0xaddr2": {...}}
- [{"address": "0xaddr", ...}, {"address": "0xaddr2", ...}]
- {"items": [{"address": "0xaddr", ...}, ...]}

Accepted log summary input shapes:
- {"0xaddr": {...}, "0xaddr2": {...}}
- [{"address": "0xaddr", ...}, {"address": "0xaddr2", ...}]
- {"items": [{"address": "0xaddr", ...}, ...]}
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any


SUPPORTED_CHAINS = {"OPTIMISM", "POLYGON", "ARBITRUM"}


def load_json_any(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


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


def coerce_receipt_summary(obj: Any) -> dict[str, Any]:
    if not isinstance(obj, dict):
        return empty_receipt_summary()

    try:
        tx_count = int(obj.get("tx_count", 0))
    except Exception:
        tx_count = 0
    try:
        success_count = int(obj.get("success_count", 0))
    except Exception:
        success_count = 0
    try:
        failure_count = int(obj.get("failure_count", 0))
    except Exception:
        failure_count = 0
    try:
        failure_rate_pct = float(obj.get("failure_rate_pct", 0.0))
    except Exception:
        failure_rate_pct = 0.0

    return {
        "tx_count": tx_count,
        "success_count": success_count,
        "failure_count": failure_count,
        "failure_rate_pct": round(failure_rate_pct, 4),
    }


def coerce_log_summary(obj: Any) -> dict[str, Any]:
    if not isinstance(obj, dict):
        return empty_log_summary()

    try:
        unique_emitters = int(obj.get("unique_emitters", 0))
    except Exception:
        unique_emitters = 0
    try:
        unique_topic0s = int(obj.get("unique_topic0s", 0))
    except Exception:
        unique_topic0s = 0

    return {
        "unique_emitters": unique_emitters,
        "unique_topic0s": unique_topic0s,
        "top_emitters": normalize_key_count_list(obj.get("top_emitters")),
        "top_topic0s": normalize_key_count_list(obj.get("top_topic0s")),
    }


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


def load_address_index_from_summary_file(path: Path | None, kind: str) -> dict[str, dict[str, Any]]:
    if path is None or not path.exists():
        return {}

    raw = load_json_any(path)
    index: dict[str, dict[str, Any]] = {}

    def add_entry(address: Any, payload: Any) -> None:
        addr = normalize_address(address)
        if not addr:
            return
        if kind == "receipt":
            index[addr] = coerce_receipt_summary(payload)
        elif kind == "log":
            index[addr] = coerce_log_summary(payload)

    if isinstance(raw, dict):
        if "items" in raw and isinstance(raw["items"], list):
            for item in raw["items"]:
                if not isinstance(item, dict):
                    continue
                add_entry(item.get("address"), item)
        else:
            all_address_keys = all(normalize_address(k) for k in raw.keys()) if raw else False
            if all_address_keys:
                for addr, payload in raw.items():
                    add_entry(addr, payload)
            elif "address" in raw:
                add_entry(raw.get("address"), raw)
    elif isinstance(raw, list):
        for item in raw:
            if not isinstance(item, dict):
                continue
            add_entry(item.get("address"), item)

    return index


def load_payload_index(input_dir: Path) -> dict[str, dict[str, Any]]:
    index: dict[str, dict[str, Any]] = {}
    for path in sorted(input_dir.glob("*.json")):
        if path.name.endswith("manifest.json"):
            continue
        raw = load_json_any(path)
        if not isinstance(raw, dict):
            continue
        address = normalize_address(raw.get("address"))
        chain = str(raw.get("chain", "")).strip().upper()
        if not address or chain not in SUPPORTED_CHAINS:
            continue
        index[address] = raw
    return index


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

    registry_hits = enriched.get("registry_hits")
    enriched["registry_hits"] = coerce_registry_hits(registry_hits)

    return enriched


def run_phase2_merge(
    *,
    expected_chain: str,
    input_dir: Path,
    receipt_summary_file: Path | None,
    log_summary_file: Path | None,
    out_dir: Path,
    script_name: str,
) -> None:
    payload_index = load_payload_index(input_dir)
    receipt_index = load_address_index_from_summary_file(receipt_summary_file, "receipt")
    log_index = load_address_index_from_summary_file(log_summary_file, "log")

    out_dir.mkdir(parents=True, exist_ok=True)

    written_files: list[str] = []
    skipped_files: list[str] = []

    for address, payload in payload_index.items():
        chain = str(payload.get("chain", "")).strip().upper()
        if chain != expected_chain:
            skipped_files.append(f"{address}.json")
            continue

        enriched = enrich_payload(
            base_payload=payload,
            receipt_index=receipt_index,
            log_index=log_index,
        )
        out_path = out_dir / f"{address}.json"
        write_json(out_path, enriched)
        written_files.append(out_path.name)
        print(f"Wrote {out_path}")

    manifest = {
        "script": script_name,
        "input_dir": str(input_dir),
        "out_dir": str(out_dir),
        "expected_chain": expected_chain,
        "receipt_summary_file": str(receipt_summary_file) if receipt_summary_file else "",
        "log_summary_file": str(log_summary_file) if log_summary_file else "",
        "written_count": len(written_files),
        "skipped_count": len(skipped_files),
        "written_files": written_files,
        "skipped_files": skipped_files,
    }
    manifest_path = out_dir / "phase2_extract_manifest.json"
    write_json(manifest_path, manifest)
    print(f"Wrote {manifest_path}")