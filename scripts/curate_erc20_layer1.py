#!/usr/bin/env python3
"""
Create curated ERC-20 Layer 1 case artifacts from extracted ERC-20 summaries.

Input:
- extracted summary JSON files from scripts/extract_erc20_layer1.py

Output:
- curated case JSON files under data/cases/curated-erc20/

Example:
  python3 scripts/curate_erc20_layer1.py \
    --in /tmp/crypto-profiler-erc20-extracted \
    --out data/cases/curated-erc20
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
from pathlib import Path
from typing import Any, Dict, List


DEFAULT_LIMITATIONS: List[str] = [
    "Derived from ERC-20 transfer rows only",
    "Does not yet include trace-native ERC-20 scoring or swap decoding",
    "Not an authoritative label of illicit activity",
]


CASE_DEFS: Dict[str, Dict[str, Any]] = {
    "0x7a250d5630b4cf539739df2c5dacb4c659f2488d": {
        "case_id": "erc20-uniswap-v2-router-trusted-token-hub",
        "title": "ERC-20 Trusted Protocol Token Hub",
        "label": "PROTOCOL: Uniswap V2 Router",
        "risk_posture": "LOW_REVIEWABLE_PROTOCOL_HUB",
        "description": (
            "Trusted protocol router with a large ERC-20 transfer surface, broad counterparty reach, "
            "and token-diverse operational flow."
        ),
        "narrative": (
            "This case captures token-heavy ERC-20 routing activity at a known trusted protocol address. "
            "It is useful for demonstrating how a very broad token surface can still be contextualized as "
            "protocol-driven rather than automatically high-risk."
        ),
        "typologies": [
            "trusted protocol token hub",
            "broad ERC-20 counterparty surface",
            "mixed token activity",
        ],
    },
    "0x28c6c06298d514db089934071355e5743bf21d60": {
        "case_id": "erc20-exchange-like-broad-service-surface",
        "title": "ERC-20 Exchange-Like Broad Service Surface",
        "label": "EXCHANGE-LIKE TOKEN SERVICE SURFACE",
        "risk_posture": "LOW_REVIEWABLE_SERVICE_SURFACE",
        "description": (
            "Inbound-heavy ERC-20 service-like address with broad counterparty reach, many token contracts, "
            "and repeated settlement-style interaction patterns."
        ),
        "narrative": (
            "This case is useful as an ERC-20 benchmark for exchange-like or service-style token activity: "
            "broad but structured counterparty exposure, repeated settlement interactions, and a high-volume "
            "token surface that should not be interpreted the same way as a noisy inbound address."
        ),
        "typologies": [
            "exchange-style service surface",
            "repeated ERC-20 counterparty interaction",
            "token-heavy operational behavior",
            "broad ERC-20 counterparty surface",
        ],
    },
    "0xf89d7b9c864f589bbf53a82105107622b35eaa40": {
        "case_id": "erc20-noisy-inbound-broad-token-surface",
        "title": "ERC-20 Noisy Inbound Broad Token Surface",
        "label": "NOISY TOKEN INBOUND SURFACE",
        "risk_posture": "REVIEWABLE_NOISY_TOKEN_SURFACE",
        "description": (
            "Inbound-heavy ERC-20 address with many counterparties, many token contracts, and a broad token "
            "surface consistent with noisy distribution or spam-like inbound flow."
        ),
        "narrative": (
            "This case is intentionally different from the trusted protocol and operational surface examples. "
            "It highlights an inbound-heavy token surface where breadth and token diversity are the main signals, "
            "rather than a trusted label or protocol context."
        ),
        "typologies": [
            "noisy token inbound observation",
            "broad ERC-20 counterparty surface",
            "mixed token activity",
        ],
    },
}


def now_iso() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def load_json(path: Path) -> Dict[str, Any]:
    with path.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def build_case(extracted: Dict[str, Any], case_def: Dict[str, Any]) -> Dict[str, Any]:
    summary = extracted.get("summary", {})
    token_breakdown = extracted.get("token_breakdown", [])
    top_counterparties = extracted.get("top_counterparties", [])
    sample_transfers = extracted.get("sample_transfers", [])

    return {
        "case_id": case_def["case_id"],
        "title": case_def["title"],
        "description": case_def["description"],
        "risk_posture": case_def["risk_posture"],
        "label": case_def["label"],
        "address": extracted.get("address"),
        "chain": "EVM",
        "generated_at": now_iso(),
        "source_dataset_type": extracted.get("dataset_type"),
        "source_row_count": extracted.get("source_row_count", 0),
        "raw_subset_file": extracted.get("raw_subset_file", ""),
        "erc20_summary": {
            "first_seen": summary.get("first_seen"),
            "last_seen": summary.get("last_seen"),
            "inbound_transfer_count": summary.get("inbound_transfer_count", 0),
            "outbound_transfer_count": summary.get("outbound_transfer_count", 0),
            "self_transfer_count": summary.get("self_transfer_count", 0),
            "inbound_counterparties": summary.get("inbound_counterparties", 0),
            "outbound_counterparties": summary.get("outbound_counterparties", 0),
            "unique_counterparties": summary.get("unique_counterparties", 0),
            "unique_token_contracts": summary.get("unique_token_contracts", 0),
            "repeated_counterparties": summary.get("repeated_counterparties", 0),
            "max_counterparty_interactions": summary.get("max_counterparty_interactions", 0),
            "dominant_direction": summary.get("dominant_direction", ""),
            "dominant_token_address": summary.get("dominant_token_address", ""),
            "dominant_token_symbol": summary.get("dominant_token_symbol", ""),
            "dominant_token_transfer_share_pct": summary.get("dominant_token_transfer_share_pct", 0),
        },
        "token_breakdown": token_breakdown[:20],
        "top_counterparties": top_counterparties[:20],
        "sample_transfers": sample_transfers[:50],
        "curation_notes": {
            "narrative": case_def["narrative"],
            "erc20_layer": "Layer 1 ERC-20 transfer dataset",
            "intended_typologies": case_def.get("typologies", []),
            "limitations": case_def.get("limitations", DEFAULT_LIMITATIONS),
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Curate ERC-20 Layer 1 cases")
    parser.add_argument("--in", dest="in_dir", required=True, help="Input extracted-erc20 directory")
    parser.add_argument("--out", required=True, help="Output curated-erc20 directory")
    args = parser.parse_args()

    in_dir = Path(args.in_dir).expanduser().resolve()
    out_dir = Path(args.out).expanduser().resolve()
    out_dir.mkdir(parents=True, exist_ok=True)

    for address, case_def in CASE_DEFS.items():
        in_path = in_dir / f"{address[2:]}.json"
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
