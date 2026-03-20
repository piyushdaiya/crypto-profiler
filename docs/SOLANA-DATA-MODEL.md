# Crypto Profiler Solana Data Model

## Purpose

This document defines the Solana MVP data model for Crypto Profiler.

It explains:

- which Solana datasets are in scope for the MVP
- how address history, parsed transactions, token accounts, and token balances fit together
- what Solana behaviors the MVP can support now
- what remains planned for later phases

This document is intentionally MVP-focused and aligned to the current architecture style used for Bitcoin, Ethereum, and ERC-20.

---

## Scope

The Solana MVP data model is built from four practical layers:

1. **Address transaction history**
2. **Parsed transaction details**
3. **SPL token account inventory**
4. **Optional enhanced transaction parsing**

These layers serve different purposes.

- address history gives the event timeline
- parsed transactions give instruction-level and balance-change context
- token account inventory gives SPL token footprint
- enhanced parsing improves readability for complex Solana activity

The MVP remains **address-scoped**, not full-chain.

---

## Canonical MVP Approach

The Solana MVP should use an **address-first ingestion model**.

### Why address-first

Crypto Profiler is a KYW and exposure-analysis platform, not a chain-wide Solana warehouse.

The fastest path to MVP value is:

1. start with a Solana wallet or program address
2. fetch recent signatures involving that address
3. fetch parsed transaction details for those signatures
4. summarize counterparties, programs, SPL token activity, failures, and recency
5. produce curated case artifacts and explainable signals

This is the right MVP tradeoff because it supports:

- demos
- case studies
- explainable rules
- lightweight infrastructure

---

## Canonical MVP Window

Unlike the current Bitcoin/Ethereum path, Solana MVP ingestion should be defined by:

- **address-scoped history depth**
- not a fixed full-chain date window

### Recommended starting window

For the first Solana MVP slice, use:

- up to **90 days** of transaction history per address
- or the most recent **2,000–10,000 signatures**, whichever is reached first

This gives enough data to support:

- velocity analysis
- repeated interaction analysis
- concentration patterns
- token footprint summaries
- curated case generation

---

## Current Recommended Data Sources

## 1. Address Signature History

### Source

- `getSignaturesForAddress`

### What it provides

- transaction signatures involving the address
- slot
- block time
- success / error summary
- pagination cursoring via `before` / `until`

### Why it matters

This is the entrypoint for address-scoped Solana history collection.

### MVP use

- establish activity timeline
- identify the transactions to hydrate next
- count activity volume
- track first seen / last seen
- count failed transactions

---

## 2. Parsed Transaction Details

### Source

- `getTransaction`
- preferred encoding: `jsonParsed`

### What it provides

- parsed message and instruction structure
- account keys
- fee
- block time
- inner instructions
- token balance changes
- lamport balance changes
- status / errors

### Why it matters

This is the main semantic layer for Solana KYW reasoning.

It makes it possible to summarize:

- counterparties
- token movement
- program interaction
- failed instructions
- balance-change patterns

---

## 3. SPL Token Accounts

### Source

- `getTokenAccountsByOwner`

### What it provides

- token accounts owned by an address
- token mint mapping
- current balance-bearing token account footprint

### Why it matters

It lets the MVP distinguish:

- native SOL-only wallets
- token-heavy wallets
- stablecoin-active wallets
- token inventory breadth

### Optional supporting calls

- `getTokenAccountBalance`
- `getAccountInfo`

These are useful when token account context needs to be refreshed or enriched.

---

## 4. Optional Enhanced Solana Parsing

### Source

- Helius Enhanced Transactions API

### What it provides

- structured transaction history by address
- human-readable interpretation of swaps, transfers, NFT activity, and DeFi actions
- easier parsing for complex Solana workflows

### Why it matters

Raw Solana transactions can be noisy and instruction-heavy.

For the MVP, Helius can reduce implementation complexity when:

- building case studies
- summarizing token/program activity
- interpreting DeFi-style transaction flows

---

## Core Solana Entities

## A. Address

A Solana base58 public key that may represent:

- user wallet
- exchange deposit address
- protocol-controlled address
- treasury
- program-owned account
- token account
- scam or exploit-linked destination

Crypto Profiler should treat the user-supplied wallet address as the primary profiled object.

