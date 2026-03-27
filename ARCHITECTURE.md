# Crypto Profiler Architecture

## Purpose

This document describes the architecture that is actually implemented in the repository today.

The current implementation picture is that Crypto Profiler has:

- one shared live analyzer used primarily for EVM
- chain-specific dataset-mode scoring adapters for ERC-20, Solana, and Bitcoin
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
- ERC-20 Layer 1 token-surface scoring
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
- ERC-20 Layer 1 curated cases
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
- loading chain-specific curated dataset cases
- building dataset-mode wallet profiles for curated EVM cases

### `cmd/validator/dataset_*_context.go`

Responsible for chain-specific dataset scoring adapters:

- `dataset_trace_context.go`
- `dataset_erc20_layer1_context.go`
- `dataset_solana_stablecoin_context.go`
- `dataset_bitcoin_layer1_context.go`

This is an important architecture detail:

- EVM live scoring uses the shared analyzer
- ERC-20, Solana, and Bitcoin Layer 1 currently score through chain-specific dataset adapters

---

## Chain Architecture Today

| Chain    | Current source path                                                                  | Current scoring path                             | Current limit                                                                      |
|----------|--------------------------------------------------------------------------------------|--------------------------------------------------|------------------------------------------------------------------------------------|
| Ethereum | Etherscan live txs, extracted EVM datasets, optional trace summaries                 | Shared analyzer plus trace-aware dataset context | No trace-driven live scoring, no hop-aware graph scoring                           |
| ERC-20   | Local Blockchair ERC-20 shards, latest token metadata snapshot, curated ERC-20 cases | Dataset-mode ERC-20 Layer 1 scoring adapter      | No live token scoring, no swap-aware decoding, no trace-aware pass-through scoring |
| Solana   | Local stablecoin-flow Parquet exports and curated cases                              | Dataset-mode stablecoin scoring adapter          | No general instruction-aware live scoring                                          |
| Bitcoin  | Local Blockchair inputs/outputs and curated cases                                    | Dataset-mode UTXO-flow scoring adapter           | No cluster-aware or graph-aware scoring                                            |

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

## ERC-20 Layer 1 Architecture

ERC-20 Layer 1 is now transfer-event first.

### Current ERC-20 flow

1. local Blockchair ERC-20 shards and token metadata stay outside git
2. `scripts/mine_erc20_candidates.py` identifies behavior-driven ERC-20 candidates from the canonical window
3. `scripts/extract_erc20_layer1.py` builds address-scoped summaries plus compressed raw subsets
4. `scripts/curate_erc20_layer1.py` creates curated ERC-20 benchmark cases
5. `cmd/validator --dataset` applies ERC-20-specific Layer 1 scoring

### Current ERC-20 responsibilities

- token-aware inbound vs outbound role summaries
- broad token counterparty surface scoring
- repeated counterparty interaction scoring
- token diversity and single-token concentration observation
- trusted protocol or exchange-style contextual reasoning when labels exist

### Current ERC-20 limitation

This is ERC-20 transfer-row Layer 1 scoring, not full DeFi intent decoding.

It does not yet reconstruct swaps, decode protocol semantics, or use traces for pass-through scoring.

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
It also provides trusted-service context for ERC-20 curated cases in dataset mode.

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
- [`docs/ERC20-DATA-MODEL.md`](docs/ERC20-DATA-MODEL.md)
- [`docs/SOLANA-DATA-MODEL.md`](docs/SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](docs/BITCOIN-DATA-MODEL.md)
