#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

# Replace these after candidate review.
PHASE1_SHORTLIST = [
    "0xreplace_1",
    "0xreplace_2",
    "0xreplace_3",
    "0xreplace_4",
]

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Write the Polygon Phase 1 shortlist file")
    parser.add_argument("--out", required=True)
    return parser.parse_args()

def main() -> None:
    args = parse_args()
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(PHASE1_SHORTLIST) + "\n", encoding="utf-8")
    print(f"Wrote {len(PHASE1_SHORTLIST)} addresses to {out_path}")

if __name__ == "__main__":
    main()