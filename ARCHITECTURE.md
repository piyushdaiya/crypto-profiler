# Crypto Profiler Architecture

## Purpose

Crypto Profiler is a Go-based wallet intelligence platform for AML, fraud, sanctions, and crypto-risk analysis.

The architecture is designed to support:

- deterministic wallet validation
- exact-match watchlist screening
- blockchain activity extraction and normalization
- graph-aware exposure reasoning
- explainable scoring
- investigator-friendly outputs
- stable dataset-backed demos for portfolio and product storytelling

The current MVP is intentionally focused on:

- Bitcoin
- Ethereum / EVM
- selected ERC-20 activity
- curated case datasets
- JSON and CLI outputs

---

## High-Level Architecture

```text
                    +----------------------+
                    |   User / Analyst     |
                    | CLI / API / Future UI|
                    +----------+-----------+
                               |
                               v
                    +----------------------+
                    |   API / Execution     |
                    | validator / future API|
                    +----------+-----------+
                               |
                +--------------+--------------+
                |                             |
                v                             v
    +------------------------+   +--------------------------+
    | Live Chain Providers   |   | Curated Dataset Loader   |
    | Etherscan / CoinStats  |   | Extracted JSON case sets |
    +-----------+------------+   +------------+-------------+
                |                             |
                +--------------+--------------+
                               |
                               v
                    +----------------------+
                    | Normalization Layer   |
                    | addresses / transfers |
                    | token metadata        |
                    +----------+-----------+
                               |
                               v
                    +----------------------+
                    | Entity / Watchlist    |
                    | labels / scams / OFAC |
                    | watchlist engine      |
                    +----------+-----------+
                               |
                               v
                    +----------------------+
                    | Graph / Exposure      |
                    | counterparties / hops |
                    | transaction patterns  |
                    +----------+-----------+
                               |
                               v
                    +----------------------+
                    | Scoring Engine        |
                    | deterministic +       |
                    | heuristic + combo     |
                    +----------+-----------+
                               |
                               v
                    +----------------------+
                    | Explanation Layer     |
                    | reasons / evidence /  |
                    | review recommendation |
                    +----------+-----------+
                               |
                               v
                    +----------------------+
                    | JSON / CLI Output     |
                    | case-study artifacts  |
                    +----------------------+
```

---

## 1. Ingestion Layer

The ingestion layer is responsible for acquiring raw blockchain and reference data.

### Current inputs

#### Live provider inputs
- Ethereum / EVM transaction state from provider APIs
- wallet activity metadata used by the validator in live mode

#### Watchlist inputs
- OFAC / sanctions source data
- internal bootstrap entity labels
- legacy labels and scam reference files

#### File-based blockchain inputs
- Blockchair Bitcoin transactions
- Blockchair Bitcoin inputs
- Blockchair Bitcoin outputs
- Blockchair Ethereum transactions
- Blockchair Ethereum calls
- Blockchair ERC-20 transactions
- Blockchair ERC-20 token metadata snapshot

### Current implementation style
- local file-based ingestion for raw dump processing
- Dockerized watchlist-engine sync for sanctions data
- provider-backed live profiling for selected chains

### Design intent
The ingestion layer should remain replaceable.

Today it is:
- file-based
- API-assisted
- watchlist-engine-backed

Later it can evolve into:
- scheduled batch ingestion
- persistent graph storage
- streaming or incremental updates

---

## 2. Normalization Layer

The normalization layer transforms heterogeneous chain/provider data into common internal shapes.

### Responsibilities
- lowercase / normalize addresses where appropriate
- validate chain-specific syntax
- normalize transfers into common structures
- attach token metadata where available
- preserve timestamps, hashes, counterparties, and value fields
- convert curated datasets into internal wallet-profile inputs

### Current normalized objects
- `WalletProfile`
- `Transaction`
- `Transfer`
- `AddressDataset`
- `CuratedCase`

### Why this matters
Without normalization, explainable scoring becomes inconsistent across:
- Bitcoin vs EVM
- live API vs extracted dataset
- native asset vs ERC-20 transfer flows

Normalization is the layer that keeps the analyzer independent from raw source quirks.

---

## 3. Entity / Watchlist Layer

The entity layer provides identity and risk context for wallets and counterparties.

### Responsibilities
- exact-match label lookup
- sanctions lookup through watchlist engine
- support for trusted labels
- support for high-risk labels
- support for protocol and exchange labels
- support for scams / legacy labels / future watchlist feeds

### Current sources
- bootstrap entity labels JSON
- legacy labels JSON
- scams JSON
- watchlist engine backed by sanctions data

### Current label categories
- `SANCTIONS`
- `MIXER`
- `EXPLOIT`
- `SCAM`
- `EXCHANGE`
- `PROTOCOL`
- `TRUSTED`

### Design principle
This layer should answer:
- what is this wallet?
- is it sanctioned?
- is it trusted?
- is it known high-risk?
- what label confidence and severity should be attached?

This is intentionally separate from the scoring engine so that:
- label coverage can evolve independently
- additional watchlists can be plugged in later
- multiple downstream components can reuse the same entity context

---

## 4. Graph / Exposure Engine

The graph / exposure engine is the reasoning layer that interprets wallet relationships and transaction structure.

### Responsibilities
- direct counterparty analysis
- 1-hop and 2-hop exposure logic
- high fan-in / fan-out observations
- transaction sequence interpretation
- curated counterparties and interaction summaries
- future graph traversal across larger datasets

