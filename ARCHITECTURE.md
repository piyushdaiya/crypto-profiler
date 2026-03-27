# Crypto Profiler Architecture

## Purpose

This document describes the architecture that is actually implemented in the repository today.

The important Wave 1 clarification is that Crypto Profiler currently has:

- one shared live analyzer used primarily for EVM
- chain-specific dataset-mode scoring adapters for Solana and Bitcoin
- trace-aware Ethereum case enrichment

It does not yet have one fully unified graph-aware engine across all chains.

---

## Design Principles

### Deterministic first

Sanctions and direct labeled exposure should outweigh weaker heuristics.

### Explainable by construction

Every score should be backed by visible `risk_reasons`, evidence counts, and plain-English descriptions.

### Practical multi-chain realism

Different chains can have different ingestion and scoring paths, as long as they converge on a coherent output shape.

### Dataset mode as a first-class product surface

Curated dataset mode is not a toy in this repo. It is the main delivery path for:

- repeatable demos
- benchmark cases
- trace-aware Ethereum examples
- current Solana Layer 1 scoring
- current Bitcoin Layer 1 scoring

---

## Runtime Surfaces

### 1. Live validator flow

`cmd/validator` chooses an address strategy and returns a `WalletProfile`.

Current live strategies:

- `internal/address/evm.go`
- `internal/address/bitcoin.go`
- `internal/address/solana.go`

Current live behavior by chain:

- EVM: validation, balance/history fetch, watchlist check, shared analyzer scoring
- Bitcoin: validation, balance/history fetch, basic activity state
- Solana: validation, balance/history fetch, basic activity state

### 2. Dataset-backed validator flow

`cmd/validator` also supports `--dataset`.

Dataset mode currently dispatches to:

- shared curated EVM cases
- Solana stablecoin-flow curated cases
- Bitcoin UTXO-flow curated cases

This is where the repo currently delivers its multi-chain Layer 1 scoring story.

---

## High-Level Architecture

```text
                      +---------------------------+
                      |        cmd/validator      |
                      |   live mode + dataset     |
                      +-------------+-------------+
                                    |
                    +---------------+---------------+
                    |                               |
                    v                               v
         +----------------------+        +----------------------+
         |  Live Address Paths  |        |   Curated Datasets   |
         |  internal/address    |        |   internal/datasets  |
         +----------+-----------+        +----------+-----------+
                    |                               |
                    v                               v
         +----------------------+        +----------------------+
         | Shared Analyzer      |        | Chain-Specific       |
         | internal/analyzer    |        | Dataset Contexts     |
         +----------+-----------+        | cmd/validator/*      |
                    |                    +----------+-----------+
                    v                               |
         +----------------------+                   v
         | WalletProfile JSON   |        +----------------------+
         | explainable output   |<-------| WalletProfile JSON   |
         +----------------------+        | explainable output   |
                                         +----------------------+
```

---

## Core Modules

### `internal/address`

Responsible for:

- chain syntax validation
- live balance and activity lookup
- initial `WalletProfile` construction

This module does not currently provide full multi-chain Layer 1 scoring on its own.

### `internal/analyzer`

Responsible for the shared live-scoring engine:

- watchlist short-circuit
- bootstrap label interpretation
- wallet-age signals
- velocity
- repeated flagged interaction
- count-based service concentration
- noisy inbound observations
- combination rules

Today this is the strongest live-scoring path for EVM.

### `internal/datasets`

Responsible for:

- loading curated cases
- loading trace summaries
- building dataset-mode wallet profiles for curated EVM cases

### `cmd/validator/dataset_*_context.go`

Responsible for chain-specific dataset scoring adapters:

- `dataset_trace_context.go`
- `dataset_solana_stablecoin_context.go`
- `dataset_bitcoin_layer1_context.go`

This is an important architecture detail:

- EVM live scoring uses the shared analyzer
- Solana and Bitcoin Layer 1 currently score through chain-specific dataset adapters

---

## Chain Architecture Today

| Chain | Current source path | Current scoring path | Current limit |
| --- | --- | --- | --- |
| Ethereum | Etherscan live txs, extracted EVM datasets, optional trace summaries | Shared analyzer plus trace-aware dataset context | No ERC-20-specific scoring, no trace-driven live scoring |
| Solana | Local stablecoin-flow Parquet exports and curated cases | Dataset-mode stablecoin scoring adapter | No general instruction-aware live scoring |
| Bitcoin | Local Blockchair inputs/outputs and curated cases | Dataset-mode UTXO-flow scoring adapter | No cluster-aware or graph-aware scoring |
| ERC-20 | Extractor groundwork exists | Not yet a completed scoring path | Wave 2 target |

