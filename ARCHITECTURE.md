# Crypto Profiler Architecture

## Purpose

This document explains the current and planned system architecture for Crypto Profiler.

It is written to support:
- technical review
- architecture walkthroughs
- portfolio presentation
- implementation alignment across the MVP roadmap

Crypto Profiler is intentionally designed as an **explainable wallet-risk platform**, not as a black-box scoring system.

---

## System Goals

Crypto Profiler is designed to answer practical investigator and compliance questions such as:

- Is this wallet valid and active?
- Is it sanctioned or directly linked to a risky entity?
- Is it repeatedly interacting with flagged infrastructure?
- Is most of its activity concentrated in a trusted service or a high-risk service?
- Does its behavior resemble mixer routing, pass-through movement, dusting, or burst activity?
- How can the system explain its conclusions in a regulator- or analyst-friendly way?

---

## Architectural Principles

1. **Deterministic-first**
   Sanctions and exact-match watchlist hits should override weaker heuristic interpretation.

2. **Explainable by construction**
   Every score adjustment should be tied to visible reason codes and evidence.

3. **Chain-specific ingestion, chain-agnostic scoring**
   Bitcoin, EVM transactions, ERC-20 events, and EVM traces can have different ingestion paths while feeding a shared scoring model.

4. **Portfolio-grade realism**
   The system should support realistic case studies, not just toy examples.

5. **Incremental graph expansion**
   The MVP focuses on direct exposure and targeted summaries, while leaving room for future 1-hop / 2-hop graph traversal.

---

## High-Level Architecture

```text
                        +---------------------------+
                        |   CLI / Dataset Mode      |
                        |   cmd/validator           |
                        +-------------+-------------+
                                      |
                                      v
                        +---------------------------+
                        |   Validation Layer        |
                        |   address strategies      |
                        +-------------+-------------+
                                      |
                                      v
     +----------------+   +---------------------------+   +----------------------+
     | Watchlist      |-->|  Analyzer / Scoring       |<--|  Bootstrap Labels    |
     | Engine         |   |  internal/analyzer        |   |  known entities      |
     +----------------+   +---------------------------+   +----------------------+
                                      ^
                                      |
                  +-------------------+-------------------+
                  |                   |                   |
                  |                   |                   |
                  v                   v                   v
      +-------------------+  +-------------------+  +----------------------+
      | Bitcoin datasets  |  | EVM tx / ERC-20   |  | EVM traces datasets   |
      | tx/input/output   |  | Blockchair        |  | BigQuery export       |
      +-------------------+  +-------------------+  +----------------------+
                  |                   |                   |
                  +-------------------+-------------------+
                                      |
                                      v
                        +---------------------------+
                        | Extract / Curate Layer    |
                        | datasets + case artifacts |
                        +---------------------------+
```

---

## Core Layers

## 1. Ingestion Layer

The ingestion layer is responsible for acquiring raw blockchain and watchlist data.

### Current sources

#### A. Watchlist / sanctions source
- watchlist engine
- OFAC-driven sanctions synchronization
- exact-match sanctioned address screening

#### B. Bitcoin raw datasets
- Bitcoin transactions
- Bitcoin inputs
- Bitcoin outputs

#### C. EVM raw datasets
- Ethereum transactions
- ERC-20 transfers
- ERC-20 token metadata snapshot

#### D. EVM traces / internal calls
- Ethereum traces exported from BigQuery
- stored as Parquet in Cloud Storage
- extracted locally into address-scoped trace subsets

### Design intent
The ingestion layer is intentionally source-flexible:
- Blockchair works well for raw historical transaction files
- BigQuery works well for large Ethereum trace slices
- the watchlist engine provides deterministic sanctions screening

---

## 2. Normalization Layer

The normalization layer converts raw external formats into internal data structures that Crypto Profiler can reason over.

### Responsibilities
- normalize wallet addresses
- normalize chain naming
- normalize timestamps
- normalize raw value fields
- preserve source-specific fields needed for explanation
- build address-level and counterparty-level summaries

### Examples

#### Address normalization
- EVM addresses are lowercased and normalized to `0x...`
- Bitcoin addresses are treated as address-level entities, not guaranteed wallet clusters

#### ERC-20 normalization
- raw integer token values are preserved
- token metadata can enrich symbol/name/decimals

#### EVM traces normalization
- normalize:
    - sender
    - recipient
    - trace path
    - depth
    - status / failure
    - value-bearing internal calls
- produce address-scoped trace summaries

### Current outputs
- extracted datasets
- curated case artifacts
- internal transaction / trace summaries for analyzer use