---

## B. Signature Record

A signature-history row should contain:

- `signature`
- `slot`
- `block_time`
- `err`
- `memo`
- `confirmation_status`

This is the lightest historical event frame.

---

## C. Parsed Transaction Record

A parsed transaction record should contain:

- `signature`
- `slot`
- `block_time`
- `fee_lamports`
- `success`
- `error`
- `signers`
- `account_keys`
- `program_ids`
- `parsed_instructions`
- `inner_instructions`
- `pre_balances`
- `post_balances`
- `pre_token_balances`
- `post_token_balances`

This is the main analysis object for Solana.

---

## D. Token Account Record

A token account inventory row should contain:

- `owner`
- `token_account`
- `mint`
- `amount_raw`
- `amount_ui`
- `decimals`
- `token_program`

This gives wallet-level SPL footprint.

---

## E. Solana Summary Record

For analyzer use, the MVP should reduce raw Solana history into an address summary with:

- `address`
- `chain`
- `first_seen`
- `last_seen`
- `signature_count`
- `failed_signature_count`
- `success_signature_count`
- `unique_counterparties`
- `top_counterparties`
- `top_programs`
- `native_inbound_count`
- `native_outbound_count`
- `spl_transfer_count`
- `stablecoin_transfer_count`
- `token_accounts_count`
- `active_token_mints_count`
- `sample_transactions`

---

## Why Solana Needs Its Own Model

Solana is not just “another EVM.”

Important differences:

- transactions reference many accounts, not only simple sender/recipient pairs
- programs drive behavior more explicitly
- native SOL and SPL token flows coexist in one transaction model
- token accounts are distinct on-chain objects
- instruction parsing matters much more for meaningful semantics

Because of this, Solana modeling should be:

- account-aware
- program-aware
- token-account-aware
- instruction-aware

---

## Current Solana MVP Behaviors the Model Can Support

With the schema above, Crypto Profiler can support:

### Directly supportable

- Solana address validation
- first seen / last seen activity
- failed transaction observation
- repeated interaction with known flagged Solana counterparties
- concentration to a known service or program
- token-heavy wallet identification
- stablecoin-heavy wallet identification
- broad counterparty surface summaries
- curated dataset generation for Solana cases

### Partially supportable

- pass-through behavior
- newly active wallet with immediate flow
- service concentration by program or destination
- repeated flagged destination activity

### Not yet fully supportable

- deep graph traversal
- cluster/entity resolution
- full DeFi semantic decoding without enhanced parsing
- chain-hopping / bridge-aware reasoning
- full wash / circular flow interpretation

---

## Recommended Extractor Plan

## Phase 1: Address History Collector

### Proposed script or command

- `scripts/extract_solana.py`
  or later
- `cmd/extractsolana`

### Input

- `--address`
- `--rpc-url`
- `--days 90` or `--max-signatures 5000`
- optional enhanced parser key

### Steps

1. call `getSignaturesForAddress`
2. paginate until:
    - 90-day window reached, or
    - max signature count reached
3. persist signatures locally

### Output

- `data/cases/extracted-solana/<address>.signatures.json`

---

## Phase 2: Parsed Transaction Hydration

For every collected signature:

1. call `getTransaction` with parsed encoding
2. extract:
    - success / failure
    - fee
    - programs
    - account keys
    - token balance changes
    - lamport changes
    - instruction summary
3. derive simplified counterparties and program usage

### Output

- `data/cases/extracted-solana/<address>.transactions.ndjson.gz`
- `data/cases/extracted-solana/<address>.json`

---

## Phase 3: Token Inventory Enrichment

### Calls

- `getTokenAccountsByOwner`
- optionally `getTokenAccountBalance`
- optionally `getAccountInfo`

### Output

- token account count
- active mint count
- top token balances
- stablecoin presence flags

### Merged into summary JSON

The main summary artifact should include both:

- transaction-derived behavior
- token inventory context

---

## Phase 4: Optional Enhanced Parsing Layer

If enabled, use Helius Enhanced Transactions to enrich:

- swap-like activity
- NFT-related activity
- DeFi interactions
- clearer transfer semantics

### Why optional

The MVP should not depend entirely on a paid indexer.
The enhanced layer should improve:

