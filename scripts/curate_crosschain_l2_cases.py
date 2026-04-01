#!/usr/bin/env python3
"""
Curate cross-chain L2 cases from the feature mart.

Case families:
- crosschain-l2-repeated-contract-service-pattern
- crosschain-l2-broad-operational-hub
- crosschain-l2-stablecoin-bridge-operational-surface

This script is intentionally heuristic and explanation-first.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Curate cross-chain L2 cases from feature mart")
    parser.add_argument("--mart-file", required=True, help="Path to l2_crosschain_feature_mart.json")
    parser.add_argument("--out-dir", required=True)
    return parser.parse_args()


def load_rows(path: Path) -> list[dict[str, Any]]:
    raw = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(raw, list):
        raise ValueError(f"{path} did not contain a mart row list")
    rows = [row for row in raw if isinstance(row, dict)]
    return rows


def compact_row(row: dict[str, Any]) -> dict[str, Any]:
    return {
        "chain": row.get("chain", ""),
        "address": row.get("address", ""),
        "tx_count": row.get("tx_count", 0),
        "inbound_count": row.get("inbound_count", 0),
        "outbound_count": row.get("outbound_count", 0),
        "unique_counterparties": row.get("unique_counterparties", 0),
        "dominant_contract_share": row.get("dominant_contract_share", 0.0),
        "failure_rate_pct": row.get("failure_rate_pct", 0.0),
        "unique_emitters": row.get("unique_emitters", 0),
        "unique_topic0s": row.get("unique_topic0s", 0),
        "bridge_hit_count": row.get("bridge_hit_count", 0),
        "protocol_hit_count": row.get("protocol_hit_count", 0),
        "stablecoin_hit_count": row.get("stablecoin_hit_count", 0),
        "service_hit_count": row.get("service_hit_count", 0),
        "top_bridge_family": row.get("top_bridge_family", ""),
        "top_protocol_family": row.get("top_protocol_family", ""),
        "top_stablecoin_family": row.get("top_stablecoin_family", ""),
    }


def write_case(out_dir: Path, payload: dict[str, Any]) -> None:
    case_id = str(payload["case_id"])
    out_path = out_dir / f"{case_id}.json"
    out_path.write_text(json.dumps(payload, indent=2, default=str), encoding="utf-8")
    print(f"Wrote {out_path}")


def build_service_case(rows: list[dict[str, Any]]) -> dict[str, Any] | None:
    service_rows = [row for row in rows if bool(row.get("service_like_candidate"))]
    if len(service_rows) < 2:
        return None

    by_address: dict[str, list[dict[str, Any]]] = {}
    for row in service_rows:
        by_address.setdefault(str(row["address"]), []).append(row)

    grouped = sorted(
        by_address.values(),
        key=lambda group: (-len(group), -sum(int(item["tx_count"]) for item in group)),
    )
    chosen = grouped[0]

    return {
        "schema_version": "0.3.0",
        "case_family": "crosschain_l2",
        "case_id": "crosschain-l2-repeated-contract-service-pattern",
        "title": "Cross-Chain L2 Repeated Contract Service Pattern",
        "description": "Cross-chain case showing repeated service-like contract concentration across multiple L2s.",
        "chains_included": sorted({row["chain"] for row in chosen}),
        "member_count": len(chosen),
        "members": [compact_row(row) for row in chosen],
        "crosschain_summary": {
            "address_count": len({row["address"] for row in chosen}),
            "chain_count": len({row["chain"] for row in chosen}),
            "total_tx_count": sum(int(row["tx_count"]) for row in chosen),
            "max_dominant_contract_share": max(float(row["dominant_contract_share"]) for row in chosen),
        },
        "curation_notes": {
            "narrative": "This case captures the same or very similar service-like repeated-contract pattern across multiple L2s.",
            "selection_basis": "Selected from Phase 2 mart rows where dominant contract share is extremely high and counterparty diversity remains relatively constrained for very large transaction volume.",
        },
    }


def build_hub_case(rows: list[dict[str, Any]]) -> dict[str, Any] | None:
    hub_rows = [row for row in rows if bool(row.get("operational_hub_candidate"))]
    if len(hub_rows) < 2:
        return None

    chosen = sorted(
        hub_rows,
        key=lambda row: (-int(row["unique_counterparties"]), -int(row["tx_count"])),
    )[:3]

    return {
        "schema_version": "0.3.0",
        "case_family": "crosschain_l2",
        "case_id": "crosschain-l2-broad-operational-hub",
        "title": "Cross-Chain L2 Broad Operational Hub",
        "description": "Cross-chain case showing very broad mixed-flow operational surfaces across multiple L2s.",
        "chains_included": sorted({row["chain"] for row in chosen}),
        "member_count": len(chosen),
        "members": [compact_row(row) for row in chosen],
        "crosschain_summary": {
            "address_count": len({row["address"] for row in chosen}),
            "chain_count": len({row["chain"] for row in chosen}),
            "total_tx_count": sum(int(row["tx_count"]) for row in chosen),
            "max_unique_counterparties": max(int(row["unique_counterparties"]) for row in chosen),
        },
        "curation_notes": {
            "narrative": "This case captures operational-hub style behavior across more than one L2, with broad counterparty reach and mixed inbound/outbound flows.",
            "selection_basis": "Selected from mart rows that show high counterparty breadth, material inbound and outbound flow, and operational-hub style transaction shape.",
        },
    }


def build_bridge_stablecoin_case(rows: list[dict[str, Any]]) -> dict[str, Any] | None:
    candidate_rows = [row for row in rows if bool(row.get("bridge_or_stablecoin_candidate"))]
    if len(candidate_rows) < 2:
        return None

    chosen = sorted(
        candidate_rows,
        key=lambda row: (
            -(int(row["bridge_hit_count"]) + int(row["stablecoin_hit_count"]) + int(row["bridge_topic_hit_count"])),
            -int(row["unique_counterparties"]),
        ),
    )[:3]

    return {
        "schema_version": "0.3.0",
        "case_family": "crosschain_l2",
        "case_id": "crosschain-l2-stablecoin-bridge-operational-surface",
        "title": "Cross-Chain L2 Stablecoin / Bridge Operational Surface",
        "description": "Cross-chain case showing bridge- or stablecoin-adjacent operational surfaces across multiple L2s.",
        "chains_included": sorted({row["chain"] for row in chosen}),
        "member_count": len(chosen),
        "members": [compact_row(row) for row in chosen],
        "crosschain_summary": {
            "address_count": len({row["address"] for row in chosen}),
            "chain_count": len({row["chain"] for row in chosen}),
            "total_tx_count": sum(int(row["tx_count"]) for row in chosen),
            "bridge_or_stablecoin_member_count": len(chosen),
        },
        "curation_notes": {
            "narrative": "This case captures cross-chain operational surfaces where registry hits suggest bridge or stablecoin context rather than purely generic contract traffic.",
            "selection_basis": "Selected from mart rows with bridge-contract, bridge-topic, or stablecoin registry support and material counterparty breadth.",
        },
    }


def main() -> None:
    args = parse_args()
    mart_rows = load_rows(Path(args.mart_file))
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    cases: list[dict[str, Any]] = []

    service_case = build_service_case(mart_rows)
    if service_case:
        cases.append(service_case)

    hub_case = build_hub_case(mart_rows)
    if hub_case:
        cases.append(hub_case)

    bridge_case = build_bridge_stablecoin_case(mart_rows)
    if bridge_case:
        cases.append(bridge_case)

    for case in cases:
        write_case(out_dir, case)

    manifest = {
        "script": "curate_crosschain_l2_cases.py",
        "mart_file": str(Path(args.mart_file)),
        "out_dir": str(out_dir),
        "case_count": len(cases),
        "case_ids": [case["case_id"] for case in cases],
    }
    manifest_path = out_dir / "crosschain_l2_curation_manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, default=str), encoding="utf-8")
    print(f"Wrote {manifest_path}")


if __name__ == "__main__":
    main()