---

## 3. Entity / Watchlist Layer

This layer provides the entity intelligence needed for risk interpretation.

### Current sources
- watchlist engine for sanctions
- bootstrap entity labels for:
    - mixers
    - scams
    - exploits
    - exchanges
    - trusted protocols

### Current behavior
- direct sanctions match short-circuit
- direct label on profiled wallet
- direct counterparty label detection
- trusted context mitigation
- repeated flagged-counterparty interaction
- service concentration using labeled counterparties

### Why this layer matters
Raw blockchain movements alone are not enough.  
This layer provides the meaning required to distinguish:
- trusted high-volume protocol activity
- exchange-related behavior
- mixer exposure
- scam or exploit linkage
- sanctions violations

---

## 4. Graph / Exposure Layer

This layer is partially implemented in the MVP and is designed to expand over time.

### Current MVP level
- direct counterparties
- repeated flagged interaction
- top counterparties from extracted datasets
- address-scoped trace counterparties for EVM internal-call context

### Planned next level
- 1-hop exposure
- 2-hop exposure
- path retention for explainability
- cluster/entity-aware repeated interaction

### Why traces matter here
Ethereum top-level transactions alone miss internal contract-mediated movement.  
The traces layer provides a stronger foundation for:
- router behavior
- internal pass-through reasoning
- round-trip / U-turn groundwork
- future internal-flow exposure summaries

---

## 5. Scoring Engine

The scoring engine lives in `internal/analyzer`.

### Current scoring model
- weighted categories:
    - `FRAUD`
    - `REPUTATION`
    - `LENDING`
- weighted combined risk score
- review recommendation decision
- reason-code output

### Current supported signals
- direct sanctions match
- profiled high-risk entity label
- direct mixer interaction
- direct high-risk entity interaction
- trusted / exchange interaction
- established history
- fresh wallet
- high-velocity behavior
- noisy inbound activity
- zero-value inbound pattern
- high counterparty fan-in
- repeated flagged-counterparty interaction
- high-risk service concentration
- trusted / protocol concentration
- exchange concentration
- combination rules such as:
    - mixer + fresh wallet
    - mixer + high velocity
    - established wallet mitigation

### Design principle
The engine favors:
- deterministic signals first
- visible explanation
- contextual mitigation where appropriate
- low false-positive behavior for observed-only patterns

---

## 6. Explanation Layer

This layer converts scoring decisions into reviewable evidence.

### Output shape
- risk score
- risk grade
- review recommendation
- risk breakdown by category
- individual `risk_reasons`
- evidence counts
- related entity / address where available

### Why it matters
A wallet-risk engine is only useful if an analyst can understand:
- what fired
- why it fired
- how strong the evidence was
- whether context reduced or increased confidence

### Current explanation strength
The project already supports:
- deterministic sanctions explanations
- mixer exposure reasoning
- trusted protocol contextual mitigation
- repeated interaction evidence
- concentration reasoning
- curated case-study storytelling

---

## 7. API / UI Layer

### Current MVP form
- CLI entrypoint via `cmd/validator`
- dataset-backed validator mode for reproducible demos
- JSON-first outputs

### Why this is enough for MVP
The project is currently optimized for:
- architecture demonstration
- portfolio evidence
- reproducible scoring behavior
- curated cases and explainable outputs

### Planned future expansion
- HTTP API
- dashboard or analyst review UI
- batch analysis endpoints
- case management integration

---

## 8. Extract / Curate Layer

This layer sits between raw source data and analyst-facing examples.

### Current commands
- `cmd/extractcases`
- `cmd/curatecases`
- `cmd/enrichcases`

### Current Python helper
- `scripts/extract_traces.py`

### Current responsibilities
- reduce large raw datasets into per-address extracted artifacts
- create curated case files with:
    - summary statistics
    - top counterparties
    - capped sample data
    - risk posture metadata
- create address-scoped trace summaries from large Parquet trace exports
- merge address-scoped trace summaries into curated case artifacts
- surface trace-aware context in dataset-backed validator flows

### Why this layer matters
This layer gives the project:
- stable portfolio artifacts
- reproducible demo cases
- realistic data without forcing live API dependence

---

## 9. Storage Model

### In-repo
The repository should keep:
- source code
- docs
- small curated JSON artifacts
- small configuration and label files

### Out-of-repo
Large datasets should stay outside git:
- raw Blockchair files
- full Ethereum traces Parquet export
- large address-scoped compressed trace files

### Recommended storage pattern
- Cloud Storage or external disk for raw export files
- local extracted summaries for development
- curated artifacts in repo for demos and tests

