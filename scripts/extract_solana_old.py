#!/usr/bin/env python3
"""
Address-scoped Solana extractor skeleton for Crypto Profiler.

What this script is meant to do:
- fetch recent signature history for one or more Solana addresses
- hydrate parsed transactions for those signatures
- derive lightweight summary features for KYW-style analysis
- optionally enrich with SPL token account inventory
- optionally use Helius enhanced transaction history for cleaner parsing

This is intentionally an MVP skeleton, not a finished production ingestor.
"""

from __future__ import annotations

import argparse
import datetime as dt
import gzip
import json
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Dict, List, Optional

import requests


SOLANA_TOKEN_PROGRAM = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
KNOWN_STABLECOIN_MINTS = {
    # Mainnet USDC mint
    "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": "USDC",
}


def utc_now_iso() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z")


def ensure_dir(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def write_json(path: Path, payload: Any) -> None:
    with path.open("w", encoding="utf-8") as fh:
        json.dump(payload, fh, indent=2)
        fh.write("\n")


def normalize_pubkey(value: Optional[str]) -> str:
    if value is None:
        return ""
    return str(value).strip()


def blocktime_to_iso(block_time: Optional[int]) -> Optional[str]:
    if block_time is None:
        return None
    return dt.datetime.fromtimestamp(block_time, tz=dt.timezone.utc).isoformat().replace("+00:00", "Z")


class SolanaRPC:
    def __init__(self, rpc_url: str, timeout_seconds: int = 30) -> None:
        self.rpc_url = rpc_url
        self.timeout_seconds = timeout_seconds
        self.session = requests.Session()

    def call(self, method: str, params: List[Any]) -> Any:
        payload = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params,
        }
        resp = self.session.post(self.rpc_url, json=payload, timeout=self.timeout_seconds)
        resp.raise_for_status()
        data = resp.json()
        if "error" in data:
            raise RuntimeError(f"{method} RPC error: {data['error']}")
        return data.get("result")

    def get_signatures_for_address(
        self,
        address: str,
        before: Optional[str] = None,
        limit: int = 1000,
    ) -> List[Dict[str, Any]]:
        config: Dict[str, Any] = {
            "limit": limit,
            "commitment": "finalized",
        }
        if before:
            config["before"] = before

        return self.call("getSignaturesForAddress", [address, config]) or []

    def get_transaction(self, signature: str) -> Optional[Dict[str, Any]]:
        return self.call(
            "getTransaction",
            [
                signature,
                {
                    "encoding": "jsonParsed",
                    "maxSupportedTransactionVersion": 0,
                    "commitment": "finalized",
                },
            ],
        )

    def get_token_accounts_by_owner(self, owner: str) -> Dict[str, Any]:
        return self.call(
            "getTokenAccountsByOwner",
            [
                owner,
                {"programId": SOLANA_TOKEN_PROGRAM},
                {"encoding": "jsonParsed", "commitment": "finalized"},
            ],
        )


@dataclass
class CounterpartySummary:
    address: str
    interactions: int = 0
    inbound_count: int = 0
    outbound_count: int = 0


@dataclass
class ProgramSummary:
    program_id: str
    interactions: int = 0


@dataclass
class MintSummary:
    mint: str
    interactions: int = 0
    stablecoin: bool = False