- readability
- case study quality
- protocol interpretation

but not block the baseline Solana ingestion path.

---

## Proposed Output Schema

## Raw extracted file

`data/cases/extracted-solana/<address>.transactions.ndjson.gz`

Each line should contain a simplified transaction object:

- `signature`
- `slot`
- `block_time`
- `success`
- `error`
- `fee_lamports`
- `signers`
- `account_keys`
- `program_ids`
- `native_balance_changes`
- `token_balance_changes`
- `counterparties`
- `instruction_count`
- `inner_instruction_count`

---

## Summary file

`data/cases/extracted-solana/<address>.json`

```json
{
  "address": "ExampleSolanaAddress111111111111111111111111111111",
  "chain": "SOLANA",
  "generated_at": "2026-03-19T00:00:00Z",
  "summary": {
    "first_seen": "2026-01-01T00:00:00Z",
    "last_seen": "2026-03-19T00:00:00Z",
    "signature_count": 0,
    "failed_signature_count": 0,
    "success_signature_count": 0,
    "unique_counterparties": 0,
    "native_inbound_count": 0,
    "native_outbound_count": 0,
    "spl_transfer_count": 0,
    "stablecoin_transfer_count": 0,
    "token_accounts_count": 0,
    "active_token_mints_count": 0
  },
  "top_counterparties": [],
  "top_programs": [],
  "top_mints": [],
  "sample_transactions": [],
  "source_transaction_count": 0
}
```

---

## Counterparty Model for Solana

Solana counterparties are trickier than EVM sender/recipient pairs.

The MVP should use a pragmatic approach:

### Counterparty heuristics

- explicit transfer destination when clearly present
- token account owner when derivable
- repeated non-self account interaction
- known labeled destination or program-linked account
- token-account owner resolution where possible

This will not be perfect, but it is enough for:

- repeated interaction analysis
- concentration analysis
- case-study generation

---

## Program Interaction Model

Programs matter a lot on Solana.

The extractor should count:

- top program ids interacted with
- repeated high-frequency program interactions
- whether activity is dominated by a single protocol/program

This enables future heuristics like:

- concentration to a trusted protocol
- concentration to a suspicious service
- repeated interaction with flagged programs

---

## Stablecoin and Token Context

The Solana MVP should explicitly track:

- USDC presence
- stablecoin-heavy transfer activity
- active token mints
- SPL token breadth

This matters because many compliance-relevant flows are better understood through:

- token movement
- not just native SOL movement

---

## Current Recommended Repository Layout

```text
data/
  candidates/
    solana_addresses.txt
  cases/
    extracted-solana/
      <address>.transactions.ndjson.gz
      <address>.json
    curated-solana/
      <case>.json

docs/
  SOLANA-DATA-MODEL.md

scripts/
  extract_solana.py
```

---

## Validator / Curated Case Integration Plan

Once Solana extracted summaries exist:

1. create Solana curated cases
2. add Solana dataset-backed validator mode examples
3. surface:
    - signature count
    - failed transaction count
    - token footprint
    - top programs
    - counterparty breadth
4. later add Solana-specific scoring rules

This mirrors the current Ethereum trace-enriched curated workflow.

---

## Current Limitations

The Solana MVP model will still have boundaries.

### Not yet included

- full chain-wide Solana warehouse
- cluster/entity resolution
- exhaustive instruction semantic decoding
- bridge-aware chain-hopping modeling
- protocol-specific deep parsers for every major Solana protocol
- full graph-based hop analysis

### Why this is acceptable

The address-scoped MVP is enough to:

- add Solana to the supported chains story
- create realistic case artifacts
- support explainable wallet-level reasoning
- extend later without redoing the architecture

---

## Recommended Day 6 / Day 7 Implementation Order

1. add `docs/SOLANA-DATA-MODEL.md`
2. add `data/candidates/solana_addresses.txt`
3. build `scripts/extract_solana.py`
4. generate 2–3 address-scoped Solana examples
5. create first Solana curated case
6. add Solana-specific heuristics later, after data quality is validated

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/ETHEREUM-DATA-MODEL.md`](ETHEREUM-DATA-MODEL.md)
- [`docs/ERC20-DATA-MODEL.md`](ERC20-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
