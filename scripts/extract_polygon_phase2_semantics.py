#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from _l2_phase2_common import run_phase2_merge


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Merge Polygon Phase 2 semantic summaries into extracted JSON artifacts")
    parser.add_argument("--input-dir", required=True, help="Registry-enriched extract directory")
    parser.add_argument("--receipt-summary-file", required=False, help="Optional JSON file with per-address receipt summaries")
    parser.add_argument("--log-summary-file", required=False, help="Optional JSON file with per-address log summaries")
    parser.add_argument("--out-dir", required=True, help="Output directory for final Polygon Phase 2 extracts")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    run_phase2_merge(
        expected_chain="POLYGON",
        input_dir=Path(args.input_dir),
        receipt_summary_file=Path(args.receipt_summary_file) if args.receipt_summary_file else None,
        log_summary_file=Path(args.log_summary_file) if args.log_summary_file else None,
        out_dir=Path(args.out_dir),
        script_name="extract_polygon_phase2_semantics.py",
    )


if __name__ == "__main__":
    main()