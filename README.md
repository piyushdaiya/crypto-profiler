# Crypto Profiler

[![CI](https://github.com/piyushdaiya/crypto-profiler/actions/workflows/ci.yml/badge.svg)](https://github.com/piyushdaiya/crypto-profiler/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8)](#tech-stack)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-active%20mvp-blue)](#current-implementation-status)

Crypto Profiler is a Go-based wallet risk profiling project focused on explainable, portfolio-grade crypto intelligence across multiple chains.

The repository currently combines:

- shared live scoring for EVM wallets
- curated dataset-mode scoring for Solana and Bitcoin Layer 1
- trace-aware Ethereum case enrichment
- a watchlist-driven sanctions path

The immediate roadmap after this Wave 1 alignment is ERC-20 Layer 1 scoring and curated ERC-20 dataset support.

---

## Overview

Crypto Profiler is built for KYW, AML, fraud, sanctions, and investigative-style wallet review.

The goal is not to be a full chain warehouse. The goal is to make practical wallet profiling explainable and reproducible, with realistic case artifacts and scoring that can be reviewed rule by rule.

---

## Current Implementation Status

| Area | Status today | Notes |
| --- | --- | --- |
| EVM live wallet profiling | Implemented | Uses Etherscan transaction history, watchlist checks, bootstrap labels, and the shared analyzer. |
| Ethereum curated Layer 1 cases | Implemented | Built from extracted address-scoped native and ERC-20 transfer activity. |
| Ethereum trace integration | Implemented | Address-scoped traces can be extracted, merged into curated cases, and surfaced in dataset mode. |
| Solana Layer 1 | Implemented in dataset mode | Current Solana layer is stablecoin-flow based and curated from large-value USDC/USDT summaries. |
| Bitcoin Layer 1 | Implemented in dataset mode | Current Bitcoin layer is address-level UTXO-flow based. |
| Solana live Layer 1 scoring | Not implemented | Live Solana strategy currently provides address validation and basic activity/balance lookup only. |
| Bitcoin live Layer 1 scoring | Not implemented | Live Bitcoin strategy currently provides address validation and basic activity/balance lookup only. |
| ERC-20 Layer 1 scoring | Not implemented | Extraction groundwork exists, but ERC-20 curated cases and validator scoring are Wave 2 work. |

---

## Multi-Chain Layer 1 Story

### Ethereum Layer 1

Ethereum is the most mature path in the repo today.

Implemented now:

- live analyzer scoring from top-level EVM activity and labels
- curated EVM case generation from extracted address-scoped transfer data
- optional trace enrichment for curated cases
- validator dataset mode that surfaces trace-aware internal-call context

Not implemented yet:

- ERC-20-specific Layer 1 scoring
- trace-driven live scoring
- hop-based or graph-aware exposure

### Solana Layer 1

Solana Layer 1 is currently stablecoin-flow based.

Implemented now:

- extracted stablecoin summaries from local whale-flow exports
- curated Solana cases under `data/cases/curated-solana/`
- validator dataset-mode scoring for role-heavy and broad-surface stablecoin behavior

Not implemented yet:

- general instruction-aware Solana profiling
- non-stablecoin Solana Layer 1 scoring
- live Solana Layer 1 scoring from full extracted history

### Bitcoin Layer 1

Bitcoin Layer 1 is currently UTXO-flow based.

Implemented now:

- extracted address-scoped Bitcoin summaries from local Blockchair inputs/outputs
- curated Bitcoin cases under `data/cases/curated-bitcoin/`
- validator dataset-mode scoring for spend-heavy, inbound-heavy, mixed-flow, and broad-surface behavior

Not implemented yet:

- cluster-aware modeling
- change detection
- graph-aware peel-chain or pass-through scoring

---

## Validator Dataset Mode

Validator dataset mode currently supports:

- curated Ethereum cases in `data/cases/curated/`
- trace-enriched Ethereum cases in `data/cases/curated-enriched/`
- curated Solana cases in `data/cases/curated-solana/`
- curated Bitcoin cases in `data/cases/curated-bitcoin/`

Examples:

```bash
go run ./cmd/validator --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
go run ./cmd/validator --dataset ./data/cases/curated-solana/solana-stablecoin-authority-operator.json
go run ./cmd/validator --dataset ./data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json
```

Dataset mode is the current delivery path for:

- reproducible demos
- case-study walkthroughs
- trace-aware EVM examples
- Solana Layer 1 stablecoin-flow scoring
- Bitcoin Layer 1 UTXO-flow scoring

---

## Live Validator Mode

The live validator currently supports three chain strategies:

- EVM via Etherscan
- Bitcoin via Blockchain.com
- Solana via CoinStats

Example:

```bash
go run ./cmd/validator 0xd90e2f925da726b50c4ed8d0fb90ad053324f31b
```

Important nuance:

- live EVM has the strongest current scoring path
- live Bitcoin and live Solana are still basic address-state lookups, not full Layer 1 dataset scoring

---

## Data Pipeline

The repo includes an intentionally lightweight extract-and-curate workflow.

### Ethereum

- `cmd/extractcases` builds address-scoped EVM extracted datasets
- `cmd/curatecases` turns extracted datasets into curated cases
- `scripts/extract_traces.py` builds address-scoped trace summaries
- `cmd/enrichcases` merges trace summaries into curated EVM cases

### Solana

- `scripts/mine_solana_whale_candidates.py` mines stablecoin-flow candidates
- `scripts/extract_solana_stablecoin.py` builds extracted stablecoin summaries
- `scripts/curate_solana_stablecoin.py` creates curated Solana cases

### Bitcoin

- `scripts/mine_bitcoin_candidates.py` mines UTXO-flow candidates
- `scripts/extract_bitcoin_layer1.py` builds extracted Bitcoin summaries
- `scripts/curate_bitcoin_layer1.py` creates curated Bitcoin cases

---

## Current Curated Cases

### Ethereum

- `data/cases/curated/public-wallet-noisy-inbound.json`
- `data/cases/curated/tornado-router-high-risk.json`
- `data/cases/curated/uniswap-v3-router-trusted-protocol.json`
- trace-enriched variants under `data/cases/curated-enriched/`

### Solana

- `data/cases/curated-solana/solana-usdc-distributor-treasury-like.json`
- `data/cases/curated-solana/solana-stablecoin-authority-operator.json`
- `data/cases/curated-solana/solana-broad-surface-authority-mixed-stablecoin.json`

### Bitcoin

- `data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json`
- `data/cases/curated-bitcoin/bitcoin-noisy-inbound-broad-surface.json`
- `data/cases/curated-bitcoin/bitcoin-legacy-mixed-flow-broad-value.json`

---

## Documentation Map

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](docs/TYPOLOGIES.md)
- [`docs/SCORING.md`](docs/SCORING.md)
- [`docs/EVM-CALLS-INTEGRATION.md`](docs/EVM-CALLS-INTEGRATION.md)
- [`docs/ETHEREUM-DATA-MODEL.md`](docs/ETHEREUM-DATA-MODEL.md)
- [`docs/SOLANA-DATA-MODEL.md`](docs/SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](docs/BITCOIN-DATA-MODEL.md)
- [`docs/ERC20-DATA-MODEL.md`](docs/ERC20-DATA-MODEL.md)
- [`docs/DATA-SOURCING-POLICY.md`](docs/DATA-SOURCING-POLICY.md)

Recommended reading order for the current repo state:

1. this README
2. `ARCHITECTURE.md`
3. `docs/TYPOLOGIES.md`
4. `docs/SCORING.md`
5. chain-specific data model docs

---

## Security

The repo includes:

- a watchlist / sanctions engine
- malformed-input and validator safety tests
- initial OWASP-oriented security coverage

See:

- [`SECURITY.md`](SECURITY.md)
- [`docs/security/owasp-test-matrix.md`](docs/security/owasp-test-matrix.md)

---

## Testing

Run locally:

```bash
go test ./...
go build ./...
```

Dataset-mode validation examples:

```bash
go run ./cmd/validator --dataset ./data/cases/curated-enriched/uniswap-v3-router-trusted-protocol.json
go run ./cmd/validator --dataset ./data/cases/curated-solana/solana-usdc-distributor-treasury-like.json
go run ./cmd/validator --dataset ./data/cases/curated-bitcoin/bitcoin-noisy-inbound-broad-surface.json
```

---

## Wave 2 Follow-On Work

Wave 2 is intentionally separate from this Wave 1 alignment pass.

Next major items are:

- ERC-20 Layer 1 curated extraction and scoring
- ERC-20 validator dataset support
- 1-hop and 2-hop exposure summaries
- pass-through and U-turn behavior
- richer graph-aware reasoning

---

## Tech Stack

- Go 1.23
- Docker / Docker Compose
- watchlist-driven sanctions checks
- Blockchair historical datasets for EVM and Bitcoin extraction
- BigQuery exports for Ethereum traces and Solana stablecoin-flow source data
