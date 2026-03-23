#!/usr/bin/env python3
import argparse
import datetime as dt
import json
import os
import sys
import time
import requests

# 90-Day Target Window
START_DATE = dt.datetime(2025, 3, 16, tzinfo=dt.timezone.utc)
END_DATE = dt.datetime(2025, 6, 17, tzinfo=dt.timezone.utc)

START_TS = int(START_DATE.timestamp())
END_TS = int(END_DATE.timestamp())

def post_rpc(url, method, params):
    payload = {"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
    try:
        resp = requests.post(url, json=payload, timeout=30)
        resp.raise_for_status()
        data = resp.json()
        if "error" in data:
            print(f"  ! RPC Error: {data['error'].get('message')}", file=sys.stderr)
            return None
        return data.get("result")
    except Exception as e:
        print(f"  ! Request failed: {e}", file=sys.stderr)
        return None

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--rpc", required=True, help="Helius URL")
    parser.add_argument("--addresses", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--limit", type=int, default=50, help="Max txs to fetch PER ADDRESS inside the window")
    args = parser.parse_args()

    os.makedirs(args.out, exist_ok=True)
    with open(args.addresses, 'r') as f:
        addrs = [l.strip() for l in f if l.strip() and not l.startswith("#")]

    for addr in addrs:
        print(f"\nProcessing {addr}...", file=sys.stderr)
        print(f"Paging backward to find transactions between {START_DATE.date()} and {END_DATE.date()}...", file=sys.stderr)

        target_signatures = []
        last_sig = None
        hit_target_window = False
        went_too_far = False

        # Phase 1: Page backward through signatures until we hit the date range
        while not went_too_far and len(target_signatures) < args.limit:
            config = {"limit": 1000, "commitment": "confirmed"} # Fetch max allowed signatures per page
            if last_sig:
                config["before"] = last_sig

            sigs = post_rpc(args.rpc, "getSignaturesForAddress", [addr, config])

            if not sigs:
                print("  ! Reached end of available history for this node.", file=sys.stderr)
                break

            for s in sigs:
                block_time = s.get("blockTime")
                if not block_time:
                    continue

                if block_time > END_TS:
                    # Transaction is newer than June 17, 2025. Keep paging backward.
                    continue
                elif START_TS <= block_time <= END_TS:
                    # Bingo! Inside the 90-day window.
                    if not hit_target_window:
                        print("  > Reached target date window! Collecting signatures...", file=sys.stderr)
                        hit_target_window = True

                    target_signatures.append(s["signature"])
                    if len(target_signatures) >= args.limit:
                        break
                elif block_time < START_TS:
                    # We have gone past March 16, 2025. Stop searching.
                    print("  > Reached data older than target window. Stopping search.", file=sys.stderr)
                    went_too_far = True
                    break

            last_sig = sigs[-1]["signature"]
            time.sleep(0.15) # Protect your 10 RPS free tier limit

        print(f"  Found {len(target_signatures)} signatures in target window. Fetching full details...", file=sys.stderr)

        # Phase 2: Fetch the full details for the matching signatures
        full_txs = []
        for i, sig in enumerate(target_signatures):
            tx = post_rpc(args.rpc, "getTransaction", [
                sig,
                {"encoding": "json", "maxSupportedTransactionVersion": 0, "commitment": "confirmed"}
            ])
            if tx:
                full_txs.append(tx)

            if (i + 1) % 10 == 0:
                print(f"    Fetched {i+1}/{len(target_signatures)} full transactions...", file=sys.stderr)
            time.sleep(0.12)

        # Phase 3: Save results
        out_path = os.path.join(args.out, f"{addr}.json")
        with open(out_path, "w") as f:
            json.dump({"address": addr, "transactions": full_txs}, f, indent=2)
        print(f"  Saved {len(full_txs)} transactions to {out_path}")

if __name__ == "__main__":
    main()