
# Crypto Profiler Architecture

## Purpose

This document describes the architecture that is actually implemented in the repository today.

The current implementation picture is that Crypto Profiler has:

- one shared live analyzer used primarily for EVM
- chain-specific dataset-mode scoring adapters for Optimism, Polygon, Arbitrum, ERC-20, Solana, and Bitcoin
- trace-aware Ethereum case enrichment
- an attribution resolver with Tier 1 precedence, bounded secondary corroboration, and actor/exposure refinement
- a bounded graph summary and graph-aware scoring layer
- an analyst-facing report layer on top of the validator output
- partial Phase 2 semantic enrichment now populated for Optimism and Arbitrum, with Polygon intentionally left tx-first / registry-aware for the current pass
- an initial populated Phase 3 cross-chain L2 mart, curated case set, and report path
- scaffolded Phase 2 L2 semantic enrichment for Optimism, Polygon, and Arbitrum
- scaffolded Phase 3 cross-chain L2 dataset routing and report support

It does not yet have one fully unified graph-aware engine across all chains, and Optimism, Polygon, and Arbitrum Phase 1 remain the production L2 path today while Phase 2 / Phase 3 semantic datasets are still being prepared.

---

## Architecture At A Glance

If you only skim one section, this is the story:

1. chain-specific source data is extracted into address-scoped summaries outside git
2. small, high-signal curated case artifacts are committed into `data/cases/...`
3. `cmd/validator --dataset` probes the case shape and routes to the right chain adapter
4. behavioral scoring produces a shared `WalletProfile`
5. Tier 1 attribution resolves actor and label context
6. secondary sources can corroborate or conflict without overriding Tier 1
7. a bounded actor-aware, hop-aware, and pass-through / U-turn refinement layer runs where attribution support is strong
8. a bounded graph summary and motif layer runs when attributed graph coverage is meaningful
9. `cmd/validator --report` turns that profile into a concise analyst-facing brief
10. future behavior layers will sit on top of that shared output contract rather than replacing it

This is why the project works well as both an engineering artifact and a portfolio demo.

---

## Design Principles

### Deterministic first

Sanctions and direct labeled exposure should outweigh weaker heuristics.

### Explainable by construction

Every score should be backed by visible `risk_reasons`, evidence counts, and plain-English descriptions.

### Attribution-aware, not attribution-only

Tier 1 labels can sharpen a score or suppress a false positive, but they do not replace behavioral reasoning.

### Secondary sources stay bounded

Corroborating sources can raise confidence, add modest bounded adjustments, or surface conflicts, but they do not get to dominate scoring on their own.

### Actor-aware refinement stays bounded

Repeated interaction and concentration can roll up to actor level, but only when attribution confidence is strong enough to justify it.

### Graph-aware output stays bounded

Graph summary and graph-aware score changes only appear when attributed graph coverage is meaningful enough to support them.

### Practical multi-chain realism

Different chains can have different ingestion and scoring paths, as long as they converge on a coherent output shape.

### Dataset mode as a first-class product surface

Curated dataset mode is not a toy in this repo. It is the main delivery path for:

- repeatable demos
- benchmark cases
- trace-aware Ethereum examples
- Optimism Phase 1 tx-only Layer 2 scoring
- Polygon Phase 1 tx-only Layer 2 scoring
- Arbitrum Phase 1 tx-only Layer 2 scoring
- ERC-20 token-surface scoring
- current Solana scoring
- current Bitcoin scoring
- reproducible graph-aware and attribution-aware report output
- partial Phase 2 semantic enrichment for Optimism and Arbitrum, with Polygon kept tx-first / registry-aware for the current pass
- initial populated Phase 3 cross-chain L2 case routing and report output


### Portfolio-ready outputs matter

The architecture is intentionally designed so the same validator can serve:

- machine-readable JSON for engineering workflows
- analyst-facing report output for demos and portfolio review

That keeps the repo useful for both implementation depth and presentation quality.

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
- Optimism Phase 1 curated cases
- Polygon Phase 1 curated cases
- Arbitrum Phase 1 curated cases
- initial curated cross-chain L2 pilot cases
- ERC-20 curated cases
- Solana curated stablecoin-flow cases
- Bitcoin curated UTXO-flow cases

