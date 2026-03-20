#!/usr/bin/env python3
"""
Create first curated Solana case artifacts from extracted stablecoin summaries.

Input:
- extracted summary JSON files from scripts/extract_solana_stablecoin.py

Output:
- curated case JSON files under data/cases/curated-solana/

Example:
  python3 scripts/curate_solana_stablecoin.py \
    --in data/cases/extracted-solana-stablecoin \
    --out data/cases/curated-solana
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
from pathlib import Path
from typing import Any, Dict


CASE_DEFS: Dict[str, Dict[str, str]] = {
    "Bgq7trRgVMeq33yt235zM2onQ4bRDBsY5EWiTetF4qw6": {
        "case_id": "solana-usdc-distributor-treasury-like",
        "title": "Solana USDC Distributor Treasury-Like Wallet",
        "label": "SOLANA STABLECOIN DISTRIBUTOR (TREASURY-LIKE)",
        "risk_posture": "REVIEWABLE_OPERATIONAL_CONCENTRATION",
        "description": (
            "High-volume source-heavy USDC wallet showing treasury/exchange-like "
            "distribution behavior across a broad stablecoin counterparty surface."
        ),
        "narrative": (
            "This case is dominated by outbound USDC activity and looks more like a large "
            "operational distributor or treasury-style wallet than a retail user. It is "
            "useful as a Solana Layer 1 benchmark for concentration, repeated large-value "
            "flows, and stablecoin-heavy service behavior."
        ),
    },
    "9DrvZvyWh1HuAoZxvYWMvkf2XCzryCpGgHqrMjyDWpmo": {
        "case_id": "solana-stablecoin-authority-operator",
        "title": "Solana Stablecoin Authority-Heavy Operator",
        "label": "SOLANA STABLECOIN AUTHORITY OPERATOR",
        "risk_posture": "REVIEWABLE_AUTHORITY_CONCENTRATION",
        "description": (
            "Authority-heavy stablecoin operator controlling a broad transfer surface, with "
            "very large value concentration and repeated operational linkage to major counterparties."
        ),
        "narrative": (
            "This case is dominated by authority-driven stablecoin flows rather than direct "
            "source or destination behavior. It is a strong benchmark for authority-role "
            "analysis, repeated interaction, and operational control-surface patterns."
        ),
    },
    "5tzFkiKscXHK5ZXCGbXZxdw7gTjjD1mBwuoFbhUvuAi9": {
        "case_id": "solana-broad-surface-authority-mixed-stablecoin",
        "title": "Solana Broad-Surface Authority with Mixed Stablecoins",
        "label": "SOLANA BROAD AUTHORITY SURFACE (MIXED STABLECOINS)",
        "risk_posture": "REVIEWABLE_BROAD_COUNTERPARTY_SURFACE",
        "description": (
            "Authority-heavy wallet with a very broad counterparty surface and mixed USDC/USDT "
            "activity, suitable for noisy-authority and reviewable broad-surface flow analysis."
        ),
        "narrative": (
            "This case mixes USDC and USDT while touching an unusually wide authority-linked "
            "counterparty set relative to its row count. It is useful for broad-surface review, "
            "mixed stablecoin activity, and repeated authority-driven flow analysis."
        ),
    },
}


def now_iso() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def load_json(path: Path) -> Dict[str, Any]:
    with path.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def detect_dominant_role(summary: Dict[str, Any]) -> str:
    roles = {
        "source": int(summary.get("source_transfer_count", 0)),
        "destination": int(summary.get("destination_transfer_count", 0)),
        "authority": int(summary.get("authority_transfer_count", 0)),
    }
    return max(roles.items(), key=lambda kv: kv[1])[0]


def detect_dominant_mint(mint_breakdown: list[dict]) -> str:
    if not mint_breakdown:
        return "UNKNOWN"
    return mint_breakdown[0].get("mint", "UNKNOWN")


def build_case(extracted: Dict[str, Any], case_def: Dict[str, str]) -> Dict[str, Any]:
    summary = extracted.get("summary", {})
    mint_breakdown = extracted.get("mint_breakdown", [])
    top_counterparties = extracted.get("top_counterparties", [])
    top_authority_pairs = extracted.get("top_authority_pairs", [])
    sample_transfers = extracted.get("sample_transfers", [])

    dominant_role = detect_dominant_role(summary)
    dominant_mint = detect_dominant_mint(mint_breakdown)

    return {
        "case_id": case_def["case_id"],
        "title": case_def["title"],
        "description": case_def["description"],
        "risk_posture": case_def["risk_posture"],
        "label": case_def["label"],
        "address": extracted.get("address"),
        "chain": "SOLANA",
        "generated_at": now_iso(),
        "source_dataset_type": extracted.get("dataset_type"),
        "source_row_count": extracted.get("source_row_count", 0),
        "stablecoin_summary": {
            "first_seen": summary.get("first_seen"),
            "last_seen": summary.get("last_seen"),
            "source_transfer_count": summary.get("source_transfer_count", 0),
            "destination_transfer_count": summary.get("destination_transfer_count", 0),
            "authority_transfer_count": summary.get("authority_transfer_count", 0),
            "source_value_raw": summary.get("source_value_raw", "0"),
            "destination_value_raw": summary.get("destination_value_raw", "0"),
            "authority_value_raw": summary.get("authority_value_raw", "0"),
            "unique_counterparties": summary.get("unique_counterparties", 0),
            "source_counterparties": summary.get("source_counterparties", 0),
            "destination_counterparties": summary.get("destination_counterparties", 0),
            "authority_counterparties": summary.get("authority_counterparties", 0),
            "dominant_role": dominant_role,
            "dominant_mint": dominant_mint,
        },
        "mint_breakdown": mint_breakdown,
        "transfer_type_breakdown": extracted.get("transfer_type_breakdown", []),
        "top_counterparties": top_counterparties[:20],
        "top_authority_pairs": top_authority_pairs[:20],
        "sample_transfers": sample_transfers[:50],
        "curation_notes": {
            "narrative": case_def["narrative"],
            "solana_layer": "Layer 1 stablecoin-flow dataset",
            "intended_typologies": [
                "repeated interaction with counterparties",
                "service / treasury concentration",
                "authority-driven operational control",
                "stablecoin-heavy behavior",
            ],
            "limitations": [
                "Derived from large-value stablecoin transfer rows only",
                "Does not yet include full Solana instruction/program context",
                "Not an authoritative label of illicit activity",
            ],
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Curate first Solana stablecoin cases")
    parser.add_argument("--in", dest="in_dir", required=True, help="Input extracted-solana-stablecoin directory")
    parser.add_argument("--out", required=True, help="Output curated-solana directory")
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