### Current state
The MVP currently supports:
- direct counterparty reasoning
- curated top-counterparty summaries
- transaction-sample inspection
- labeled interaction detection

### Planned evolution
The next steps for this layer include:
- explicit 1-hop / 2-hop exposure search
- Bitcoin fund-flow reconstruction from inputs + outputs
- Ethereum call-aware internal flow reasoning
- mixer proximity patterns
- pass-through wallet detection
- peeling-chain style layering detection

### Why this matters
Risk is rarely contained in a single wallet field.
It often emerges from:
- who a wallet interacts with
- how often it interacts
- how value moves across entities
- how quickly funds arrive and leave

The graph / exposure engine is what turns raw transfers into intelligence.

---

## 5. Scoring Engine

The scoring engine converts deterministic hits and heuristic observations into explainable risk.

### Responsibilities
- evaluate deterministic watchlist outcomes
- evaluate profile labels
- evaluate counterparty labels
- evaluate behavioral heuristics
- apply combination rules
- generate category-level and overall score
- map score to grade and review guidance

### Current scoring model
The MVP scoring engine supports:
- sanctions short-circuit
- profiled address label detection
- direct mixer interaction
- trusted/protocol mitigation
- established history mitigation
- fresh wallet escalation
- high velocity escalation
- noisy inbound / high fan-in / zero-value inbound observations
- review recommendation logic
- combination rules for correlated signals

### Output concepts
- `risk_score`
- `risk_grade`
- `review_recommended`
- `risk_breakdown`
- `risk_reasons`

### Design principle
A single signal should not always produce a failing outcome.

The engine is intentionally designed around:
- deterministic controls where required
- multi-signal correlation for stronger conclusions
- contextual mitigation to reduce false positives
- explainability over opaque scoring

---

## 6. Explanation Layer

The explanation layer translates engine behavior into analyst-readable evidence.

### Responsibilities
- attach reason codes
- provide human-readable descriptions
- preserve source metadata
- preserve related entity and address context
- preserve severity / confidence where known
- distinguish between observed, reviewable, and critical cases

### Current evidence format
Each reason can include:
- code
- category
- description
- offset
- source
- related entity
- related address
- severity
- confidence
- evidence count

### Why this matters
Explainability is central to:
- analyst trust
- compliance defensibility
- product credibility
- portfolio presentation

The explanation layer is what makes the platform interpretable rather than just functional.

---

## 7. API / UI Layer

The API / UI layer is how users and systems interact with Crypto Profiler.

### Current interfaces
#### CLI
- live profiling by wallet address
- dataset-backed profiling by curated case file
- JSON output suitable for demos and automation

#### Watchlist engine
- internal HTTP `/check` path for sanctions lookup
- Dockerized local service execution

### Future interfaces
- lightweight HTTP API for validator/profile execution
- analyst review endpoints
- future web UI / investigation console
- export/report workflows

### Why this layer matters
A good intelligence engine needs a usable interaction model.

The current CLI-first approach is intentional:
- fast to build
- easy to test
- easy to demonstrate
- good for portfolio storytelling

The design leaves room for an eventual service/API/UI without forcing that complexity into the MVP.

---

## 8. Data Modes

Crypto Profiler currently supports two important operating modes.

### Live mode
Used for:
- provider-backed wallet profiling
- current watchlist-engine interaction
- realistic end-to-end demos

### Dataset-backed mode
Used for:
- reproducible portfolio demos
- stable case-study walkthroughs
- testing without live API drift
- curated examples of public, trusted, risky, and sanctioned wallets

This split is important because:
- live mode proves integration
- dataset mode proves repeatable product behavior

---

## 9. Current Repository Structure

```text
cmd/
  profiler/
  validator/
  extractcases/
  curatecases/

internal/
  address/
  analyzer/
  datasets/
  model/
  watchlist/

data/
  candidates/
  cases/
    curated/
    extracted/
  labels/

docs/
  case-studies/
  security/
```

### Structure intent
- `address` owns chain validation and fetch strategies
- `analyzer` owns scoring and reasoning
- `datasets` owns extraction and curation logic
- `watchlist` owns watchlist-engine client behavior
- `model` owns shared internal data structures

---

## 10. MVP Hosting Shape

The current MVP can be hosted with a lightweight deployment model:

- one small Go service container
- one watchlist / sanctions service container
- SQLite-backed local storage
- optional object/file storage for curated case artifacts

This is intentionally small enough for:
- local Docker
- a single low-cost VM
- small managed container platforms

The hosted MVP does not require a full microservice environment.

---

## 11. Near-Term Roadmap

### Completed
- wallet validation
- sanctions short-circuiting
- label-aware scoring
- curated dataset generation
- dataset-backed validator mode
- unit tests
- OWASP Phase 1 baseline

### Next
- Bitcoin outputs integration
- Ethereum calls integration
- ERC-20 token metadata enrichment
- 1-hop / 2-hop exposure logic
- dusting / noisy inbound refinement
- improved README and architecture diagrams
- low-cost hosted MVP deployment

---

## 12. Guiding Principle

Crypto Profiler is being built as a practical, explainable crypto-risk platform.

The architecture intentionally favors:
- clarity over cleverness
- modularity over premature scale
- deterministic evidence over black-box claims
- portfolio-grade storytelling backed by working code and reproducible data