The important nuance is that cross-chain L2 routing and reporting are no longer just scaffolded: the repo now has an initial populated cross-chain mart and first curated benchmark cases built from best-available L2 inputs rather than full semantic parity across all three chains.

### 3. Analyst-facing report flow

`cmd/validator --report` renders a concise analyst brief on top of either:

- live validator output
- curated dataset-mode output

The report layer is intentionally thin:

- it does not invent new scoring logic
- it presents existing scoring, reasons, attribution, actor/exposure findings, graph summary, counterparties, and case context in a demo-friendly shape

---

## High-Level Architecture

```mermaid
flowchart LR
    A["cmd/validator"] --> B["Live Address Paths<br/>internal/address"]
    A --> C["Curated Dataset Loaders<br/>internal/datasets"]
    B --> D["Shared Analyzer<br/>internal/analyzer"]
    C --> E["Chain-Specific Dataset Contexts<br/>cmd/validator/dataset_*_context.go"]
    D --> F["Attribution Resolver<br/>Tier 1 + Secondary Corroboration<br/>internal/attribution"]
    E --> F
    F --> G["Actor/Exposure Refinement<br/>actor-aware / hop-aware / pass-through<br/>internal/attribution"]
    G --> H["Bounded Graph Summary + Motifs<br/>coverage-gated graph reporting + bounded scoring"]
    H --> I["WalletProfile"]
    I --> J["JSON Output"]
    I --> K["Analyst Report Output<br/>--report"]
```

## Artifact Lifecycle

```mermaid
flowchart TD
    A["Chain-Specific Inputs<br/>Etherscan / BigQuery / Blockchair / Solana stablecoin exports / traces"] --> B["Extraction + Candidate Mining"]
    B --> C["Address-Scoped Summaries"]
    C --> D["Curated Case Artifacts<br/>data/cases/..."]
    D --> E["Validator Dataset Mode<br/>cmd/validator --dataset"]
    E --> F["Behavior Scoring"]
    F --> G["Attribution Resolution<br/>Tier 1 anchor + secondary corroboration"]
    G --> H["Actor/Exposure Refinement<br/>actor rollups / direct+near exposure / pass-through / U-turn"]
    H --> I["Bounded Graph Summary + Graph-Aware Scoring<br/>coverage-gated actor rollups / motifs / bounded modifiers"]
    I --> J["WalletProfile"]
    J --> K["JSON Output"]
    J --> L["Analyst Report Output<br/>cmd/validator --report"]
    J --> M["Future Behavioral Layer<br/>value-weighted graph scoring / fresh-wallet large-flow / richer live paths"]
```

### End-to-end data flow in practice

1. raw chain data is extracted into address-scoped summaries outside git
2. curated benchmark cases are committed into `data/cases/...`
3. `cmd/validator --dataset` probes the case shape and chooses the right chain adapter
4. chain-specific or shared behavior scoring produces a baseline `WalletProfile`
5. Tier 1 attribution resolves contextual or risk-escalating actor information
6. secondary corroboration can raise confidence or surface conflicts
7. actor-aware repeated-interaction or concentration refinement can be added, plus bounded exposure findings
8. graph summary and bounded graph-aware scoring can be added when attributed graph coverage is meaningful
9. the output is rendered either as JSON or as an analyst-facing report

### What a reviewer should take away

- the repo is multi-chain, but not falsely chain-agnostic
- curated artifacts are a deliberate product surface, not throwaway fixtures
- Tier 1 attribution improves precision without pretending the repo already has full graph intelligence
- secondary corroboration improves analyst confidence without turning weak sources into hard risk jumps
- the actor/exposure layer adds real investigative nuance without claiming full graph reconstruction
- the graph layer is deliberately sampled, bounded, and coverage-gated
- report mode is a presentation layer on top of real scoring, not a mock demo veneer
- future behavior work is staged as an additive layer, not something the docs pretend already exists
- Optimism, Polygon, and Arbitrum Phase 1 are intentionally tx-only, cost-aware, and homelab-first

---

## Core Modules

### `internal/address`

Responsible for:

- chain syntax validation
- live balance and activity lookup
- initial `WalletProfile` construction

This module does not currently provide full multi-chain scoring on its own.