---

## Ethereum Layer 1 Architecture

Ethereum currently combines three pieces:

1. top-level live profiling from Etherscan
2. extracted address-scoped curated cases
3. optional trace enrichment for curated cases

### Current EVM flow

1. `cmd/extractcases` scans Blockchair Ethereum and ERC-20 transaction files
2. `cmd/curatecases` builds curated EVM cases
3. `scripts/extract_traces.py` builds address-scoped trace summaries
4. `cmd/enrichcases` merges trace summaries into curated EVM cases
5. `cmd/validator --dataset` loads the curated case and shared analyzer output
6. trace context is appended as observational reasoning

### Why traces are separate

The repo uses traces today to enrich explanation without overstating trace-native scoring maturity.

That keeps the implementation honest:

- trace extraction is real
- trace-aware explanation is real
- trace-native heuristics remain future work

---

## Solana Layer 1 Architecture

Solana Layer 1 is currently stablecoin-flow first.

### Current Solana flow

1. export or collect large-value stablecoin-flow source data outside git
2. `scripts/mine_solana_whale_candidates.py` identifies candidate addresses
3. `scripts/extract_solana_stablecoin.py` builds address-scoped stablecoin summaries
4. `scripts/curate_solana_stablecoin.py` creates curated Solana cases
5. `cmd/validator --dataset` applies Solana-specific stablecoin scoring

### Current Solana responsibilities

- source / destination / authority role interpretation
- broad counterparty surface scoring
- mixed-mint observation
- repeated large counterparty interaction

### Current Solana limitation

This is not yet a full Solana transaction- or program-semantics architecture.

It is a practical Layer 1 stablecoin-flow slice.

---

## Bitcoin Layer 1 Architecture

Bitcoin Layer 1 is currently UTXO-flow first.

### Current Bitcoin flow

1. local Blockchair inputs/outputs live outside git
2. `scripts/mine_bitcoin_candidates.py` identifies candidate addresses
3. `scripts/extract_bitcoin_layer1.py` builds address-scoped UTXO summaries
4. `scripts/curate_bitcoin_layer1.py` creates curated Bitcoin cases
5. `cmd/validator --dataset` applies Bitcoin-specific UTXO-flow scoring

### Current Bitcoin responsibilities

- inbound-receipt vs outbound-spend role analysis
- broad counterparty surface scoring
- operational hub scoring
- repeated counterparty interaction scoring

### Current Bitcoin limitation

This is address-level UTXO profiling, not cluster-aware wallet analytics.

---

## Watchlist and Labels

The entity layer has two active sources:

- watchlist engine for sanctions checks
- bootstrap entity labels for trusted services and high-risk infrastructure

This layer is used in both live scoring and curated EVM cases.

It is not yet a full entity-resolution or clustering system.

---

## Output Contract

Every current scoring path converges on the same high-level output shape:

- address
- network
- validity and validation details
- activity metadata
- `risk_score`
- `risk_grade`
- `review_recommended`
- `risk_breakdown`
- `risk_reasons`

This shared output contract is why the repo can mix:

- one shared analyzer
- several chain-specific dataset adapters

without losing coherence for the user.

---

## Data Storage Model

### Kept in git

- source code
- docs
- curated case JSON artifacts
- compact extracted reference fixtures used for examples
- small label/config files

### Kept outside git by default

- raw Blockchair dumps
- raw BigQuery / Parquet exports
- large working extraction outputs
- local candidate lists

The repo deliberately keeps the committed artifacts small enough to support demos and tests without trying to store a full historical warehouse.

---

## Near-Term Expansion

The architecture is set up so the next wave can add:

- ERC-20 Layer 1 curated cases and scoring
- trace-aware pass-through or U-turn rules
- 1-hop / 2-hop exposure summaries
- more graph-aware scoring

Those are next-stage additions, not claims about the current implementation.

---

## Related Documents

- [`README.md`](README.md)
- [`docs/TYPOLOGIES.md`](docs/TYPOLOGIES.md)
- [`docs/SCORING.md`](docs/SCORING.md)
- [`docs/EVM-CALLS-INTEGRATION.md`](docs/EVM-CALLS-INTEGRATION.md)
- [`docs/SOLANA-DATA-MODEL.md`](docs/SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](docs/BITCOIN-DATA-MODEL.md)