@dataclass
class SolanaAddressSummary:
    address: str
    generated_at: str = field(default_factory=utc_now_iso)
    first_seen: Optional[str] = None
    last_seen: Optional[str] = None
    signature_count: int = 0
    failed_signature_count: int = 0
    success_signature_count: int = 0
    unique_counterparties: set = field(default_factory=set)
    native_inbound_count: int = 0
    native_outbound_count: int = 0
    spl_transfer_count: int = 0
    stablecoin_transfer_count: int = 0
    token_accounts_count: int = 0
    active_token_mints_count: int = 0
    counterparties: Dict[str, CounterpartySummary] = field(default_factory=dict)
    programs: Dict[str, ProgramSummary] = field(default_factory=dict)
    mints: Dict[str, MintSummary] = field(default_factory=dict)
    sample_transactions: List[Dict[str, Any]] = field(default_factory=list)

    def observe_signature(self, block_time_iso: Optional[str], success: bool) -> None:
        self.signature_count += 1
        if success:
            self.success_signature_count += 1
        else:
            self.failed_signature_count += 1

        if block_time_iso:
            if self.first_seen is None or block_time_iso < self.first_seen:
                self.first_seen = block_time_iso
            if self.last_seen is None or block_time_iso > self.last_seen:
                self.last_seen = block_time_iso

    def add_counterparty(self, address: str, direction: str) -> None:
        address = normalize_pubkey(address)
        if not address or address == self.address:
            return

        self.unique_counterparties.add(address)
        entry = self.counterparties.get(address)
        if entry is None:
            entry = CounterpartySummary(address=address)
            self.counterparties[address] = entry
        entry.interactions += 1
        if direction == "inbound":
            entry.inbound_count += 1
        elif direction == "outbound":
            entry.outbound_count += 1

    def add_program(self, program_id: str) -> None:
        program_id = normalize_pubkey(program_id)
        if not program_id:
            return
        entry = self.programs.get(program_id)
        if entry is None:
            entry = ProgramSummary(program_id=program_id)
            self.programs[program_id] = entry
        entry.interactions += 1

    def add_mint(self, mint: str) -> None:
        mint = normalize_pubkey(mint)
        if not mint:
            return
        entry = self.mints.get(mint)
        if entry is None:
            entry = MintSummary(
                mint=mint,
                stablecoin=mint in KNOWN_STABLECOIN_MINTS,
            )
            self.mints[mint] = entry
        entry.interactions += 1
        if entry.stablecoin:
            self.stablecoin_transfer_count += 1

    def to_summary_json(self, source_transaction_count: int) -> Dict[str, Any]:
        top_counterparties = sorted(
            self.counterparties.values(),
            key=lambda x: (-x.interactions, x.address),
        )[:20]
        top_programs = sorted(
            self.programs.values(),
            key=lambda x: (-x.interactions, x.program_id),
        )[:20]
        top_mints = sorted(
            self.mints.values(),
            key=lambda x: (-x.interactions, x.mint),
        )[:20]

        return {
            "address": self.address,
            "chain": "SOLANA",
            "generated_at": self.generated_at,
            "summary": {
                "first_seen": self.first_seen,
                "last_seen": self.last_seen,
                "signature_count": self.signature_count,
                "failed_signature_count": self.failed_signature_count,
                "success_signature_count": self.success_signature_count,
                "unique_counterparties": len(self.unique_counterparties),
                "native_inbound_count": self.native_inbound_count,
                "native_outbound_count": self.native_outbound_count,
                "spl_transfer_count": self.spl_transfer_count,
                "stablecoin_transfer_count": self.stablecoin_transfer_count,
                "token_accounts_count": self.token_accounts_count,
                "active_token_mints_count": self.active_token_mints_count,
            },
            "top_counterparties": [
                {
                    "address": cp.address,
                    "interactions": cp.interactions,
                    "inbound_count": cp.inbound_count,
                    "outbound_count": cp.outbound_count,
                }
                for cp in top_counterparties
            ],
            "top_programs": [
                {
                    "program_id": p.program_id,
                    "interactions": p.interactions,
                }
                for p in top_programs
            ],
            "top_mints": [
                {
                    "mint": m.mint,
                    "interactions": m.interactions,
                    "stablecoin": m.stablecoin,
                }
                for m in top_mints
            ],
            "sample_transactions": self.sample_transactions,
            "source_transaction_count": source_transaction_count,
        }


