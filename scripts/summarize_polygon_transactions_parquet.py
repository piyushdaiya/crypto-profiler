
# scripts/summarize_polygon_transactions_parquet.py
#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
from collections import Counter
from pathlib import Path
from typing import Any

import pandas as pd


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Summarize exported Polygon transaction parquet files")
    parser.add_argument("--input-dir", required=True)
    parser.add_argument("--out-dir", required=True)
    parser.add_argument("--top-n", type=int, default=25)
    parser.add_argument("--sample-limit", type=int, default=100)
    return parser.parse_args()


def top_n_counter(counter: Counter[str], n: int) -> list[dict[str, Any]]:
    return [{"key": key, "count": count} for key, count in counter.most_common(n)]


def first_nonempty(series: pd.Series) -> str:
    values = [str(v) for v in series.dropna().tolist() if str(v).strip()]
    return values[0] if values else ""


def build_extract(df: pd.DataFrame, address: str, top_n: int, sample_limit: int) -> dict[str, Any]:
    df = df.sort_values("block_timestamp", ascending=False).copy()

    counterparties = Counter(df["counterparty"].dropna().astype(str).tolist())
    to_addresses = Counter(df["to_address"].dropna().astype(str).tolist())
    from_addresses = Counter(df["from_address"].dropna().astype(str).tolist())

    inbound_count = int((df["direction"] == "inbound").sum())
    outbound_count = int((df["direction"] == "outbound").sum())

    dominant_direction = "mixed"
    if outbound_count > inbound_count:
        dominant_direction = "outbound"
    elif inbound_count > outbound_count:
        dominant_direction = "inbound"

    tx_count = len(df)
    dominant_counterparty_share = round((counterparties.most_common(1)[0][1] / tx_count) * 100, 2) if counterparties and tx_count else 0.0
    dominant_contract_share = round((to_addresses.most_common(1)[0][1] / tx_count) * 100, 2) if to_addresses and tx_count else 0.0

    sample_df = df.head(sample_limit).copy()
    for col in sample_df.columns:
        if str(sample_df[col].dtype).startswith("datetime64"):
            sample_df[col] = sample_df[col].astype(str)

    return {
        "chain": "POLYGON",
        "address": address,
        "window_start": first_nonempty(df.sort_values("block_timestamp")["block_timestamp"]),
        "window_end": first_nonempty(df.sort_values("block_timestamp", ascending=False)["block_timestamp"]),
        "summary": {
            "first_seen": str(df["block_timestamp"].min()) if tx_count else "",
            "last_seen": str(df["block_timestamp"].max()) if tx_count else "",
            "tx_count": tx_count,
            "inbound_count": inbound_count,
            "outbound_count": outbound_count,
            "unique_counterparties": len(counterparties),
            "unique_to_addresses": len(to_addresses),
            "unique_from_addresses": len(from_addresses),
            "dominant_direction": dominant_direction,
            "dominant_counterparty_share": dominant_counterparty_share,
            "dominant_contract_share": dominant_contract_share,
        },
        "top_counterparties": top_n_counter(counterparties, top_n),
        "top_to_addresses": top_n_counter(to_addresses, top_n),
        "top_from_addresses": top_n_counter(from_addresses, top_n),
        "sample_transfers": sample_df.to_dict(orient="records"),
        "dataset_notes": {
            "source": "BigQuery export from Google Cloud Blockchain Analytics",
            "extraction_mode": "transactions_only_phase1_minimal",
        },
    }


def main() -> None:
    args = parse_args()
    input_dir = Path(args.input_dir)
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    parquet_files = sorted(
        path for path in input_dir.rglob("*.parquet")
        if not path.name.startswith("._")
    )
    if not parquet_files:
        raise SystemExit(f"No parquet files found under {input_dir}")

    frames = [pd.read_parquet(path) for path in parquet_files]
    df = pd.concat(frames, ignore_index=True)

    required_columns = {
        "profiled_address",
        "block_timestamp",
        "from_address",
        "to_address",
        "direction",
        "counterparty",
    }
    missing = required_columns - set(df.columns)
    if missing:
        raise SystemExit(f"Missing expected columns: {sorted(missing)}")

    for address, group in df.groupby("profiled_address"):
        payload = build_extract(group, address, args.top_n, args.sample_limit)
        out_path = out_dir / f"{address}.json"
        out_path.write_text(json.dumps(payload, indent=2, default=str), encoding="utf-8")
        print(f"Wrote {out_path}")

    manifest = {
        "script": "summarize_polygon_transactions_parquet.py",
        "input_dir": str(input_dir),
        "out_dir": str(out_dir),
        "address_count": int(df["profiled_address"].nunique()),
        "row_count": int(len(df)),
    }
    (out_dir / "extract_manifest.json").write_text(json.dumps(manifest, indent=2, default=str), encoding="utf-8")
    print(f"Wrote {(out_dir / 'extract_manifest.json')}")


if __name__ == "__main__":
    main()