### `internal/analyzer`

Responsible for the shared live-scoring engine:

- watchlist short-circuit
- wallet-age signals
- velocity
- repeated flagged interaction
- count-based service concentration
- noisy inbound observations
- combination rules

Today this is the strongest live-scoring path for EVM.

### `internal/attribution`

Responsible for the attribution, actor/exposure, and bounded graph layer:

- loading Tier 1 structured label fixtures
- loading Bitcoin mining-pool context
- loading repo-local bootstrap overrides
- loading secondary corroborating sources
- resolving a final attribution decision
- applying controlled post-behavior score modifiers
- rolling repeated interaction or concentration up to actor level when attribution support is strong
- surfacing direct or near actor exposure summaries
- detecting bounded pass-through or U-turn patterns from sampled activity
- building graph summaries from attributed sampled flows
- deriving bounded graph motifs
- applying bounded graph-aware score modifiers when coverage thresholds are met

This package intentionally stops short of full cluster or graph resolution.

### `internal/datasets`

Responsible for:

- loading curated cases
- loading trace summaries
- loading chain-specific curated dataset cases
- building dataset-mode wallet profiles for curated EVM cases

### `cmd/validator/dataset_*_context.go`

Responsible for chain-specific dataset scoring adapters:

- `dataset_trace_context.go`
- `dataset_optimism_layer2_context.go`
- `dataset_polygon_layer2_context.go`
- `dataset_arbitrum_layer2_context.go`
- `dataset_erc20_layer1_context.go`
- `dataset_solana_stablecoin_context.go`
- `dataset_bitcoin_layer1_context.go`

This is an important architecture detail:

- EVM live scoring uses the shared analyzer
- Optimism, Polygon, Arbitrum, ERC-20, Solana, and Bitcoin currently score through dataset adapters
- Optimism, Polygon, and Arbitrum Phase 1 use tx-only adapters derived from exported Parquet summarized locally

### Report layer responsibilities

The report layer adds:

- case title and dataset context
- resolved attribution and source context
- concise risk-summary framing
- chain context lines
- actor-aware or hop-aware findings when they materially improve interpretation
- graph summary when attributed graph coverage is meaningful
- top-counterparty presentation for demos and portfolio review

It does not change the underlying scoring model.

---

## Chain Architecture Today

| Chain    | Current source path                                                                           | Current scoring path                                                                                                                                                                     | Current limit                                                                     |
|----------|-----------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------|
| Ethereum | Etherscan live txs, extracted EVM datasets, optional trace summaries                          | Shared analyzer, Tier 1 attribution, bounded secondary corroboration, actor/exposure refinement, trace-aware dataset context, bounded graph-aware refinement when coverage is meaningful | No trace-driven live scoring, no generalized graph scoring                        |
| Optimism | BigQuery Blockchain Analytics transactions export, local Parquet summarization, curated cases | Dataset-mode tx-only Layer 2 scoring adapter plus attribution hierarchy                                                                                                                  | Decoded-events deferred in Phase 1 due scan-cost / ROI                            |
| Polygon  | BigQuery Blockchain Analytics transactions export, local Parquet summarization, curated cases | Dataset-mode tx-only Layer 2 scoring adapter plus attribution hierarchy                                                                                                                  | Decoded-events deferred in Phase 1 due scan-cost / ROI                            |
| Arbitrum | BigQuery Blockchain Analytics transactions export, local Parquet summarization, curated cases | Dataset-mode tx-only Layer 2 scoring adapter plus attribution hierarchy                                                                                                                  | Decoded-events deferred in Phase 1 due scan-cost / ROI                            |
| ERC-20   | Local Blockchair ERC-20 shards, latest token metadata snapshot, curated ERC-20 cases          | Dataset-mode ERC-20 scoring adapter plus attribution hierarchy, actor/exposure refinement, and bounded graph-summary output when coverage allows                                         | No live token scoring, no swap-aware decoding, no generalized token graph scoring |
| Solana   | Local stablecoin-flow Parquet exports and curated cases                                       | Dataset-mode stablecoin scoring adapter plus attribution hierarchy                                                                                                                       | No general instruction-aware live scoring                                         |
| Bitcoin  | Local Blockchair inputs/outputs and curated cases                                             | Dataset-mode UTXO-flow scoring adapter plus attribution hierarchy, bounded cluster-aware interpretation, and bounded graph-summary output when coverage allows                           | No generalized cluster graph scoring                                              |

