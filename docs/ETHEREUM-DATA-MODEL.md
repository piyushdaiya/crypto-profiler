# Crypto Profiler Ethereum Data Model

## Purpose

This document defines the Ethereum / EVM core data model for Crypto Profiler.

It explains:

- which Ethereum datasets are in scope for the MVP
- how Ethereum transactions differ from ERC-20 transfers and traces
- how these datasets complement each other
- what Ethereum behaviors the MVP can support today
- what remains planned for later phases

This document is intentionally MVP-focused and aligned to the current architecture.

Wave 1 status note:

- Ethereum live scoring and curated EVM case support are implemented today.
- Trace extraction and trace-enriched curated cases are implemented today.
- ERC-20 extraction groundwork exists, but ERC-20 Layer 1 scoring and validator dataset support remain Wave 2 work.

---

## Scope

The Ethereum data model in Crypto Profiler is built from three related but distinct layers:

1. **Ethereum transactions**
2. **ERC-20 transfers**
3. **Ethereum traces / internal calls**

These layers serve different purposes.

- Ethereum transactions provide the top-level transaction frame.
- ERC-20 transfers provide token movement semantics.
- Ethereum traces provide internal contract-mediated execution context.

Together, they give a practical EVM foundation for explainable KYW and crypto-risk analysis.

---

## Canonical MVP Window

The current recommended shared EVM MVP window is:

- **2025-03-16 → 2025-06-17**

This window is used because it aligns with the current:
- Bitcoin outputs coverage
- Ethereum transactions coverage
- ERC-20 transfers coverage
- Ethereum traces export

This creates one consistent cross-source slice for:
- curated case generation
- dataset-backed demos
- trace-aware summaries
- future graph and exposure work

---

## Data Sources

## 1. Ethereum Transactions

### Current source
- Blockchair Ethereum transactions

### What this layer provides
- top-level transaction frame
- sender and recipient at the transaction level
- transaction timestamp
- block linkage
- top-level value transfer context

### What it does not provide
- internal contract-mediated calls
- router-internal fund movement
- nested execution paths

### Why it matters
This is still the simplest and most stable top-level EVM activity model.

---

## 2. ERC-20 Transfers

### Current source
- Blockchair ERC-20 transfers
- latest ERC-20 token metadata snapshot

### What this layer provides
- token transfer events
- token-aware sender/recipient activity
- token identity and metadata enrichment
- stablecoin and asset-mix context

### Why it matters
Ethereum transactions alone do not explain token movement.  
ERC-20 transfer events are required for token-centric behavior.

### Current implementation note
The repo can already extract and carry ERC-20 transfer context inside curated EVM cases, but ERC-20 does not yet have its own completed Layer 1 scoring path.

### Related document
- [`docs/ERC20-DATA-MODEL.md`](ERC20-DATA-MODEL.md)

---

## 3. Ethereum Traces / Internal Calls

### Current source
- BigQuery export of Ethereum traces
- Parquet export stored outside git
- address-scoped extracted trace summaries created locally

### What this layer provides
- internal sender/recipient call paths
- nested execution depth
- failed internal calls
- internal value-bearing calls
- contract routing behavior
- broader internal counterparty surfaces

### Why it matters
Top-level EVM transactions can hide important behavior inside contract execution.  
Traces make it possible to reason about:
- routers
- internal pass-through behavior
- deeper internal call stacks
- failed call patterns
- address-scoped internal activity summaries

---

## Core Ethereum Entities

The Ethereum model uses the following conceptual entities.

### A. Wallet / address
An externally visible EVM address that may represent:
- EOAs
- contract wallets
- router contracts
- protocol infrastructure
- exchanges
- mixers
- scam or exploit entities

Crypto Profiler currently treats addresses as first-class profiled objects.

### B. Transaction
A top-level Ethereum transaction with:
- sender
- recipient
- block
- timestamp
- top-level value
- tx hash

### C. Transfer event
An ERC-20 movement record emitted within a transaction.

### D. Trace row
An internal call / execution row with:
- transaction hash
- sender
- recipient
- trace path
- depth
- call type
- failure status
- internal value

---

## Why Transactions, ERC-20, and Traces Are Separate

These are not interchangeable.

### Ethereum transactions answer:
- who initiated the top-level transaction?
- what was the top-level destination?
- when did it happen?

### ERC-20 transfers answer:
- which tokens moved?
- between which addresses?
- how much token value moved?

### Traces answer:
- what happened inside the transaction execution?
- which internal calls fired?
- which internal recipients were involved?
- which parts failed?
- how deep did execution go?

This separation is essential to avoid over-simplified EVM modeling.

---

## Current Internal Trace Export Shape

The current trace export keeps these fields:

- `block_id`
- `time`
- `transaction_hash`
- `transaction_index`
- `trace_path`
- `depth`
- `trace_type`
- `call_type`
- `sender`
- `recipient`
- `value`
- `gas`
- `gas_used`
- `child_call_count`
- `status`
- `failed`
- `fail_reason`

