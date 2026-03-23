#!/usr/bin/env python3
"""
Create first curated Bitcoin Layer 1 case artifacts from extracted Bitcoin summaries.

Input:
- extracted summary JSON files from scripts/extract_bitcoin_layer1.py

Output:
- curated case JSON files under data/cases/curated-bitcoin/

Example:
  python3 scripts/curate_bitcoin_layer1.py \
    --in data/cases/extracted-bitcoin \
    --out data/cases/curated-bitcoin
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
from pathlib import Path
from typing import Any, Dict


CASE_DEFS: Dict[str, Dict[str, str]] = {
    "bc1q8ys49pxp3c6um7enemwdkl4ud5fwwg2rpdegxu": {
        "case_id": "bitcoin-broad-spend-heavy-operational-hub",
        "title": "Bitcoin Broad Spend-Heavy Operational Hub",
        "label": "BITCOIN SPEND-HEAVY OPERATIONAL HUB",
        "risk_posture": "REVIEWABLE_OPERATIONAL_CONCENTRATION",
        "description": (
            "Broad spend-heavy Bitcoin wallet showing high outbound flow activity, "
            "repeated interaction concentration, and service-like operational behavior."
        ),
        "narrative": (
            "This case is dominated by outbound spend activity and has a broad but still "
            "structured counterparty surface. It is useful as a Bitcoin Layer 1 benchmark "
            "for repeated interaction, spend-heavy operational flow, and service-like UTXO routing."
        ),
    },
    "bc1qp8j9sx6609h7llqufurxjgrwsqwt020tqzn0gs": {
        "case_id": "bitcoin-noisy-inbound-broad-surface",
        "title": "Bitcoin Noisy Inbound Broad-Surface Wallet",
        "label": "BITCOIN NOISY INBOUND BROAD SURFACE",
        "risk_posture": "OBSERVATIONAL_BROAD_SURFACE",
        "description": (
            "Receive-heavy Bitcoin wallet with extremely broad counterparty exposure and "
            "minimal relative outbound activity, suitable for noisy inbound observational analysis."
        ),
        "narrative": (
            "This case has a very large inbound surface and far less outbound activity, making it "
            "a strong observational benchmark for noisy inbound behavior, broad public-facing exposure, "
            "and low-signal counterparty sprawl."
        ),
    },
    "1FWQiwK27EnGXb6BiBMRLJvunJQZZPMcGd": {
        "case_id": "bitcoin-legacy-mixed-flow-broad-value",
        "title": "Bitcoin Legacy Mixed-Flow Broad-Value Wallet",
        "label": "BITCOIN LEGACY MIXED-FLOW BROAD VALUE",
        "risk_posture": "REVIEWABLE_MIXED_FLOW_CONCENTRATION",
        "description": (
            "Legacy-format Bitcoin wallet showing mixed inbound and outbound value movement with "
            "very broad counterparty coverage and high total BTC value across the observed window."
        ),
        "narrative": (
            "This case is useful as a legacy Bitcoin benchmark with both inbound and outbound value flow, "
            "broad counterparty exposure, and a large BTC throughput profile. It helps diversify the Bitcoin "
            "case set beyond bech32-only service-like patterns."
        ),
    },
}


def now_iso() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def load_json(path: Path) -> Dict[str, Any]:
    with path.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def detect_dominant_role(summary: Dict[str, Any]) -> str:
    inbound = int(summary.get("inbound_receipt_count", 0))
    outbound = int(summary.get("outbound_spend_count", 0))

    if outbound > inbound:
        return "outbound"
    if inbound > outbound:
        return "inbound"
    return "balanced"


def build_case(extracted: Dict[str, Any], case_def: Dict[str, str]) -> Dict[str, Any]:
    summary = extracted.get("summary", {})
    top_counterparties = extracted.get("top_counterparties", [])
    sample_events = extracted.get("sample_events", [])

    dominant_role = detect_dominant_role(summary)

    return {
        "case_id": case_def["case_id"],
        "title": case_def["title"],
        "description": case_def["description"],
        "risk_posture": case_def["risk_posture"],
        "label": case_def["label"],
        "address": extracted.get("address"),
        "chain": "BITCOIN",
        "generated_at": now_iso(),
        "source_dataset_type": extracted.get("dataset_type"),
        "source_row_count": extracted.get("source_row_count", 0),
        "utxo_summary": {
            "first_seen": summary.get("first_seen"),
            "last_seen": summary.get("last_seen"),
            "inbound_receipt_count": summary.get("inbound_receipt_count", 0),
            "outbound_spend_count": summary.get("outbound_spend_count", 0),
            "inbound_value_sats": summary.get("inbound_value_sats", 0),
            "outbound_value_sats": summary.get("outbound_value_sats", 0),
            "inbound_value_btc": summary.get("inbound_value_btc", "0"),
            "outbound_value_btc": summary.get("outbound_value_btc", "0"),
            "unique_counterparties": summary.get("unique_counterparties", 0),
            "counterparty_resolution_mode": summary.get("counterparty_resolution_mode", ""),
            "dominant_role": dominant_role,
        },
        "top_counterparties": top_counterparties[:20],
        "sample_events": sample_events[:50],
        "curation_notes": {
            "narrative": case_def["narrative"],
            "bitcoin_layer": "Layer 1 UTXO flow dataset",
            "intended_typologies": [
                "repeated interaction with counterparties",
                "service / operational concentration",
                "noisy inbound broad-surface behavior",
                "mixed-flow high-value wallet behavior",
            ],
            "limitations": [
                "Derived from Blockchair input/output UTXO rows only",
                "Counterparties are approximated from transaction input/output recipients",
                "Not an authoritative label of illicit activity",
            ],
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Curate first Bitcoin Layer 1 cases")
    parser.add_argument("--in", dest="in_dir", required=True, help="Input extracted-bitcoin directory")
    parser.add_argument("--out", required=True, help="Output curated-bitcoin directory")
    args = parser.parse_args()

    in_dir = Path(args.in_dir).expanduser().resolve()
    out_dir = Path(args.out).expanduser().resolve()
    out_dir.mkdir(parents=True, exist_ok=True)

    for address, case_def in CASE_DEFS.items():
        in_path = in_dir / f"{address}.json"
        if not in_path.exists():
            raise FileNotFoundError(f"missing extracted summary: {in_path}")

        extracted = load_json(in_path)
        curated = build_case(extracted, case_def)

        out_path = out_dir / f"{case_def['case_id']}.json"
        with out_path.open("w", encoding="utf-8") as fh:
            json.dump(curated, fh, indent=2)
            fh.write("\n")

        print(f"wrote {out_path}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())