---

## Ethereum Architecture

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
7. attribution, actor/exposure refinement, and bounded graph scoring can refine the final profile when supported

### Why traces are separate

The repo uses traces today to enrich explanation without overstating trace-native scoring maturity.

That keeps the implementation honest:

- trace extraction is real
- trace-aware explanation is real
- trace-native heuristics remain future work

---

## Optimism Layer 2 Architecture (Phase 1)

Optimism Phase 1 is intentionally transactions-only.

### Current Optimism flow

1. candidate mining runs against Google Cloud Blockchain Analytics / BigQuery over the repo’s canonical 90-day window
2. shortlisted transactions are exported to Parquet
3. exported Parquet is transferred to the homelab
4. local summarization derives:
   - tx counts
   - inbound / outbound split
   - unique counterparties
   - top counterparties
   - dominant destination contract share
   - dominant function-selector share
5. curated Optimism cases are written under `data/cases/curated-optimism/`
6. `cmd/validator --dataset` applies Optimism-specific dataset scoring
7. report mode renders Optimism cases using the same shared analyst-facing output contract

### Why Phase 1 is tx-only

Decoded-events were tested during implementation, but the scan-cost / ROI tradeoff was not strong enough for the first Optimism pass.

So Phase 1 deliberately favors:

- cheaper and more repeatable export workflow
- homelab-first summarization
- contract-destination and selector-based interpretation
- portfolio-safe, reproducible dataset-mode cases

### What Phase 1 proves

Even without decoded-events, the current Optimism implementation can already show:

- repeated-contract router-like behavior
- broad operational hub behavior
- very high transaction concentration to one contract
- broad mixed-flow surface across many counterparties
- a practical Layer 2 dataset-mode workflow built under cost constraints

---

## Polygon Layer 2 Architecture (Phase 1)

Polygon Phase 1 is intentionally transactions-only.

### Current Polygon flow

1. candidate mining runs against Google Cloud Blockchain Analytics / BigQuery over the repo’s canonical 90-day window
2. shortlisted transactions are exported to Parquet
3. exported Parquet is transferred to the homelab
4. local summarization derives:
   - tx counts
   - inbound / outbound split
   - unique counterparties
   - top counterparties
   - dominant destination contract share
5. curated Polygon cases are written under `data/cases/curated-polygon/`
6. `cmd/validator --dataset` applies Polygon-specific dataset scoring
7. report mode renders Polygon cases using the same shared analyst-facing output contract

### Why Phase 1 is tx-only

Decoded-events were tested during implementation, but the scan-cost / ROI tradeoff was not strong enough for the first Polygon pass.

So Phase 1 deliberately favors:

- cheaper and more repeatable export workflow
- homelab-first summarization
- contract-destination-based interpretation
- portfolio-safe, reproducible dataset-mode cases

### What Phase 1 proves

Even without decoded-events, the current Polygon implementation can already show:

- repeated-contract service-like behavior
- broad operational hub behavior
- very high transaction concentration to one contract
- broad mixed-flow surface across many counterparties
- a practical Layer 2 dataset-mode workflow built under cost constraints

---

## Arbitrum Layer 2 Architecture (Phase 1)

Arbitrum Phase 1 is intentionally transactions-only.

### Current Arbitrum flow

1. candidate mining runs against Google Cloud Blockchain Analytics / BigQuery over the repo’s canonical 90-day window
2. shortlisted transactions are exported to Parquet
3. exported Parquet is transferred to the homelab
4. local summarization derives:
   - tx counts
   - inbound / outbound split
   - unique counterparties
   - top counterparties
   - dominant destination contract share
5. curated Arbitrum cases are written under `data/cases/curated-arbitrum/`
6. `cmd/validator --dataset` applies Arbitrum-specific dataset scoring
7. report mode renders Arbitrum cases using the same shared analyst-facing output contract

### Why Phase 1 is tx-only

Decoded-events were tested during implementation, but the scan-cost / ROI tradeoff was not strong enough for the first Arbitrum pass.