This is intentionally a compact but useful internal-call schema.

---

## Address-Scoped Trace Extraction

The repository now includes a trace extractor that:

- scans exported Parquet shards
- filters rows where sender or recipient matches tracked addresses
- writes compressed per-address raw trace subsets
- writes summary JSON artifacts per tracked address

### Current extracted summary fields
- first seen
- last seen
- inbound trace count
- outbound trace count
- self trace count
- failed trace count
- value-bearing trace count
- unique counterparties
- max depth
- top trace counterparties
- source trace count

### Why this matters
This gives Crypto Profiler a practical way to use traces without requiring the full raw export to be loaded into the Go application.

---

## Current Ethereum-Aware Behaviors Supported

The current MVP can now support:

### Directly supported today
- top-level EVM wallet profiling
- profiled-address label handling
- repeated flagged-counterparty interaction
- concentration to a high-risk or trusted service
- dataset-backed profiling for known EVM cases
- trace-aware contextual summaries in curated case artifacts
- dataset-mode surfacing of internal-call context

### Supported indirectly
- protocol/router activity contextualization
- public-wallet noisy inbound analysis with EVM context
- trusted-vs-high-risk service distinction

### Not yet fully wired into live scoring
- trace-derived round-trip / U-turn detection
- internal-call graph traversal
- trace-aware repeated internal counterparty weighting
- trace-derived path scoring

---

## What the Current Trace Data Already Shows

The extracted trace summaries already highlight strong distinctions across example addresses:

### Public wallet
- extremely high inbound skew
- low outbound trace count
- many counterparties
- low failure rate

### High-risk routing infrastructure
- large bidirectional trace volume
- moderate max depth
- non-trivial failed call activity
- concentration toward known mixer context

### Trusted protocol routers
- extremely high trace volume
- very large counterparty surface
- deep routing paths
- large failed-call counts that are normal in high-activity protocol contexts

These observations are important because they help the project distinguish:
- malicious concentration
- public-wallet noise
- trusted protocol scale

---

## Ethereum Data Model Responsibilities by Layer

## Transaction layer
Responsible for:
- top-level transfer context
- transaction timestamps
- sender/recipient at transaction boundary
- case-level transfer sampling

## ERC-20 layer
Responsible for:
- token transfer interpretation
- token concentration
- token-level entity interaction
- stablecoin-heavy flow modeling later

## Trace layer
Responsible for:
- internal-call structure
- execution depth
- internal value-bearing movement
- failure context
- address-scoped internal counterparties

---

## Normalization Rules

### Address normalization
- lowercase
- `0x` prefix preserved
- empty values treated safely

### Timestamp normalization
- UTC / RFC3339-style output in extracted summaries

### Value normalization
- preserve raw numeric string or numeric-compatible representation
- do not assume floating-point precision is safe for scoring logic

### Trace path normalization
- preserve raw trace path
- also compute `depth` explicitly for easier downstream use

---

## Current Repository Commands Related to Ethereum

### Existing
- `cmd/extractcases`
- `cmd/curatecases`
- `cmd/validator`

### Added during trace work
- `scripts/extract_traces.py`
- `cmd/enrichcases`

### What they do
- build per-address extracted transaction datasets
- build curated transaction-backed cases
- merge trace-aware summaries into curated cases
- surface trace-aware context in validator dataset mode

---

## How Validator Dataset Mode Uses Ethereum Trace Context

The validator’s dataset mode now supports:
- loading curated cases that include trace summaries
- surfacing trace-aware context in `validation_details`
- adding observational trace-context reasons without destabilizing the live scoring engine

This is a deliberate design choice:
- use trace context now for explanation and dataset richness
- wire deeper trace-aware scoring rules incrementally later

---

## Current Limitations

The current Ethereum model still has clear MVP boundaries.

### Not yet supported at production depth
- full internal-call graph store
- trace-driven hop-based exposure
- trace-aware U-turn / round-trip logic
- protocol-specific call decoding
- bridge-aware chain-hopping logic
- value-weighted concentration by trace counterparties
- cluster-level EVM entity resolution

### Why this is acceptable
The current MVP already supports:
- realistic EVM demos
- credible protocol/router case studies
- explainable risk reasoning
- a practical path to deeper EVM intelligence later

---

## Next Ethereum-Focused Enhancements

### Near-term
- trace-aware curated case enrichment
- Ethereum protocol/router case studies
- value-aware service concentration
- repeated interaction weighted by trace context

### Mid-term
- trace-informed round-trip detection
- trace-aware internal pass-through heuristics
- 1-hop / 2-hop EVM exposure using internal call context

### Later
- protocol semantic decoding
- bridge-aware and cross-chain reasoning
- persistent graph-backed EVM entity analytics

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/ERC20-DATA-MODEL.md`](ERC20-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
