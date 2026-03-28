# Crypto Profiler

Portfolio-grade multi-chain crypto risk profiling for AML, sanctions, fraud, and regtech-style wallet review.

[![CI](https://github.com/piyushdaiya/crypto-profiler/actions/workflows/ci.yml/badge.svg)](https://github.com/piyushdaiya/crypto-profiler/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8)](#tech-stack)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-active%20mvp-blue)](#current-implementation-status)

Crypto Profiler is a Go-based wallet risk profiling project focused on explainable, portfolio-grade crypto intelligence across multiple chains.

The repository currently combines:

- shared live scoring for EVM wallets
- curated dataset-mode scoring for ERC-20, Solana, and Bitcoin Layer 1
- trace-aware Ethereum case enrichment
- a watchlist-driven sanctions path

The current next-step roadmap is deeper behavior scoring on top of the now-aligned multi-chain Layer 1 base.

---

## Portfolio Snapshot

- Built to demonstrate multi-chain Layer 1 reasoning without pretending Ethereum, Solana, Bitcoin, and ERC-20 all share the same data model.
- Produces explainable risk output with visible reasons, review recommendations, and an analyst-facing report mode for demos.
- Packages the engineering story end to end: extraction patterns, curated case artifacts, validator dataset mode, CI, security checks, docs, and sample outputs.

---

## Target Problem Space

Crypto Profiler is built for KYW, AML, fraud, sanctions, and investigative-style wallet review.

The goal is not to be a full chain warehouse. The goal is to make practical wallet profiling explainable and reproducible, with realistic case artifacts and scoring that can be reviewed rule by rule.

Why this matters for crypto-risk and regtech work:

- analysts and compliance reviewers need evidence they can inspect, not just opaque scores
- different chains expose different useful primitives, so the modeling layer should reflect reality
- curated benchmark cases are useful for demos, interviews, and regression testing when live data is noisy or unstable

---

## What This Project Demonstrates

This repository is designed to be portfolio-grade in both engineering and presentation.

It demonstrates:

- multi-chain Layer 1 thinking without pretending every chain should share one identical data model
- explainable risk scoring through visible `risk_reasons`, evidence counts, and review-oriented grades
- a practical curated-case workflow from extraction to analyst-facing report output
- disciplined repository storytelling through aligned docs, curated artifacts, tests, and CI security checks

---

## Why The Architecture Looks Like This

Crypto Profiler deliberately separates:

- live EVM scoring via a shared analyzer
- curated dataset-mode scoring for ERC-20, Solana, and Bitcoin Layer 1
- trace-aware Ethereum enrichment for cases where internal call context materially improves the story

That choice keeps the implementation honest:

- Ethereum is the most mature live path
- Solana, Bitcoin, and ERC-20 are real Layer 1 slices, but currently delivered through curated dataset mode
- the shared output contract stays consistent even when the ingestion and scoring path differs by chain

---

## Demo Entry Points

If you only have a few minutes, start here:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
go run ./cmd/validator --report --dataset ./data/cases/curated-solana/solana-stablecoin-authority-operator.json
go run ./cmd/validator --report --dataset ./data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json
go run ./cmd/validator --report --dataset ./data/cases/curated-erc20/erc20-uniswap-v2-router-trusted-token-hub.json
```

Use these for:

- a trace-aware Ethereum risk case
- a Solana authority-driven stablecoin-flow case
- a Bitcoin spend-heavy UTXO-flow case
- an ERC-20 trusted protocol token-surface case

Walkthrough notes live in [`docs/DEMO-WALKTHROUGH.md`](docs/DEMO-WALKTHROUGH.md).
Interview framing notes live in [`docs/INTERVIEW-TALK-TRACK.md`](docs/INTERVIEW-TALK-TRACK.md).
Sample report notes live in [`docs/sample-reports/README.md`](docs/sample-reports/README.md).

Static sample outputs live in:

- [`docs/sample-reports/ethereum-tornado-router.txt`](docs/sample-reports/ethereum-tornado-router.txt)
- [`docs/sample-reports/solana-authority-operator.txt`](docs/sample-reports/solana-authority-operator.txt)
- [`docs/sample-reports/bitcoin-operational-hub.txt`](docs/sample-reports/bitcoin-operational-hub.txt)
- [`docs/sample-reports/erc20-uniswap-v2-router.txt`](docs/sample-reports/erc20-uniswap-v2-router.txt)

---

## Analyst Report Mode

The validator now supports an analyst-facing report mode on top of the existing JSON output.

JSON remains the default:

```bash
go run ./cmd/validator --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
```

Use `--report` for a demo-friendly summary:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
```

Report mode is designed to surface:

- address and network
- case title and dataset context
- risk score, grade, and review recommendation
- top reasons
- top counterparties
- short interpretation
- chain-specific Layer 1 context

This mode is meant to make the repo easy to demo in interviews and portfolio reviews without changing the underlying JSON contract used by tests and engineering workflows.

---

## Current Implementation Status

| Area                               | Status today                | Notes                                                                                                                                            |
|------------------------------------|-----------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| EVM live wallet profiling          | Implemented                 | Uses Etherscan transaction history, watchlist checks, bootstrap labels, and the shared analyzer.                                                 |
| Ethereum curated Layer 1 cases     | Implemented                 | Built from extracted address-scoped native and ERC-20 transfer activity.                                                                         |
| Ethereum trace integration         | Implemented                 | Address-scoped traces can be extracted, merged into curated cases, and surfaced in dataset mode.                                                 |
| ERC-20 Layer 1                     | Implemented in dataset mode | Address-scoped ERC-20 transfer summaries, curated cases, and validator dataset scoring are now in place.                                         |
| Solana Layer 1                     | Implemented in dataset mode | Current Solana layer is stablecoin-flow based and curated from large-value USDC/USDT summaries.                                                  |
| Bitcoin Layer 1                    | Implemented in dataset mode | Current Bitcoin layer is address-level UTXO-flow based.                                                                                          |
| Solana live Layer 1 scoring        | Not implemented             | Live Solana strategy currently provides address validation and basic activity/balance lookup only.                                               |
| Bitcoin live Layer 1 scoring       | Not implemented             | Live Bitcoin strategy currently provides address validation and basic activity/balance lookup only.                                              |
| ERC-20 live or graph-aware scoring | Not implemented             | Current ERC-20 support is curated dataset mode only; live token scoring, swap-aware interpretation, and graph-aware exposure remain future work. |

---

## Quickest Demo Path

For a short live walkthrough:

1. Start with Ethereum to show the strongest end-to-end path and trace-aware enrichment.
2. Jump to Solana or Bitcoin to prove the repo is genuinely multi-chain.
3. End on ERC-20 to show token-surface reasoning and contextual scoring.

If you only show one command, use:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
```

That example shows the strongest combination of curated data, trace context, differentiated reasons, and analyst-facing output.

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

- trace-driven live scoring
- hop-based or graph-aware exposure

### ERC-20 Layer 1

ERC-20 Layer 1 is now a dedicated dataset-mode path built from local Blockchair ERC-20 transfer shards plus the latest token metadata snapshot.

Implemented now:

- behavior-driven ERC-20 candidate mining from local transfer data
- address-scoped ERC-20 extraction with raw subset artifacts and summary JSON
- curated ERC-20 cases under `data/cases/curated-erc20/`
- validator dataset-mode scoring for trusted protocol hubs, noisy inbound token surfaces, broad token surfaces, mixed token activity, repeated counterparties, and token concentration

Not implemented yet:

- live ERC-20 scoring inside the EVM address strategy
- swap-aware decoding or protocol-intent interpretation
- trace-aware pass-through or U-turn scoring for ERC-20 flows
- hop-based or graph-aware token exposure

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
- curated ERC-20 cases in `data/cases/curated-erc20/`
- curated Solana cases in `data/cases/curated-solana/`
- curated Bitcoin cases in `data/cases/curated-bitcoin/`

Examples:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
go run ./cmd/validator --report --dataset ./data/cases/curated-solana/solana-stablecoin-authority-operator.json
go run ./cmd/validator --report --dataset ./data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json
go run ./cmd/validator --report --dataset ./data/cases/curated-erc20/erc20-uniswap-v2-router-trusted-token-hub.json
```

Dataset mode is the current delivery path for:

- reproducible demos
- case-study walkthroughs
- trace-aware EVM examples
- ERC-20 Layer 1 token-surface scoring
- Solana Layer 1 stablecoin-flow scoring
- Bitcoin Layer 1 UTXO-flow scoring

---

## Curated Case Coverage By Chain

The repo currently includes checked-in benchmark cases for:

- Ethereum native Layer 1 and trace-enriched Ethereum cases under `data/cases/curated/` and `data/cases/curated-enriched/`
- Solana stablecoin-flow Layer 1 cases under `data/cases/curated-solana/`
- Bitcoin UTXO-flow Layer 1 cases under `data/cases/curated-bitcoin/`
- ERC-20 Layer 1 token-surface cases under `data/cases/curated-erc20/`

These are not toy fixtures. They are the primary way the repo demonstrates repeatable scoring behavior, report rendering, and chain-specific Layer 1 interpretation.

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

### ERC-20

- `scripts/mine_erc20_candidates.py` mines ERC-20 Layer 1 candidates from local Blockchair transfer shards
- `scripts/extract_erc20_layer1.py` builds extracted ERC-20 summaries and compressed raw subsets
- `scripts/curate_erc20_layer1.py` creates curated ERC-20 cases

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

### ERC-20

- `data/cases/curated-erc20/erc20-uniswap-v2-router-trusted-token-hub.json`
- `data/cases/curated-erc20/erc20-exchange-like-broad-service-surface.json`
- `data/cases/curated-erc20/erc20-noisy-inbound-broad-token-surface.json`

---

## Documentation Map

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`docs/DEMO-WALKTHROUGH.md`](docs/DEMO-WALKTHROUGH.md)
- [`docs/INTERVIEW-TALK-TRACK.md`](docs/INTERVIEW-TALK-TRACK.md)
- [`docs/sample-reports/README.md`](docs/sample-reports/README.md)
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
- focused dataset-loader and dataset-mode regression tests
- reproducible `govulncheck` and `gosec` checks in CI and locally
- practical OWASP-oriented security coverage

See:

- [`SECURITY.md`](SECURITY.md)
- [`docs/security/owasp-test-matrix.md`](docs/security/owasp-test-matrix.md)

---

## Testing

Run locally:

```bash
make test
make test-verbose
make build
make security
```

Direct commands also work after `make security-tools` or when `./.tools/bin` is on your `PATH`:

```bash
go test ./... -v
govulncheck ./...
gosec ./...
```

What the current automated coverage emphasizes:

- validator dataset-mode routing across Ethereum, Solana, Bitcoin, and ERC-20 curated cases
- curated loader failure modes for malformed JSON and missing required fields
- chain-specific Solana, Bitcoin, and ERC-20 dataset scoring thresholds and differentiated reason generation
- file/path safety regressions around local trace-summary lookups
- watchlist, label-loading, and malformed-input CLI safety checks already in the repo

What it does not claim yet:

- fuzzing across the extraction scripts
- full secret scanning / SBOM generation
- deployment hardening or external service penetration testing

Dataset-mode validation examples:

```bash
go run ./cmd/validator --dataset ./data/cases/curated-enriched/uniswap-v3-router-trusted-protocol.json
go run ./cmd/validator --dataset ./data/cases/curated-erc20/erc20-exchange-like-broad-service-surface.json
go run ./cmd/validator --dataset ./data/cases/curated-solana/solana-usdc-distributor-treasury-like.json
go run ./cmd/validator --dataset ./data/cases/curated-bitcoin/bitcoin-noisy-inbound-broad-surface.json
```

---

## Next Practical Work

The next major items after the current implementation are:

- 1-hop and 2-hop exposure summaries
- pass-through and U-turn behavior
- fresh-wallet plus immediate large-flow reasoning
- richer graph-aware reasoning
- stronger live Solana and Bitcoin Layer 1 scoring

---

## Tech Stack

- Go 1.25
- Docker / Docker Compose
- watchlist-driven sanctions checks
- Blockchair historical datasets for EVM and Bitcoin extraction
- BigQuery exports for Ethereum traces and Solana stablecoin-flow source data