So Phase 1 deliberately favors:

- cheaper and more repeatable export workflow
- homelab-first summarization
- contract-destination-based interpretation
- portfolio-safe, reproducible dataset-mode cases

### What Phase 1 proves

Even without decoded-events, the current Arbitrum implementation can already show:

- repeated-contract service-like behavior
- broad operational hub behavior
- very high transaction concentration to one contract
- broad mixed-flow surface across many counterparties
- a practical Layer 2 dataset-mode workflow built under cost constraints

## L2 Phase 2 Semantic Enrichment Architecture (Scaffolded)

The next L2 layer is designed as a bounded semantic enrichment pass rather than a broad decoded-events-first warehouse.

Prepared now:

- shared bridge / protocol / stablecoin / topic registries
- shared Phase 2 extract schema
- registry-hit enrichment helper
- chain-specific Phase 2 merge scripts for Optimism, Polygon, and Arbitrum

The intended Phase 2 architecture is:

1. keep the current tx-only Phase 1 extracts as the base
2. merge per-address receipt summaries
3. merge per-address log emitter and topic0 summaries
4. enrich with registry hits
5. refresh curated L2 cases with stronger semantic evidence

This architecture is intentionally cost-aware:

- it preserves the current homelab-first workflow
- it improves semantic precision without requiring broad decoded-events exports as the default path
- it keeps decoded-events as a targeted enhancement path rather than a mandatory dependency

## L2 Phase 3 Cross-Chain Mart Architecture (Scaffolded)

Phase 3 is designed as a normalization layer above the three L2 Phase 2 outputs.

Prepared now:

- cross-chain feature mart builder
- curated cross-chain case generator
- cross-chain mart schema
- curated cross-chain case schema
- validator routing and report scaffolding for cross-chain L2 cases

The intended Phase 3 architecture is:

1. read populated Phase 2 extracts from Optimism, Polygon, and Arbitrum
2. normalize them into one cross-chain feature mart
3. derive cross-chain candidate patterns
4. generate curated cross-chain benchmark cases
5. route those through `cmd/validator --dataset`
6. render cross-chain analyst reports with member-level summaries

This keeps cross-chain reasoning additive and explainable rather than pretending the repo already has a generalized live multi-chain graph engine.
---

## Solana Architecture

Solana is currently stablecoin-flow first.

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

It is a practical stablecoin-flow slice.

---

## ERC-20 Architecture

ERC-20 is transfer-event first.

### Current ERC-20 flow

1. local Blockchair ERC-20 shards and token metadata stay outside git
2. `scripts/mine_erc20_candidates.py` identifies behavior-driven ERC-20 candidates from the canonical window
3. `scripts/extract_erc20_layer1.py` builds address-scoped summaries plus compressed raw subsets
4. `scripts/curate_erc20_layer1.py` creates curated ERC-20 benchmark cases
5. `cmd/validator --dataset` applies ERC-20-specific scoring
6. attribution, actor/exposure refinement, and bounded graph-summary output can refine the final case when attributed coverage is sufficient

### Current ERC-20 responsibilities

- token-aware inbound vs outbound role summaries
- broad token counterparty surface scoring
- repeated counterparty interaction scoring
- token diversity and single-token concentration observation
- trusted protocol or exchange-style contextual reasoning when Tier 1 attribution exists
- bounded actor-aware refinement when counterparties resolve to the same actor
- bounded graph-summary output when attributed graph coverage is meaningful

### Current ERC-20 limitation

This is ERC-20 transfer-row scoring, not full DeFi intent decoding.

It does not yet reconstruct swaps, decode protocol semantics, or use traces for full path scoring.

---

## Bitcoin Architecture

Bitcoin is currently UTXO-flow first.

### Current Bitcoin flow

1. local Blockchair inputs/outputs live outside git
2. `scripts/mine_bitcoin_candidates.py` identifies candidate addresses
3. `scripts/extract_bitcoin_layer1.py` builds address-scoped UTXO summaries
4. `scripts/curate_bitcoin_layer1.py` creates curated Bitcoin cases
5. `cmd/validator --dataset` applies Bitcoin-specific UTXO-flow scoring
6. attribution, actor/exposure refinement, and bounded graph-summary output can refine the final case when attributed coverage is sufficient

