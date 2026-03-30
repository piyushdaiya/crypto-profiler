# scripts/build_optimism_phase1_shortlist.py
#!/usr/bin/env python3

from __future__ import annotations

import argparse
from pathlib import Path

PHASE1_SHORTLIST = [
    "0x8d371bc560246dc632c4e707707d85d2e568a832",
    "0xf70da97812cb96acdf810712aa562db8dfa3dbef",
    "0x623777cc098c6058a46cf7530f45150ff6a8459d",
    "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789",
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Write the Optimism Phase 1 shortlist file")
    parser.add_argument(
        "--out",
        required=True,
        help="Output text file path, e.g. data/candidates/optimism_phase1_shortlist.txt",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(PHASE1_SHORTLIST) + "\n", encoding="utf-8")
    print(f"Wrote {len(PHASE1_SHORTLIST)} addresses to {out_path}")


if __name__ == "__main__":
    main()