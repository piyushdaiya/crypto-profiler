#!/usr/bin/env python3
"""
build_l2_phase2_registry_hits.py

Builds Phase 2 registry-hit enrichment for L2 extracted JSON artifacts.

Supported chains:
- OPTIMISM
- POLYGON
- ARBITRUM

What it does:
- reads extracted JSON artifacts from --input-dir
- matches observed addresses against:
  - data/registries/l2_bridge_registry.json
  - data/registries/l2_protocol_registry.json
- matches observed topic0 values (if present) against:
  - data/registries/l2_topic_registry.json
- writes enriched JSON artifacts with a new `registry_hits` section to --out-dir
- writes an enrichment manifest

This script is safe to run on:
- Phase 1 tx-only extracts
- future Phase 2 semantic extracts that add log_summary / receipt_summary

Expected useful input sections (best effort, not all required):
- address
- chain
- top_counterparties
- top_to_addresses
- top_from_addresses
- sample_transfers
- log_summary.top_emitters
- log_summary.top_topic0s
"""

from __future__ import annotations

import argparse
import json
from collections import Counter
from pathlib import Path
from typing import Any


SUPPORTED_CHAINS = {"optimism", "polygon", "arbitrum"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build Phase 2 registry hits for L2 extracted JSON artifacts")
    parser.add_argument("--input-dir", required=True, help="Directory containing extracted JSON artifacts")
    parser.add_argument("--out-dir", required=True, help="Directory to write enriched JSON artifacts")
    parser.add_argument(
        "--bridge-registry",
        default="data/registries/l2_bridge_registry.json",
        help="Path to bridge registry JSON",
    )
    parser.add_argument(
        "--protocol-registry",
        default="data/registries/l2_protocol_registry.json",
        help="Path to protocol registry JSON",
    )
    parser.add_argument(
        "--topic-registry",
        default="data/registries/l2_topic_registry.json",
        help="Path to topic registry JSON",
    )
    parser.add_argument(
        "--top-n",
        type=int,
        default=25,
        help="Maximum matched hits to keep per hit list",
    )
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError(f"{path} did not load as a JSON object")
    return data


def normalize_address(value: Any) -> str:
    if value is None:
        return ""
    s = str(value).strip().lower()
    if s.startswith("0x") and len(s) == 42:
        return s
    return ""


def normalize_topic0(value: Any) -> str:
    if value is None:
        return ""
    s = str(value).strip().lower()
    if s.startswith("0x") and len(s) == 66:
        return s
    return ""


def extract_chain(payload: dict[str, Any]) -> str:
    chain = str(payload.get("chain", "")).strip().lower()
    if chain not in SUPPORTED_CHAINS:
        return ""
    return chain


def iter_keyed_list(items: Any) -> list[tuple[str, int]]:
    """
    Accepts shapes like:
    - [{"key": "...", "count": 10}, ...]
    - [{"address": "...", "count": 10}, ...]
    - [{"topic0": "...", "count": 10}, ...]
    """
    out: list[tuple[str, int]] = []
    if not isinstance(items, list):
        return out

    for item in items:
        if not isinstance(item, dict):
            continue
        value = ""
        for candidate_key in ("key", "address", "topic0", "contract", "emitter"):
            if candidate_key in item:
                value = str(item.get(candidate_key, "")).strip()
                break
        if not value:
            continue
        count_raw = item.get("count", 0)
        try:
            count = int(count_raw)
        except Exception:
            count = 0
        out.append((value, count))
    return out


def collect_observed_addresses(payload: dict[str, Any]) -> Counter[str]:
    observed: Counter[str] = Counter()

    top_level_address = normalize_address(payload.get("address"))
    if top_level_address:
        observed[top_level_address] += 1

    for field in ("top_counterparties", "top_to_addresses", "top_from_addresses"):
        for value, count in iter_keyed_list(payload.get(field)):
            addr = normalize_address(value)
            if addr:
                observed[addr] += max(1, count)

    sample_transfers = payload.get("sample_transfers", [])
    if isinstance(sample_transfers, list):
        for row in sample_transfers:
            if not isinstance(row, dict):
                continue
            for field in ("from_address", "to_address", "counterparty", "profiled_address"):
                addr = normalize_address(row.get(field))
                if addr:
                    observed[addr] += 1

    log_summary = payload.get("log_summary", {})
    if isinstance(log_summary, dict):
        for field in ("top_emitters", "top_contracts"):
            for value, count in iter_keyed_list(log_summary.get(field)):
                addr = normalize_address(value)
                if addr:
                    observed[addr] += max(1, count)

    return observed


def collect_observed_topics(payload: dict[str, Any]) -> Counter[str]:
    observed: Counter[str] = Counter()

    log_summary = payload.get("log_summary", {})
    if isinstance(log_summary, dict):
        for value, count in iter_keyed_list(log_summary.get("top_topic0s")):
            topic0 = normalize_topic0(value)
            if topic0:
                observed[topic0] += max(1, count)

    return observed


def is_chain_relevant_bridge_contract(chain: str, family_chain: str, network_scope: str) -> bool:
    """
    Only match L2-side bridge contracts for current-chain extracts.
    L1 bridge addresses are intentionally excluded from L2 transaction extracts.
    """
    if chain != family_chain:
        return False

    scope = network_scope.strip().lower()
    if scope == "l2":
        return True
    if scope == f"{chain}_l2":
        return True
    return False


def match_bridge_contracts(
    chain: str,
    observed_addresses: Counter[str],
    bridge_registry: dict[str, Any],
    top_n: int,
) -> list[dict[str, Any]]:
    matches: list[dict[str, Any]] = []

    families = bridge_registry.get("families", [])
    if not isinstance(families, list):
        return matches

    for family in families:
        if not isinstance(family, dict):
            continue

        family_chain = str(family.get("chain", "")).strip().lower()
        family_id = str(family.get("bridge_family_id", "")).strip()
        display_name = str(family.get("display_name", "")).strip()
        source_tier = str(family.get("source_tier", "")).strip()
        tags = family.get("tags", [])
        if not isinstance(tags, list):
            tags = []

        contracts = family.get("contracts", [])
        if not isinstance(contracts, list):
            continue

        for contract in contracts:
            if not isinstance(contract, dict):
                continue

            network_scope = str(contract.get("network_scope", "")).strip()
            if not is_chain_relevant_bridge_contract(chain, family_chain, network_scope):
                continue

            address = normalize_address(contract.get("address"))
            if not address or address not in observed_addresses:
                continue

            matches.append(
                {
                    "bridge_family_id": family_id,
                    "display_name": display_name,
                    "role": str(contract.get("role", "")).strip(),
                    "address": address,
                    "network_scope": network_scope,
                    "source_tier": source_tier,
                    "confidence": str(contract.get("confidence", "")).strip(),
                    "observed_count": observed_addresses[address],
                    "tags": tags,
                    "match_source": "address",
                }
            )

    matches.sort(key=lambda item: (-int(item["observed_count"]), item["display_name"], item["address"]))
    return matches[:top_n]


def match_protocol_contracts(
    chain: str,
    observed_addresses: Counter[str],
    protocol_registry: dict[str, Any],
    top_n: int,
) -> tuple[list[dict[str, Any]], list[dict[str, Any]], list[dict[str, Any]]]:
    all_hits: list[dict[str, Any]] = []
    stablecoin_hits: list[dict[str, Any]] = []
    service_hits: list[dict[str, Any]] = []

    families = protocol_registry.get("families", [])
    if not isinstance(families, list):
        return all_hits, stablecoin_hits, service_hits

    for family in families:
        if not isinstance(family, dict):
            continue

        family_id = str(family.get("family_id", "")).strip()
        display_name = str(family.get("display_name", "")).strip()
        actor_class = str(family.get("actor_class", "")).strip()
        source_tier = str(family.get("source_tier", "")).strip()
        tags = family.get("tags", [])
        if not isinstance(tags, list):
            tags = []

        contracts = family.get("contracts", [])
        if not isinstance(contracts, list):
            continue

        for contract in contracts:
            if not isinstance(contract, dict):
                continue
            contract_chain = str(contract.get("chain", "")).strip().lower()
            if contract_chain != chain:
                continue

            address = normalize_address(contract.get("address"))
            if not address or address not in observed_addresses:
                continue

            hit = {
                "family_id": family_id,
                "display_name": display_name,
                "actor_class": actor_class,
                "address": address,
                "label": str(contract.get("label", "")).strip(),
                "canonical": bool(contract.get("canonical", False)),
                "status": str(contract.get("status", "")).strip(),
                "bridge_family": str(contract.get("bridge_family", "")).strip(),
                "source_tier": source_tier,
                "confidence": str(contract.get("confidence", "")).strip(),
                "observed_count": observed_addresses[address],
                "tags": tags,
                "match_source": "address",
            }

            all_hits.append(hit)
            if actor_class == "stablecoin":
                stablecoin_hits.append(hit)
            if actor_class == "account_abstraction_service":
                service_hits.append(hit)

    sort_key = lambda item: (-int(item["observed_count"]), item["display_name"], item["address"])
    all_hits.sort(key=sort_key)
    stablecoin_hits.sort(key=sort_key)
    service_hits.sort(key=sort_key)

    return all_hits[:top_n], stablecoin_hits[:top_n], service_hits[:top_n]


def match_topics(
    chain: str,
    observed_topics: Counter[str],
    topic_registry: dict[str, Any],
    top_n: int,
) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    bridge_hits: list[dict[str, Any]] = []
    protocol_hits: list[dict[str, Any]] = []

    topics = topic_registry.get("topics", [])
    if isinstance(topics, list):
        for topic in topics:
            if not isinstance(topic, dict):
                continue

            topic0 = normalize_topic0(topic.get("topic0"))
            if not topic0 or topic0 not in observed_topics:
                continue

            hit = {
                "topic_family_id": str(topic.get("topic_family_id", "")).strip(),
                "display_name": str(topic.get("display_name", "")).strip(),
                "event_signature": str(topic.get("event_signature", "")).strip(),
                "topic0": topic0,
                "actor_class": str(topic.get("actor_class", "")).strip(),
                "match_mode": str(topic.get("match_mode", "")).strip(),
                "observed_count": observed_topics[topic0],
                "match_source": "topic0",
            }

            actor_class = str(topic.get("actor_class", "")).strip().lower()
            if "bridge" in actor_class:
                bridge_hits.append(hit)
            else:
                protocol_hits.append(hit)

    placeholders = topic_registry.get("chain_specific_placeholders", [])
    if isinstance(placeholders, list):
        for placeholder in placeholders:
            if not isinstance(placeholder, dict):
                continue
            placeholder_chain = str(placeholder.get("chain", "")).strip().lower()
            if placeholder_chain != chain:
                continue

            entries = placeholder.get("entries", [])
            if not isinstance(entries, list):
                continue

            for entry in entries:
                if not isinstance(entry, dict):
                    continue
                topic0 = normalize_topic0(entry.get("topic0"))
                if not topic0 or topic0 not in observed_topics:
                    continue

                hit = {
                    "topic_family_id": str(entry.get("topic_family_id", placeholder.get("topic_family_id", ""))).strip(),
                    "display_name": str(entry.get("display_name", "")).strip(),
                    "event_signature": str(entry.get("event_signature", "")).strip(),
                    "topic0": topic0,
                    "actor_class": str(entry.get("actor_class", "")).strip(),
                    "match_mode": str(entry.get("match_mode", "topic0_only")).strip(),
                    "observed_count": observed_topics[topic0],
                    "match_source": "topic0",
                }

                actor_class = str(entry.get("actor_class", "")).strip().lower()
                if "bridge" in actor_class:
                    bridge_hits.append(hit)
                else:
                    protocol_hits.append(hit)

    sort_key = lambda item: (-int(item["observed_count"]), item["display_name"], item["topic0"])
    bridge_hits.sort(key=sort_key)
    protocol_hits.sort(key=sort_key)

    return bridge_hits[:top_n], protocol_hits[:top_n]


def build_summary(
    observed_addresses: Counter[str],
    observed_topics: Counter[str],
    bridge_contract_hits: list[dict[str, Any]],
    protocol_contract_hits: list[dict[str, Any]],
    stablecoin_contract_hits: list[dict[str, Any]],
    service_contract_hits: list[dict[str, Any]],
    bridge_topic_hits: list[dict[str, Any]],
    protocol_topic_hits: list[dict[str, Any]],
) -> dict[str, Any]:
    return {
        "observed_address_count": len(observed_addresses),
        "observed_topic0_count": len(observed_topics),
        "bridge_contract_match_count": len(bridge_contract_hits),
        "protocol_contract_match_count": len(protocol_contract_hits),
        "stablecoin_contract_match_count": len(stablecoin_contract_hits),
        "service_contract_match_count": len(service_contract_hits),
        "bridge_topic_match_count": len(bridge_topic_hits),
        "protocol_topic_match_count": len(protocol_topic_hits),
        "matched_bridge_families": sorted({item["bridge_family_id"] for item in bridge_contract_hits}),
        "matched_protocol_families": sorted({item["family_id"] for item in protocol_contract_hits}),
        "matched_topic_families": sorted(
            {item["topic_family_id"] for item in bridge_topic_hits} |
            {item["topic_family_id"] for item in protocol_topic_hits}
        ),
    }


def enrich_payload(
    payload: dict[str, Any],
    bridge_registry: dict[str, Any],
    protocol_registry: dict[str, Any],
    topic_registry: dict[str, Any],
    top_n: int,
) -> dict[str, Any]:
    chain = extract_chain(payload)
    if not chain:
        raise ValueError("Unsupported or missing chain field in payload")

    observed_addresses = collect_observed_addresses(payload)
    observed_topics = collect_observed_topics(payload)

    bridge_contract_hits = match_bridge_contracts(chain, observed_addresses, bridge_registry, top_n)
    protocol_contract_hits, stablecoin_contract_hits, service_contract_hits = match_protocol_contracts(
        chain, observed_addresses, protocol_registry, top_n
    )
    bridge_topic_hits, protocol_topic_hits = match_topics(chain, observed_topics, topic_registry, top_n)

    registry_hits = {
        "bridge_contract_hits": bridge_contract_hits,
        "protocol_contract_hits": protocol_contract_hits,
        "stablecoin_contract_hits": stablecoin_contract_hits,
        "service_contract_hits": service_contract_hits,
        "bridge_topic_hits": bridge_topic_hits,
        "protocol_topic_hits": protocol_topic_hits,
        "summary": build_summary(
            observed_addresses=observed_addresses,
            observed_topics=observed_topics,
            bridge_contract_hits=bridge_contract_hits,
            protocol_contract_hits=protocol_contract_hits,
            stablecoin_contract_hits=stablecoin_contract_hits,
            service_contract_hits=service_contract_hits,
            bridge_topic_hits=bridge_topic_hits,
            protocol_topic_hits=protocol_topic_hits,
        ),
    }

    enriched = dict(payload)
    enriched["registry_hits"] = registry_hits
    return enriched


def main() -> None:
    args = parse_args()

    input_dir = Path(args.input_dir)
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    bridge_registry = load_json(Path(args.bridge_registry))
    protocol_registry = load_json(Path(args.protocol_registry))
    topic_registry = load_json(Path(args.topic_registry))

    json_files = sorted(input_dir.glob("*.json"))
    if not json_files:
        raise SystemExit(f"No JSON files found under {input_dir}")

    written_files: list[str] = []
    skipped_files: list[str] = []
    chain_counter: Counter[str] = Counter()

    for path in json_files:
        if path.name == "extract_manifest.json":
            continue

        try:
            payload = load_json(path)
            enriched = enrich_payload(
                payload=payload,
                bridge_registry=bridge_registry,
                protocol_registry=protocol_registry,
                topic_registry=topic_registry,
                top_n=args.top_n,
            )

            chain = extract_chain(enriched)
            if chain:
                chain_counter[chain] += 1

            out_path = out_dir / path.name
            out_path.write_text(json.dumps(enriched, indent=2, default=str), encoding="utf-8")
            written_files.append(out_path.name)
            print(f"Wrote {out_path}")
        except Exception as exc:
            skipped_files.append(path.name)
            print(f"Skipped {path}: {exc}")

    manifest = {
        "script": "build_l2_phase2_registry_hits.py",
        "input_dir": str(input_dir),
        "out_dir": str(out_dir),
        "bridge_registry": str(Path(args.bridge_registry)),
        "protocol_registry": str(Path(args.protocol_registry)),
        "topic_registry": str(Path(args.topic_registry)),
        "written_count": len(written_files),
        "skipped_count": len(skipped_files),
        "written_files": written_files,
        "skipped_files": skipped_files,
        "chains": dict(chain_counter),
    }
    manifest_path = out_dir / "registry_hits_manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, default=str), encoding="utf-8")
    print(f"Wrote {manifest_path}")


if __name__ == "__main__":
    main()