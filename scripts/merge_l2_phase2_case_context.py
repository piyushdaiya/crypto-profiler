#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Merge Phase 2 semantic context into curated L2 case files"
    )
    parser.add_argument("--curated-dir", required=True)
    parser.add_argument("--phase2-dir", required=True)
    parser.add_argument("--out-dir", required=True)
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError(f"{path} did not load as an object")
    return data


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.write_text(json.dumps(payload, indent=2, default=str), encoding="utf-8")


def normalize_address(value: Any) -> str:
    if value is None:
        return ""
    s = str(value).strip().lower()
    return s if s.startswith("0x") and len(s) == 42 else ""


def log_summary_populated(log_summary: dict[str, Any]) -> bool:
    if not isinstance(log_summary, dict):
        return False
    if int(log_summary.get("unique_emitters", 0)) > 0:
        return True
    if int(log_summary.get("unique_topic0s", 0)) > 0:
        return True
    if log_summary.get("top_emitters"):
        return True
    if log_summary.get("top_topic0s"):
        return True
    return False


def main() -> None:
    args = parse_args()

    curated_dir = Path(args.curated_dir)
    phase2_dir = Path(args.phase2_dir)
    out_dir = Path(args.out_dir)

    out_dir.mkdir(parents=True, exist_ok=True)

    phase2_index: dict[str, dict[str, Any]] = {}
    for path in sorted(phase2_dir.glob("*.json")):
        if path.name.startswith("._") or path.name.endswith("manifest.json"):
            continue
        payload = load_json(path)
        address = normalize_address(payload.get("address"))
        if address:
            phase2_index[address] = payload

    written: list[str] = []
    skipped: list[str] = []

    for path in sorted(curated_dir.glob("*.json")):
        if path.name.startswith("._") or path.name.endswith("manifest.json"):
            continue

        curated = load_json(path)
        address = normalize_address(curated.get("address"))
        if not address or address not in phase2_index:
            skipped.append(path.name)
            continue

        phase2 = phase2_index[address]
        receipt_summary = phase2.get("receipt_summary", {})
        log_summary = phase2.get("log_summary", {})
        registry_hits = phase2.get("registry_hits", {})

        merged = dict(curated)
        merged["receipt_summary"] = receipt_summary
        merged["log_summary"] = log_summary
        merged["registry_hits"] = registry_hits

        dataset_notes = dict(merged.get("dataset_notes", {}))
        dataset_notes["phase2_semantics_status"] = (
            "receipt_and_log_context"
            if log_summary_populated(log_summary)
            else "receipt_and_registry_context_only"
        )
        dataset_notes["logs_deferred_due_scan_cost"] = not log_summary_populated(log_summary)
        merged["dataset_notes"] = dataset_notes

        curation_notes = dict(merged.get("curation_notes", {}))
        curation_notes["phase2_refresh_note"] = (
            "Refreshed from extracted-phase2. Receipt summaries and registry hits are populated. "
            "Log summaries remain deferred for this pass where scan cost was not acceptable."
        )
        merged["curation_notes"] = curation_notes

        out_path = out_dir / path.name
        write_json(out_path, merged)
        written.append(out_path.name)
        print(f"Wrote {out_path}")

    manifest = {
        "script": "merge_l2_phase2_case_context.py",
        "curated_dir": str(curated_dir),
        "phase2_dir": str(phase2_dir),
        "out_dir": str(out_dir),
        "written_count": len(written),
        "skipped_count": len(skipped),
        "written_files": written,
        "skipped_files": skipped,
    }
    write_json(out_dir / "phase2_case_merge_manifest.json", manifest)
    print(f"Wrote {out_dir / 'phase2_case_merge_manifest.json'}")


if __name__ == "__main__":
    main()