def extract_program_ids(tx_result: Dict[str, Any]) -> List[str]:
    out: List[str] = []
    if not tx_result:
        return out

    transaction = tx_result.get("transaction") or {}
    message = transaction.get("message") or {}
    instructions = message.get("instructions") or []

    for ix in instructions:
        if isinstance(ix, dict):
            program_id = ix.get("programId")
            if program_id:
                out.append(program_id)

    meta = tx_result.get("meta") or {}
    inner = meta.get("innerInstructions") or []
    for inner_group in inner:
        for ix in inner_group.get("instructions", []):
            if isinstance(ix, dict):
                program_id = ix.get("programId")
                if program_id:
                    out.append(program_id)

    return out


def extract_account_keys(tx_result: Dict[str, Any]) -> List[str]:
    transaction = tx_result.get("transaction") or {}
    message = transaction.get("message") or {}
    account_keys = message.get("accountKeys") or []

    out: List[str] = []
    for item in account_keys:
        if isinstance(item, str):
            out.append(item)
        elif isinstance(item, dict):
            pubkey = item.get("pubkey")
            if pubkey:
                out.append(pubkey)
    return out


def extract_token_mints(tx_result: Dict[str, Any]) -> List[str]:
    meta = tx_result.get("meta") or {}
    out: List[str] = []

    for side in ("preTokenBalances", "postTokenBalances"):
        balances = meta.get(side) or []
        for entry in balances:
            mint = entry.get("mint")
            if mint:
                out.append(mint)

    return out


def summarize_transaction(address: str, signature_row: Dict[str, Any], tx_result: Optional[Dict[str, Any]]) -> Dict[str, Any]:
    block_time_iso = blocktime_to_iso(signature_row.get("blockTime"))
    success = signature_row.get("err") is None

    account_keys = extract_account_keys(tx_result or {})
    program_ids = extract_program_ids(tx_result or {})
    token_mints = extract_token_mints(tx_result or {})

    return {
        "signature": signature_row.get("signature"),
        "slot": signature_row.get("slot"),
        "block_time": block_time_iso,
        "success": success,
        "error": signature_row.get("err"),
        "account_keys": account_keys,
        "program_ids": program_ids,
        "token_mints": token_mints,
    }


def fetch_signatures(
    rpc: SolanaRPC,
    address: str,
    days: int,
    max_signatures: int,
    pause_ms: int,
) -> List[Dict[str, Any]]:
    cutoff = dt.datetime.now(dt.timezone.utc) - dt.timedelta(days=days)
    before: Optional[str] = None
    collected: List[Dict[str, Any]] = []

    while len(collected) < max_signatures:
        batch = rpc.get_signatures_for_address(address, before=before, limit=min(1000, max_signatures - len(collected)))
        if not batch:
            break

        stop = False
        for row in batch:
            block_time = row.get("blockTime")
            if block_time is not None:
                ts = dt.datetime.fromtimestamp(block_time, tz=dt.timezone.utc)
                if ts < cutoff:
                    stop = True
                    break
            collected.append(row)

        if stop:
            break

        before = batch[-1].get("signature")
        if not before:
            break

        if pause_ms > 0:
            time.sleep(pause_ms / 1000.0)

    return collected


def enrich_token_accounts(rpc: SolanaRPC, address: str, summary: SolanaAddressSummary) -> None:
    result = rpc.get_token_accounts_by_owner(address)
    value = result.get("value") if isinstance(result, dict) else None
    if not isinstance(value, list):
        return

    summary.token_accounts_count = len(value)

    active_mints = set()
    for entry in value:
        account = entry.get("account", {})
        data = account.get("data", {})
        parsed = data.get("parsed", {})
        info = parsed.get("info", {})
        mint = info.get("mint")
        token_amount = info.get("tokenAmount", {})
        amount_raw = token_amount.get("amount", "0")
        if mint:
            active_mints.add(mint)
            if amount_raw not in ("0", "", None):
                summary.add_mint(mint)

    summary.active_token_mints_count = len(active_mints)


