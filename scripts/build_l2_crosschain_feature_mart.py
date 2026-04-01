#!/usr/bin/env python3
"""
Build a normalized cross-chain L2 feature mart from Phase 2 extracts.

Input:
- one or more Phase 2 extract directories for:
  - Optimism
  - Polygon
  - Arbitrum

Output:
- l2_crosschain_feature_mart.json
- l2_crosschain_feature_mart.csv
- l2_crosschain_feature_mart_manifest.json
"""

from __future__ import annotations

import argparse
import csv
import json
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build cross-chain L2 feature mart")
    parser.add_argument("--optimism-dir", required=False)
    parser.add_argument("--polygon-dir", required=False)
    parser.add_argument("--arbitrum-dir", required=False)
    parser.add_argument("--out-dir", required=True)
    return parser.parse_args()


def load_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def normalize_address(value: Any) -> str:
    if value is None:
        return ""
    s = str(value).strip().lower()
    if s.startswith("0x") and len(s) == 42:
        return s
    return ""


def pick_top_family(hits: Any, key_name: str) -> str:
    if not isinstance(hits, list) or not hits:
        return ""
    best = None
    best_count = -1
    for item in hits:
        if not isinstance(item, dict):
            continue
        family = str(item.get(key_name, "")).strip()
        if not family:
            continue
        try:
            observed_count = int(item.get("observed_count", 0))
        except Exception:
            observed_count = 0
        if observed_count > best_count:
            best = family
            best_count = observed_count
    return best or ""


def bool_service_like(row: dict[str, Any]) -> bool:
    return (
        row["tx_count"] >= 100000
        and row["dominant_contract_share"] >= 95.0
        and row["unique_counterparties"] <= 2000
        and row["inbound_count"] > 0
    )


def bool_operational_hub(row: dict[str, Any]) -> bool:
    return (
        row["unique_counterparties"] >= 100000
        and row["inbound_count"] > 0
        and row["outbound_count"] > 0
    )


def bool_bridge_or_stablecoin_surface(row: dict[str, Any]) -> bool:
    return (
        row["unique_counterparties"] >= 25000
        and (
            row["bridge_hit_count"] > 0
            or row["stablecoin_hit_count"] > 0
            or row["bridge_topic_hit_count"] > 0
        )
    )


def build_row(payload: dict[str, Any]) -> dict[str, Any]:
    summary = payload.get("summary", {})
    receipt_summary = payload.get("receipt_summary", {})
    log_summary = payload.get("log_summary", {})
    registry_hits = payload.get("registry_hits", {})
    registry_summary = registry_hits.get("summary", {})

    row = {
        "chain": str(payload.get("chain", "")).strip().upper(),
        "address": normalize_address(payload.get("address")),
        "window_start": str(payload.get("window_start", "")).strip(),
        "window_end": str(payload.get("window_end", "")).strip(),
        "tx_count": int(summary.get("tx_count", 0)),
        "inbound_count": int(summary.get("inbound_count", 0)),
        "outbound_count": int(summary.get("outbound_count", 0)),
        "unique_counterparties": int(summary.get("unique_counterparties", 0)),
        "dominant_contract_share": float(summary.get("dominant_contract_share", 0.0)),
        "dominant_counterparty_share": float(summary.get("dominant_counterparty_share", 0.0)),
        "failure_rate_pct": float(receipt_summary.get("failure_rate_pct", 0.0)),
        "success_count": int(receipt_summary.get("success_count", 0)),
        "failure_count": int(receipt_summary.get("failure_count", 0)),
        "unique_emitters": int(log_summary.get("unique_emitters", 0)),
        "unique_topic0s": int(log_summary.get("unique_topic0s", 0)),
        "bridge_hit_count": int(registry_summary.get("bridge_contract_match_count", 0)),
        "protocol_hit_count": int(registry_summary.get("protocol_contract_match_count", 0)),
        "stablecoin_hit_count": int(registry_summary.get("stablecoin_contract_match_count", 0)),
        "service_hit_count": int(registry_summary.get("service_contract_match_count", 0)),
        "bridge_topic_hit_count": int(registry_summary.get("bridge_topic_match_count", 0)),
        "protocol_topic_hit_count": int(registry_summary.get("protocol_topic_match_count", 0)),
        "matched_bridge_families": registry_summary.get("matched_bridge_families", []),
        "matched_protocol_families": registry_summary.get("matched_protocol_families", []),
        "matched_topic_families": registry_summary.get("matched_topic_families", []),
        "top_bridge_family": pick_top_family(registry_hits.get("bridge_contract_hits", []), "bridge_family_id"),
        "top_protocol_family": pick_top_family(registry_hits.get("protocol_contract_hits", []), "family_id"),
        "top_stablecoin_family": pick_top_family(registry_hits.get("stablecoin_contract_hits", []), "family_id"),
    }

    row["service_like_candidate"] = bool_service_like(row)
    row["operational_hub_candidate"] = bool_operational_hub(row)
    row["bridge_or_stablecoin_candidate"] = bool_bridge_or_stablecoin_surface(row)
    return row