---
## Solana Layer 1

The current Solana architecture is intentionally **stablecoin-flow first**.

### Current Solana Layer 1 inputs

- locally exported whale stablecoin-flow Parquet shards
- address-scoped extracted Solana stablecoin summaries
- curated Solana case artifacts built from those extracted summaries

### Current Solana Layer 1 responsibilities

- identify large-value stablecoin transfer activity
- summarize source / destination / authority roles
- count repeated counterparty interaction
- surface broad counterparty and authority-control patterns
- produce curated Solana benchmark cases for demos and future scoring work

### Current limitation

The current Solana layer is not yet full instruction-aware or program-aware.
It is a value-flow layer designed to establish an MVP baseline quickly and economically.

### Current end-to-end Solana flow

1. export whale stablecoin flows from historical Solana data
2. extract address-scoped stablecoin summaries locally
3. mine candidate wallets from the extracted value-flow layer
4. curate selected Solana case artifacts
5. use those artifacts to drive future Solana scoring and case-study work

---

## Current Data Coverage

The current MVP data foundation now includes:

### Bitcoin
- transactions
- inputs
- outputs

### EVM
- Ethereum transactions
- ERC-20 transactions
- ERC-20 token metadata snapshot
- Ethereum traces export
- address-scoped extracted trace datasets

This means the MVP now has both:
- top-level EVM activity context
- internal-call / trace context

---

## Current Heuristic Coverage

### Implemented
- sanctions short-circuit
- direct high-risk counterparty exposure
- repeated flagged-counterparty interaction
- velocity burst
- noisy inbound / dusting-like observation
- high counterparty fan-in
- trusted protocol / exchange context
- service concentration to high-risk or trusted services

### Partially Implemented
- fresh wallet with immediate flow
- broader mixer/tumbler exposure
- trace-aware context as a separate extracted layer rather than fully wired into the scoring engine

### Planned
- 1-hop / 2-hop exposure
- peel-chain behavior
- round-trip / U-turn behavior
- deeper trace-aware EVM behavioral rules
- value-weighted and cluster-aware concentration

---

## Example End-to-End Flows

## A. Live wallet flow
1. user supplies wallet address
2. validator chooses chain strategy
3. wallet is validated and normalized
4. watchlist engine checks sanctions
5. analyzer loads labels and transaction context
6. scoring rules fire
7. explainable JSON output is returned

## B. Dataset-backed curated case flow
1. curated JSON case is loaded
2. validator runs in dataset mode
3. analyzer applies scoring using stable case inputs
4. consistent JSON output is produced for demos and tests

## C. Trace extraction flow
1. Ethereum traces are exported from BigQuery to Parquet
2. `scripts/extract_traces.py` scans Parquet files
3. traces are filtered to tracked addresses
4. address-scoped trace summaries and compressed raw subsets are written
5. extracted trace artifacts support later case and heuristic enrichment

## D. Trace-enriched curated case flow
1. curated case JSON is generated from transaction datasets
2. address-scoped trace summary is loaded
3. `cmd/enrichcases` merges trace context into the curated artifact
4. validator dataset mode loads the enriched case
5. output includes trace-aware explanation context

---

## Operational Notes

### Why BigQuery was used for traces
Downloading 90 days of Ethereum calls from flat-file providers was too slow for MVP iteration.  
BigQuery provided a practical path to export a 90-day raw traces base table.

### Why Python was used for trace extraction
The trace export is large enough that a small streaming-oriented PyArrow utility is a pragmatic extraction tool.  
The scoring engine remains Go-first.

### Why the scoring engine is still Go-first
The analyzer, validator, tests, and core portfolio signal logic all remain in Go.  
The Python script is an ingestion-side helper, not the product core.

---

## Future Architecture Expansion

### Near-term
- trace-aware curated case enrichment
- Bitcoin extracted case generation
- trace-informed protocol case studies
- stronger concentration reasoning

### Mid-term
- graph traversal service
- hop-based exposure engine
- value-aware repeated interaction
- router / internal-flow reasoning in analyzer

### Later
- HTTP API
- analyst review UI
- persistent graph store
- cluster resolution layer
- cross-chain and bridge-aware analytics

---

## Related Documents

- [`README.md`](README.md)
- [`docs/TYPOLOGIES.md`](docs/TYPOLOGIES.md)
- [`docs/BITCOIN-DATA-MODEL.md`](docs/BITCOIN-DATA-MODEL.md)
- [`docs/ERC20-DATA-MODEL.md`](docs/ERC20-DATA-MODEL.md)