def main() -> int:
    parser = argparse.ArgumentParser(description="Extract address-scoped Solana datasets for Crypto Profiler")
    parser.add_argument("--address", required=True, help="Solana address to profile")
    parser.add_argument("--rpc-url", required=True, help="Solana RPC URL")
    parser.add_argument("--out", required=True, help="Output directory, e.g. ./data/cases/extracted-solana")
    parser.add_argument("--days", type=int, default=90, help="How many days of history to fetch")
    parser.add_argument("--max-signatures", type=int, default=5000, help="Maximum signatures to fetch")
    parser.add_argument("--sample", type=int, default=50, help="Maximum transactions to keep in summary sample")
    parser.add_argument("--pause-ms", type=int, default=100, help="Sleep between paginated RPC calls")
    parser.add_argument(
        "--use-helius-enhanced",
        action="store_true",
        help="Reserved flag for future Helius enhanced transaction history integration",
    )
    args = parser.parse_args()

    address = normalize_pubkey(args.address)
    out_dir = Path(args.out).expanduser().resolve()
    ensure_dir(out_dir)

    rpc = SolanaRPC(args.rpc_url)
    summary = SolanaAddressSummary(address=address)

    print(f"Fetching signatures for {address}...", file=sys.stderr)
    signatures = fetch_signatures(
        rpc=rpc,
        address=address,
        days=args.days,
        max_signatures=args.max_signatures,
        pause_ms=args.pause_ms,
    )

    signatures_path = out_dir / f"{address}.signatures.json"
    write_json(signatures_path, signatures)

    raw_tx_path = out_dir / f"{address}.transactions.ndjson.gz"
    sample_transactions: List[Dict[str, Any]] = []

    print(f"Hydrating {len(signatures)} transactions...", file=sys.stderr)
    with gzip.open(raw_tx_path, "wt", encoding="utf-8") as fh:
        for idx, sig_row in enumerate(signatures, start=1):
            signature = sig_row.get("signature")
            tx_result = None
            if signature:
                try:
                    tx_result = rpc.get_transaction(signature)
                except Exception as exc:  # noqa: BLE001
                    tx_result = {"_fetch_error": str(exc)}

            block_time_iso = blocktime_to_iso(sig_row.get("blockTime"))
            success = sig_row.get("err") is None
            summary.observe_signature(block_time_iso, success)

            simplified = summarize_transaction(address, sig_row, tx_result if isinstance(tx_result, dict) else None)

            for program_id in simplified.get("program_ids", []):
                summary.add_program(program_id)

            for acct in simplified.get("account_keys", []):
                acct = normalize_pubkey(acct)
                if acct and acct != address:
                    summary.add_counterparty(acct, "outbound")

            for mint in simplified.get("token_mints", []):
                summary.add_mint(mint)
                summary.spl_transfer_count += 1

            if len(sample_transactions) < args.sample:
                sample_transactions.append(simplified)

            fh.write(json.dumps(simplified, separators=(",", ":")) + "\n")

            if idx % 250 == 0:
                print(f"hydrated {idx} transactions...", file=sys.stderr)

    summary.sample_transactions = sample_transactions

    print("Fetching token accounts...", file=sys.stderr)
    try:
        enrich_token_accounts(rpc, address, summary)
    except Exception as exc:  # noqa: BLE001
        print(f"warning: token account enrichment failed: {exc}", file=sys.stderr)

    summary_path = out_dir / f"{address}.json"
    write_json(summary_path, summary.to_summary_json(source_transaction_count=len(signatures)))

    print(f"wrote {signatures_path}")
    print(f"wrote {raw_tx_path}")
    print(f"wrote {summary_path}")

    if args.use_helius_enhanced:
        print("note: --use-helius-enhanced is a placeholder in this MVP skeleton", file=sys.stderr)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
