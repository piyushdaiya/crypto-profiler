#!/usr/bin/env python3

import argparse
import datetime as dt
import gzip
import json
import os
import sys
import time
from collections import defaultdict
from dataclasses import dataclass, field
from typing import Dict, List, Optional
import random
import requests


# -----------------------------
# Helpers
# -----------------------------

def now_iso():
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


import random

def post_rpc(url, method, params, retries=5):
    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params,
    }

    for attempt in range(retries):
        try:
            resp = requests.post(url, json=payload, timeout=20)

            if resp.status_code == 429:
                sleep_time = (2 ** attempt) + random.uniform(0, 1)
                print(f"rate limited, retrying in {sleep_time:.2f}s...", file=sys.stderr)
                time.sleep(sleep_time)
                continue

            resp.raise_for_status()
            data = resp.json()

            if "error" in data:
                raise RuntimeError(data["error"])

            return data["result"]

        except requests.exceptions.RequestException as e:
            if attempt == retries - 1:
                raise
            time.sleep(2 ** attempt)

    raise RuntimeError("RPC failed after retries")


def load_addresses(path: str) -> List[str]:
    out = []
    seen = set()

    with open(path, "r") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if line not in seen:
                seen.add(line)
                out.append(line)

    return out


# -----------------------------
# Data classes
# -----------------------------

@dataclass
class Counterparty:
    address: str
    interactions: int = 0


@dataclass
class ProgramSummary:
    program_id: str
    interactions: int = 0


@dataclass
class SolanaDatasetBuilder:
    address: str
    out_dir: str
    sample_limit: int

    generated_at: str = field(default_factory=now_iso)

    first_seen: Optional[str] = None
    last_seen: Optional[str] = None

    total_txs: int = 0
    failed_txs: int = 0

    counterparties: Dict[str, Counterparty] = field(default_factory=dict)
    programs: Dict[str, ProgramSummary] = field(default_factory=dict)

    sample_txs: List[dict] = field(default_factory=list)

    def __post_init__(self):
        os.makedirs(self.out_dir, exist_ok=True)
        self.raw_path = os.path.join(self.out_dir, f"{self.address}.txs.ndjson.gz")
        self._fh = gzip.open(self.raw_path, "wt")

    def close(self):
        self._fh.close()

    def add_tx(self, tx: dict):
        self.total_txs += 1

        block_time = tx.get("blockTime")
        if block_time:
            iso = dt.datetime.fromtimestamp(block_time, tz=dt.timezone.utc).isoformat().replace("+00:00", "Z")
            if not self.first_seen or iso < self.first_seen:
                self.first_seen = iso
            if not self.last_seen or iso > self.last_seen:
                self.last_seen = iso

        meta = tx.get("meta", {})
        if meta.get("err") is not None:
            self.failed_txs += 1

        # Accounts
        message = tx.get("transaction", {}).get("message", {})
        raw_keys = message.get("accountKeys", [])
        if not raw_keys:
            return

        account_keys = []
        for k in raw_keys:
            if isinstance(k, dict):
                account_keys.append(k.get("pubkey"))
            else:
                account_keys.append(k)

        for acc in account_keys:
            if acc != self.address:
                cp = self.counterparties.get(acc)
                if cp is None:
                    cp = Counterparty(address=acc)
                    self.counterparties[acc] = cp
                cp.interactions += 1

        # Programs
        instructions = message.get("instructions", [])
        for ix in instructions:
            program_id = None
            if "programIdIndex" in ix:
                idx = ix["programIdIndex"]
                if idx < len(account_keys):
                    program_id = account_keys[idx]
            if not program_id:
                continue
            ps = self.programs.get(program_id)
            if ps is None:
                ps = ProgramSummary(program_id=program_id)
                self.programs[program_id] = ps
            ps.interactions += 1

        if len(self.sample_txs) < self.sample_limit:
            self.sample_txs.append(tx)

        self._fh.write(json.dumps(tx) + "\n")

    def metadata(self):
        top_counterparties = sorted(
            self.counterparties.values(),
            key=lambda x: -x.interactions
        )[:20]

        top_programs = sorted(
            self.programs.values(),
            key=lambda x: -x.interactions
        )[:20]

        return {
            "address": self.address,
            "chain": "SOLANA",
            "generated_at": self.generated_at,
            "summary": {
                "first_seen": self.first_seen,
                "last_seen": self.last_seen,
                "total_txs": self.total_txs,
                "failed_txs": self.failed_txs,
                "unique_counterparties": len(self.counterparties),
                "unique_programs": len(self.programs),
            },
            "top_counterparties": [
                {"address": c.address, "interactions": c.interactions}
                for c in top_counterparties
            ],
            "top_programs": [
                {"program_id": p.program_id, "interactions": p.interactions}
                for p in top_programs
            ],
            "sample_txs": self.sample_txs,
            "raw_file": os.path.basename(self.raw_path),
        }


# -----------------------------
# Core extraction
# -----------------------------

def fetch_signatures(rpc_url, address, limit=200):
    return post_rpc(rpc_url, "getSignaturesForAddress", [
        address,
        {"limit": limit}
    ])


def fetch_transaction(rpc_url, signature):
    return post_rpc(rpc_url, "getTransaction", [
        signature,
        {
            "encoding": "json",
            "maxSupportedTransactionVersion": 0
        }
    ])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--rpc", required=True)
    parser.add_argument("--addresses", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--sample", type=int, default=100)

    args = parser.parse_args()

    addresses = load_addresses(args.addresses)
    if not addresses:
        print("no addresses", file=sys.stderr)
        sys.exit(1)

    builders = {
        a: SolanaDatasetBuilder(a, args.out, args.sample)
        for a in addresses
    }

    try:
        for addr in addresses:
            print(f"processing {addr}...", file=sys.stderr)

            sigs = fetch_signatures(args.rpc, addr, limit=200)

            for i, s in enumerate(sigs):
                sig = s["signature"]

                try:
                   tx = fetch_transaction(args.rpc, sig)

                   if tx is None:
                       print("tx = None (likely versioned / RPC limit)", file=sys.stderr)
                       continue

                   if "transaction" not in tx:
                       continue

                   builders[addr].add_tx(tx)
                except Exception as e:
                    print(f"tx error {sig}: {e}", file=sys.stderr)

                if i % 50 == 0:
                    print(f"{addr}: processed {i} txs", file=sys.stderr)
                    time.sleep(0.2)  # light rate limiting
                time.sleep(0.05)


    finally:
        for b in builders.values():
            b.close()

    for addr, b in builders.items():
        path = os.path.join(args.out, f"{addr}.json")
        with open(path, "w") as f:
            json.dump(b.metadata(), f, indent=2)
        print(f"wrote {path} (txs={b.total_txs})")


if __name__ == "__main__":
    main()
