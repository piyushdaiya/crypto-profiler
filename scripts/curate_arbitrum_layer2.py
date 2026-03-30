
#!/usr/bin/env python3
"""
curate_arbitrum_layer2.py

Transactions-only Phase 1 curator for Arbitrum.

Creates:
- arbitrum-repeated-contract-service-like
- arbitrum-broad-operational-hub
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


SERVICE_LIKE_ADDRESS = "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789"
HUB_ADDRESS = "0x3bdb03ad7363152dfbc185ee23ebc93f0cf93fd1"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Curate Arbitrum Layer 2 Phase 1 cases")
    parser.add_argument("--input-dir", required=True)
    parser.add_argument("--out-dir", required=True)
    return parser.parse_args()


def load_extract(path: Path) -> dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError(f"{path} did not load as an object")
    if "summary" not in data or data["summary"] is None:
        raise ValueError(f"{path} is missing summary")
    return data


def build_case(
    *,
    case_id: str,
    title: str,
    description: str,
    risk_posture: str,
    extract: dict[str, Any],
    narrative: str,
    selection_basis: str,
) -> dict[str, Any]:
    summary = extract["summary"]
    return {
        "chain": "ARBITRUM",
        "case_id": case_id,
        "title": title,
        "description": description,
        "risk_posture": risk_posture,
        "address": extract["address"],
        "window_start": extract["window_start"],
        "window_end": extract["window_end"],
        "layer2_summary": summary,
        "top_counterparties": extract.get("top_counterparties", []),
        "top_to_addresses": extract.get("top_to_addresses", []),
        "top_from_addresses": extract.get("top_from_addresses", []),
        "sample_transfers": extract.get("sample_transfers", []),
        "dataset_notes": extract.get("dataset_notes", {}),
        "curation_notes": {
            "narrative": narrative,
            "selection_basis": selection_basis,
        },
    }


def main() -> None:
    args = parse_args()
    input_dir = Path(args.input_dir)
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    service_like_path = input_dir / f"{SERVICE_LIKE_ADDRESS}.json"
    hub_path = input_dir / f"{HUB_ADDRESS}.json"

    service_like_extract = load_extract(service_like_path)
    hub_extract = load_extract(hub_path)

    service_like_case = build_case(
        case_id="arbitrum-repeated-contract-service-like",
        title="Arbitrum Repeated Contract Service-Like Wallet",
        description="Arbitrum Layer 2 case showing extremely concentrated single-contract transaction behavior with constrained counterparty diversity.",
        risk_posture="CONTEXTUAL_REPEATED_CONTRACT_SERVICE_LIKE",
        extract=service_like_extract,
        narrative=(
            "This wallet is dominated by one contract destination across a very large transaction volume with relatively low "
            "counterparty diversity. It is useful for demonstrating how repeated contract-centric activity can look "
            "operational or infrastructural rather than automatically suspicious."
        ),
        selection_basis=(
            "Selected because it showed 100% dominant contract share, very high transaction count, and low counterparty "
            "diversity relative to total activity in a tx-only Phase 1 Arbitrum workflow."
        ),
    )

    hub_case = build_case(
        case_id="arbitrum-broad-operational-hub",
        title="Arbitrum Broad Operational Hub",
        description="Arbitrum Layer 2 case showing broad mixed-flow operational activity across a very large counterparty surface.",
        risk_posture="REVIEWABLE_BROAD_OPERATIONAL_SURFACE",
        extract=hub_extract,
        narrative=(
            "This wallet shows broad mixed inbound/outbound behavior across a very large counterparty surface. "
            "It is useful for demonstrating operational-hub style Layer 2 behavior that should be reviewed but not "
            "treated as illicit by default."
        ),
        selection_basis=(
            "Selected because it combined broad unique counterparties, mixed flow direction, and very low "
            "dominant-counterparty concentration in a transactions-only Arbitrum Phase 1 workflow."
        ),
    )

    service_like_out = out_dir / "arbitrum-repeated-contract-service-like.json"
    hub_out = out_dir / "arbitrum-broad-operational-hub.json"

    service_like_out.write_text(json.dumps(service_like_case, indent=2, default=str), encoding="utf-8")
    hub_out.write_text(json.dumps(hub_case, indent=2, default=str), encoding="utf-8")

    manifest = {
        "script": "curate_arbitrum_layer2.py",
        "input_dir": str(input_dir),
        "out_dir": str(out_dir),
        "case_ids": [
            "arbitrum-repeated-contract-service-like",
            "arbitrum-broad-operational-hub",
        ],
    }
    (out_dir / "curation_manifest.json").write_text(
        json.dumps(manifest, indent=2, default=str),
        encoding="utf-8",
    )

    print(f"Wrote {service_like_out}")
    print(f"Wrote {hub_out}")
    print(f"Wrote {out_dir / 'curation_manifest.json'}")


if __name__ == "__main__":
    main()
