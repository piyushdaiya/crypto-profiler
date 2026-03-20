# Crypto Profiler

[![CI](https://github.com/piyushdaiya/crypto-profiler/actions/workflows/ci.yml/badge.svg)](https://github.com/piyushdaiya/crypto-profiler/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8)](#tech-stack)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-active%20mvp-blue)](#roadmap)

**Wallet Risk & Exposure Intelligence for AML, Fraud, Sanctions, and Crypto Surveillance**

Crypto Profiler is a Go-based platform for profiling cryptocurrency wallets using deterministic checks, graph-based exposure analysis, behavioral heuristics, curated blockchain case datasets, and address-scoped EVM trace analysis.

It is designed for financial institutions, compliance teams, investigators, RegTech product teams, and solutions architects who need a practical way to perform **Know Your Wallet (KYW)** and crypto-risk analysis with explainable results.

---

## Overview

Blockchain data is transparent, but identifying meaningful risk is not.

A wallet can appear benign at first glance while still being:
- directly linked to a sanctioned or risky entity
- only 1–2 hops away from a mixer, exploiter, or laundering route
- behaving like a pass-through mule or cash-out wallet
- showing patterns consistent with layering, smurfing, dusting, or rapid depletion after dormancy

Crypto Profiler helps transform raw wallet activity into an explainable risk assessment by combining:
- **wallet validation**
- **watchlist and risky-entity screening**
- **direct and indirect exposure analysis**
- **behavioral pattern detection**
- **weighted explainable scoring**
- **investigator-friendly outputs**

---

## Why this project exists

Crypto-risk and financial-crime teams need more than a simple blacklist check.

They need tools that can answer questions such as:
- Is this wallet valid and active?
- Is it linked to known risky entities?
- How close is it to a mixer, scam, exploit, or sanctioned actor?
- Does its behavior resemble money laundering, structuring, or rapid cash-out activity?
- Why did the system assign this risk score?

Crypto Profiler is being built to answer those questions in a portfolio-grade, explainable, and extensible way.

---
### Day 6 Update

The repository now includes an initial **Solana Layer 1** dataset built from large-value stablecoin transfer flows.

This layer currently includes:

- local Solana whale stablecoin-flow exports covering `2025-03-16` to `2025-04-14`
- address-scoped extracted Solana stablecoin summaries
- first curated Solana case artifacts derived from stablecoin-flow behavior
- Solana candidate mining from stablecoin counterparties and authority surfaces

### Solana data status

Solana support is currently **Layer 1 stablecoin-flow based**. 
Solana Layer 1 is now backed by curated stablecoin-flow case artifacts under `data/cases/curated-solana/`.

This means the first Solana MVP slice focuses on:

- USDC / USDT value movement
- source / destination / authority roles
- repeated counterparty interaction
- concentration and broad-surface flow patterns

A deeper Solana instruction/program layer may be added later as a second-stage enrichment path.

### Curated Solana Cases

The repository now includes first-pass Solana curated cases derived from the Layer 1 stablecoin-flow dataset:

- `data/cases/curated-solana/solana-usdc-distributor-treasury-like.json`
- `data/cases/curated-solana/solana-stablecoin-authority-operator.json`
- `data/cases/curated-solana/solana-broad-surface-authority-mixed-stablecoin.json`

These cases currently represent Solana stablecoin-flow behavior, not full instruction-aware protocol semantics.
They are intended to support case studies, future scoring work, and dataset-backed Solana benchmarking.
---

## Core capabilities

### 1. Wallet validation
- chain-aware address validation
- checksum verification where applicable
- normalized wallet representation

### 2. Risk screening
- exact-match screening against labeled wallets and risky entities
- support for sanctions/watchlist integration
- internal label support for exchanges, mixers, scams, exploit wallets, and trusted entities

### 3. Exposure analysis
- direct counterparty checks
- repeated flagged-counterparty interaction detection
- weighted exposure scoring
- transaction and trace-aware reasoning foundation
- architecture prepared for future 1-hop and 2-hop graph traversal

### 4. Behavioral detection
Initial and planned heuristics include:
- peeling-chain style layering
- smurfing / structured transfers
- hop-to-mixer proximity
- dusting and sweep patterns
- high-velocity burst activity
- pass-through / rapid outflow behavior
- service concentration to trusted or high-risk infrastructure

### 5. Explainable scoring
- weighted score from 0–100
- severity bands and review guidance
- triggered rules and evidence
- rationale string for analyst review

### 6. Investigator-ready output
- structured JSON reports
- CLI-readable summaries
- case-study friendly sample outputs
- dataset-backed demo mode
- trace-aware extracted artifacts for future case enrichment

---

## Architecture

See the full architecture document here:

[`ARCHITECTURE.md`](ARCHITECTURE.md)

### High-level flow

1. Accept wallet address and chain context
2. Validate and normalize the address
3. Retrieve transaction, trace, and label context
4. Build exposure and counterparty summaries
5. Apply deterministic and heuristic rules
6. Compute weighted explainable risk score
7. Generate JSON and analyst-friendly output

### Design principles

- **Go-first implementation**
- **Explainable scoring over black-box decisions**
- **Deterministic-first risk detection**
- **Graph-aware exposure analysis**
- **Modular rule engine**
- **Docker-friendly local execution**
- **Trace-aware EVM expansion path**
- **Designed to integrate with a future shared watchlist engine**

---

## Data Model Documents

The project now includes explicit data-model documentation for the current MVP sources:

- [`docs/BITCOIN-DATA-MODEL.md`](docs/BITCOIN-DATA-MODEL.md)
- [`docs/ERC20-DATA-MODEL.md`](docs/ERC20-DATA-MODEL.md)
- [`docs/TYPOLOGIES.md`](docs/TYPOLOGIES.md)
- [`docs/ETHEREUM-DATA-MODEL.md`](docs/ETHEREUM-DATA-MODEL.md)
- [`docs/DATA-SOURCING-POLICY.md`](docs/DATA-SOURCING-POLICY.md)
- [`docs/SOLANA-DATA-MODEL.md`](docs/SOLANA-DATA-MODEL.md)
- [`docs/DATA-SOURCING-POLICY.md`](docs/DATA-SOURCING-POLICY.md)

These documents define:
- the canonical MVP data window
- how raw source files are interpreted
- what behavior the current engine can support
- what remains planned for later phases

---

## Security

Crypto Profiler includes an initial security baseline covering:

- secure watchlist client behavior
- malformed input handling
- sanctions short-circuit decisioning
- OWASP Phase 1 security-focused tests

See:

- [`SECURITY.md`](SECURITY.md)
- [`docs/security/owasp-test-matrix.md`](docs/security/owasp-test-matrix.md)

---

## v0.1 MVP scope

The first public milestone is intentionally focused.

### In scope
- Bitcoin and Ethereum / EVM-first wallet model
- wallet validation and normalization
- exact-match risky wallet/entity checks
- direct and limited hop-based exposure analysis
- a small set of high-value behavioral heuristics
- explainable scoring with reason codes
- JSON and CLI outputs
- reproducible demo data and case-study examples
- address-scoped EVM traces extraction for internal-call context

### Out of scope for v0.1
- full-chain ingestion into a persistent warehouse
- production UI dashboard
- mempool surveillance
- ML-first scoring
- fuzzy entity resolution / name matching
- full cross-chain attribution
- complete market-manipulation surveillance engine
- full case management workflows

---

## Dataset-Backed Profiling

In addition to live profiling flows, Crypto Profiler supports curated dataset mode for reproducible demos and case studies.

### Example

```bash
go run ./cmd/validator --dataset ./data/cases/curated/tornado-router-high-risk.json
go run ./cmd/validator --dataset ./data/cases/curated/uniswap-v3-router-trusted-protocol.json
go run ./cmd/validator --dataset ./data/cases/curated/public-wallet-noisy-inbound.json
```

This mode is useful for:

- portfolio demos
- stable case-study walkthroughs
- screenshot/video generation
- testing without live API drift

---

## Data Pipeline

Crypto Profiler includes a lightweight data pipeline for creating portfolio-grade blockchain case artifacts.

### Extraction
Raw Blockchair Ethereum and ERC-20 transaction files can be scanned to build per-address extracted datasets.

### Curation
Large extracted datasets can then be reduced into curated case files containing:

- summary statistics
- top counterparties
- capped sample transfers
- case metadata and risk posture

### EVM traces extraction
A separate EVM trace extractor can scan exported Ethereum traces Parquet files and build per-address trace summaries containing:

- inbound / outbound / self trace counts
- failed trace counts
- value-bearing internal call counts
- max trace depth
- top trace counterparties
- sampled trace rows
- compressed raw trace subsets per address

### Trace-aware curated enrichment

Curated case artifacts can be enriched with address-scoped trace summaries:

```bash
go run ./cmd/enrichcases \
  --in ./data/cases/curated \
  --trace ./data/cases/extracted-traces \
  --out ./data/cases/curated-enriched
```

This allows dataset-backed validator runs to surface:
- internal trace counts
- failed internal call counts
- max trace depth
- broader internal counterparty surface


### Current curated cases

- `public-wallet-noisy-inbound`
- `tornado-router-high-risk`
- `uniswap-v3-router-trusted-protocol`

These complement the deterministic sanctioned-wallet case already demonstrated via the watchlist engine.



---

## Example use cases

Crypto Profiler is being designed for scenarios such as:

- screening a wallet before a transfer or onboarding decision
- triaging a wallet linked to suspicious inbound funds
- tracing whether a wallet is 1–2 hops from a mixer or exploit wallet
- identifying laundering-style behaviors such as peeling chains or rapid cash-out
- detecting persistent interaction with flagged counterparties
- distinguishing high-risk concentration from trusted protocol concentration
- generating structured risk evidence for analyst review
- demonstrating wallet intelligence architecture in regulated environments

---

## Sample output

```json
{
  "address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
  "network": "EVM",
  "is_valid": true,
  "validation_details": "Dataset Mode | High-Risk Mixer Infrastructure | Label: HIGH RISK: Tornado Cash (Router) | Risk Posture: REVIEWABLE_HIGH_RISK",
  "is_active": true,
  "balance": "",
  "tx_count": 1132,
  "first_seen": "2025-02-27T00:13:23Z",
  "last_seen": "2025-03-05T23:28:23Z",
  "risk_score": 22.5,
  "risk_grade": "ELEVATED",
  "review_recommended": true,
  "risk_breakdown": {
    "fraud_risk": 45,
    "reputation_risk": 0,
    "lending_risk": 0
  },
  "risk_reasons": [
    {
      "code": "profiled_address_high_risk_label",
      "category": "FRAUD",
      "description": "Profiled address labeled as high-risk entity: Tornado Cash Router",
      "offset": 45,
      "source": "bootstrap_entities",
      "related_entity": "Tornado Cash Router",
      "related_address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
      "severity": "HIGH",
      "confidence": "HIGH",
      "evidence_count": 1
    },
    {
      "code": "established_history",
      "category": "REPUTATION",
      "description": "Established History (>1 Year)",
      "offset": -10,
      "evidence_count": 1
    }
  ]
}
```

---

## Case Study: Established Wallet with Mixer Interaction

Crypto Profiler detected a direct interaction with a mixer-related entity, but did **not** over-penalize the wallet because there were no reinforcing suspicious signals such as fresh-wallet behavior, high-velocity bursts, or rapid pass-through activity.

This case demonstrates an important design principle in the scoring engine:

> Single-signal exposure should remain visible as evidence, while stronger conclusions should come from multi-signal correlation.

**Why it matters**
- reduces false positives
- improves analyst trust
- preserves explainability
- better reflects real compliance and investigative workflows

**Result**
- mixer interaction remained visible
- contextual mitigation was applied
- final score stayed low because suspicious context was absent

See the full write-up here:

[`docs/case-studies/established-wallet-mixer-no-reinforcing-signals.md`](docs/case-studies/established-wallet-mixer-no-reinforcing-signals.md)

---

## Case Study: Direct Sanctioned Wallet

Crypto Profiler uses the watchlist engine to detect a direct sanctions match and immediately short-circuits to a critical outcome.

This case demonstrates the deterministic end of the scoring model:
- maximum risk score
- critical grade
- mandatory review recommendation
- sanctions-first decisioning over heuristic scoring

See the full write-up here:

[`docs/case-studies/direct-sanctioned-wallet.md`](docs/case-studies/direct-sanctioned-wallet.md)

---

## Testing and CI

Crypto Profiler includes unit tests across the core MVP logic:

- address syntax validation for Bitcoin, EVM, and Solana
- analyzer scoring and combination rules
- repeated flagged-counterparty interaction
- high-risk and trusted service concentration
- watchlist client success/error handling
- validator CLI flows
- curated dataset loading and profiling support
- OWASP Phase 1 security-focused test coverage for watchlist and validator flows

Run locally:

```bash
go test ./...
go build ./...
```

GitHub Actions CI runs:

- `go build ./...`
- `go test ./...`
- Docker image build
- Docker Compose smoke test

---

## Watchlist Engine Verification

After starting the stack, verify that the watchlist engine initialized successfully and completed its sanctions sync.

### Start the stack

```bash
docker compose up -d --build
```

### Follow engine logs

```bash
docker compose logs -f engine
```

### Expected behavior

The engine should:

- start successfully
- expose the HTTP service on port `8080`
- initialize the sync loop
- detect updates to the sanctions source
- download and parse the OFAC feed
- rebuild the SQLite-backed sanctions database
- serve `/check` requests successfully

Example log flow:

```text
🔹 [ENGINE] Starting Watchlist Engine...
✅ [ENGINE] Database Available & Listening on :8080
🔹 [ENGINE] Initializing Sync Loop...
⬇️  [SYNC] Update Detected. Starting OFAC Download...
🔹 [SYNC] Parsing XML Stream...
✅ [SYNC] Done. Scanned <count> parties. Loaded <count> sanctioned addresses.
✅ [SYNC] Database Update Complete.
```

> Note: scanned-party and sanctioned-address counts may change over time as upstream sanctions data changes.

### Functional validation

Run a known sanctioned address through the validator:

```bash
docker compose exec validator ./validator bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx
```

Expected outcome:

- `risk_score = 100`
- `risk_grade = "CRITICAL (Sanctioned)"`
- `review_recommended = true`

This confirms that:

1. the validator is running
2. the watchlist engine is reachable
3. the sanctions lookup path is working end to end

---

## Tech stack

- **Go** for core logic and services
- **Docker** for local execution
- **Config-driven rules and scoring**
- **Blockchair-based extraction pipeline** for Ethereum and ERC-20 raw datasets
- **Bitcoin transactions / inputs / outputs** for UTXO-aware modeling
- **BigQuery + Cloud Storage** for Ethereum traces export
- **Python + PyArrow** for address-scoped EVM trace extraction
- **Curated case artifacts** for stable portfolio and demo flows

---

## License

This project is licensed under the MIT License. See [`LICENSE`](LICENSE).