def load_rows_from_dir(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for file_path in sorted(path.glob("*.json")):
        if file_path.name.endswith("manifest.json"):
            continue
        raw = load_json(file_path)
        if not isinstance(raw, dict):
            continue
        address = normalize_address(raw.get("address"))
        if not address:
            continue
        rows.append(build_row(raw))
    return rows


def main() -> None:
    args = parse_args()
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    all_rows: list[dict[str, Any]] = []

    for chain_name, dir_value in (
        ("OPTIMISM", args.optimism_dir),
        ("POLYGON", args.polygon_dir),
        ("ARBITRUM", args.arbitrum_dir),
    ):
        if not dir_value:
            continue
        chain_dir = Path(dir_value)
        chain_rows = load_rows_from_dir(chain_dir)
        all_rows.extend(chain_rows)
        print(f"Loaded {len(chain_rows)} {chain_name} Phase 2 rows from {chain_dir}")

    all_rows.sort(key=lambda row: (row["chain"], row["address"]))

    json_path = out_dir / "l2_crosschain_feature_mart.json"
    json_path.write_text(json.dumps(all_rows, indent=2, default=str), encoding="utf-8")

    csv_path = out_dir / "l2_crosschain_feature_mart.csv"
    fieldnames = [
        "chain",
        "address",
        "window_start",
        "window_end",
        "tx_count",
        "inbound_count",
        "outbound_count",
        "unique_counterparties",
        "dominant_contract_share",
        "dominant_counterparty_share",
        "failure_rate_pct",
        "success_count",
        "failure_count",
        "unique_emitters",
        "unique_topic0s",
        "bridge_hit_count",
        "protocol_hit_count",
        "stablecoin_hit_count",
        "service_hit_count",
        "bridge_topic_hit_count",
        "protocol_topic_hit_count",
        "top_bridge_family",
        "top_protocol_family",
        "top_stablecoin_family",
        "service_like_candidate",
        "operational_hub_candidate",
        "bridge_or_stablecoin_candidate",
    ]
    with csv_path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames)
        writer.writeheader()
        for row in all_rows:
            writer.writerow({k: row.get(k) for k in fieldnames})

    manifest = {
        "script": "build_l2_crosschain_feature_mart.py",
        "row_count": len(all_rows),
        "chains": {
            "OPTIMISM": sum(1 for row in all_rows if row["chain"] == "OPTIMISM"),
            "POLYGON": sum(1 for row in all_rows if row["chain"] == "POLYGON"),
            "ARBITRUM": sum(1 for row in all_rows if row["chain"] == "ARBITRUM"),
        },
        "json_path": str(json_path),
        "csv_path": str(csv_path),
    }
    manifest_path = out_dir / "l2_crosschain_feature_mart_manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, default=str), encoding="utf-8")

    print(f"Wrote {json_path}")
    print(f"Wrote {csv_path}")
    print(f"Wrote {manifest_path}")


if __name__ == "__main__":
    main()