### Current Bitcoin responsibilities

- inbound-receipt vs outbound-spend role analysis
- broad counterparty surface scoring
- operational hub scoring
- repeated counterparty interaction scoring
- contextual service interpretation via bounded attribution support
- bounded graph-summary output when attributed graph coverage is meaningful

### Current Bitcoin limitation

This is address-level UTXO profiling, not generalized cluster-aware wallet analytics.

---

## Watchlist, Attribution, and Graph Layer

The current entity layer has three active inputs:

- watchlist engine for sanctions checks
- Tier 1 structured attribution inputs
- repo-local bootstrap overrides for continuity and demos

Tier 1 currently means:

- GraphSense-style structured labels
- Bitcoin mining-pool context
- repo-local bootstrap labels

Secondary corroboration adds:

- WalletExplorer-style secondary source support
- repo-safe corroborating fixture inputs
- supporting and conflicting source visibility

The current actor/exposure and graph layer adds:

- actor-aware repeated-interaction and concentration refinement
- practical direct and near exposure summaries
- bounded pass-through and U-turn findings tied to attributed actors
- graph summaries from attributed sampled flows
- bounded graph motifs
- bounded graph-aware score adjustments

The important architectural choice is ordering:

1. behavior is scored first
2. Tier 1 attribution is resolved second
3. secondary corroboration can raise confidence or register a conflict
4. actor/exposure refinement is applied when attribution support is strong enough
5. graph summary and bounded graph-aware scoring are applied when attributed graph coverage is meaningful
6. the report surfaces the resolved attribution and graph context concisely

This keeps the system useful for investigations without pretending the repo already has full cluster-aware entity resolution.

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
- `attribution` when available
- `attribution_insights` when actor-aware or exposure-aware refinement adds value
- `graph_summary` when attributed graph coverage is meaningful enough to render it safely

This shared output contract is why the repo can mix:

- one shared analyzer
- several chain-specific dataset adapters
- a common attribution resolver
- a bounded graph-aware layer

without losing coherence for the user.

On top of that contract, report mode adds a human-readable presentation layer without changing the JSON schema.

### Why that matters for portfolio review

This is the architectural choice that keeps the project legible:

- engineers can inspect JSON, tests, and chain-specific adapters
- analysts can read the same result in brief form with `--report`
- recruiters and hiring managers can follow the system without first reading internal code

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

The architecture is set up so the next stage can add:

- populated L2 Phase 2 receipt/log semantic summaries
- refreshed Optimism, Polygon, and Arbitrum curated cases using semantic enrichment
- the first populated cross-chain L2 feature mart
- the first curated cross-chain L2 benchmark cases
- richer value-weighted graph scoring
- broader graph coverage where attributed graph support is currently sparse
- fresh-wallet plus immediate large-flow reasoning
- richer Solana and Bitcoin live-path reasoning
- broader protocol-intent interpretation for ERC-20 and trace-enriched EVM paths
- targeted decoded-events enhancement only where Phase 2 semantic coverage is still too ambiguous

Those are next-stage additions, not claims about the current implementation.

## Review Path

For the fastest architecture review:

1. read the portfolio snapshot in [`README.md`](README.md)
2. scan the Mermaid diagrams in this file
3. run one curated report command from [`docs/DEMO-WALKTHROUGH.md`](docs/DEMO-WALKTHROUGH.md)
4. compare the output with the static examples in [`docs/sample-reports/README.md`](docs/sample-reports/README.md)

---

## Related Documents

- [`README.md`](README.md)
- [`docs/TYPOLOGIES.md`](docs/TYPOLOGIES.md)
- [`docs/SCORING.md`](docs/SCORING.md)
- [`docs/EVM-CALLS-INTEGRATION.md`](docs/EVM-CALLS-INTEGRATION.md)
- [`docs/LABEL-SOURCE-HIERARCHY.md`](docs/LABEL-SOURCE-HIERARCHY.md)
- [`docs/ERC20-DATA-MODEL.md`](docs/ERC20-DATA-MODEL.md)
- [`docs/SOLANA-DATA-MODEL.md`](docs/SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](docs/BITCOIN-DATA-MODEL.md)
