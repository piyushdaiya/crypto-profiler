# Crypto Profiler ERC-20 Data Model

## Purpose

This document defines the ERC-20 Layer 1 model that is implemented in the repository today.

It explains:

- which ERC-20 datasets are used
- how transfer rows and token metadata are normalized
- what the extractor emits for each tracked address
- what validator dataset mode can score today
- what remains future work

This document is intentionally practical and implementation-aware.

---

## Current Status

ERC-20 Layer 1 is now implemented as a curated dataset-mode path.

Implemented today:

- behavior-driven ERC-20 candidate mining from local Blockchair transfer shards
- address-scoped ERC-20 extraction with compressed raw subsets and summary JSON
- curated ERC-20 benchmark cases under `data/cases/curated-erc20/`
- validator dataset-mode scoring for ERC-20 curated cases

Not implemented yet:

- live ERC-20 scoring inside the EVM address strategy
- swap-aware or router-intent decoding
- trace-aware ERC-20 pass-through or U-turn detection
- graph-aware 1-hop or 2-hop token exposure

---

## Canonical Window

The current ERC-20 Layer 1 window is:

- `2025-03-16 -> 2025-06-17`

This matches the multi-chain MVP window used across:

- Ethereum native / trace-backed case work
- Bitcoin UTXO Layer 1
- ERC-20 Layer 1

Token metadata is not historical in the same way. For ERC-20 metadata, the repo uses the latest available token snapshot as a reference lookup.

---

## Source Datasets

Crypto Profiler uses two local Blockchair ERC-20 inputs.

### 1. ERC-20 transfer shards

Representative fields:

- `transaction_hash`
- `time`
- `token_address`
- `token_name`
- `token_symbol`
- `token_decimals`
- `sender`
- `recipient`
- `value`

This is the behavioral layer.

It is used for:

- candidate mining
- address-scoped extraction
- inbound vs outbound token-flow summaries
- counterparty breadth and repeated-interaction analysis
- curated ERC-20 cases

### 2. Latest token metadata snapshot

Representative fields:

- `address`
- `name`
- `symbol`
- `decimals`

This is the enrichment layer.

It is used for:

- token name / symbol normalization
- decimals-aware display values
- readable token breakdowns in extracted and curated artifacts

---

## Normalization Rules

### Address normalization

Blockchair ERC-20 data uses lowercase hex addresses without `0x`.

Crypto Profiler normalizes:

- tracked wallet addresses
- `sender`
- `recipient`
- `token_address`

to:

- lowercase
- trimmed
- `0x`-prefixed form

### Metadata tolerance

The extractor tolerates:

- missing token names
- missing token symbols
- missing or malformed decimals
- incomplete token metadata coverage

When metadata is missing, the raw transfer row is still preserved.

### Value semantics

The `value` field is the raw integer amount in base units.

The extractor preserves:

- `value_raw`
- `token_decimals`
- a display-friendly normalized value for analyst-facing samples

The raw value remains authoritative.

---

## Extraction Output Shape

`scripts/extract_erc20_layer1.py` creates two artifacts per tracked address.

### Raw subset artifact

Compressed NDJSON:

- `<address>.erc20.ndjson.gz`

This contains sampled raw transfer rows for that tracked address only.

### Summary artifact

JSON:

- `<address>.json`

Current summary fields include:

- `first_seen`
- `last_seen`
- `inbound_transfer_count`
- `outbound_transfer_count`
- `self_transfer_count`
- inbound / outbound / total unique counterparties
- unique token contract count
- repeated counterparty count
- max counterparty interactions
- dominant direction
- dominant token and dominant token transfer share

The summary also includes:

- `token_breakdown`
- `top_counterparties`
- `sample_transfers`
- `raw_subset_file`

This is the main Layer 1 extraction contract used by ERC-20 curation and validator dataset mode.

---

## Candidate Mining

`scripts/mine_erc20_candidates.py` is the repo entry point for behavior-driven ERC-20 candidate discovery.

It currently mines for addresses with signals such as:

- high overall ERC-20 activity
- broad counterparty surface
- high token diversity
- repeated interaction concentration
- inbound-heavy token surfaces

The committed reference file is:

- `data/candidates/erc20_addresses.example.txt`

Local working lists stay out of git:

- `data/candidates/*.local.txt`

---

## Curated ERC-20 Cases

Curated ERC-20 cases live under:

- `data/cases/curated-erc20/`

Each curated case carries:

- case identity and narrative
- risk posture
- extracted ERC-20 summary
- token breakdown
- top counterparties
- sample transfers
- curation notes and limitations

These curated artifacts are the current delivery surface for ERC-20 Layer 1 scoring in demos and validator dataset mode.

---

## What Validator Dataset Mode Scores Today

The ERC-20 dataset-mode path currently scores practical Layer 1 rule families built from `erc20_summary`, `token_breakdown`, and `top_counterparties`.

Implemented today:

- trusted protocol token hubs
- exchange-style service surfaces when labels exist
- noisy inbound token surfaces
- broad token counterparty surfaces
- repeated counterparty activity
- mixed token activity
- dominant single-token concentration

This is transfer-row scoring, not protocol semantic decoding.

---

## What ERC-20 Layer 1 Covers Today

Current ERC-20 Layer 1 is good at:

- token-aware inbound vs outbound profiling
- measuring breadth across counterparties and tokens
- highlighting repeated ERC-20 interaction structure
- surfacing token concentration and dominant assets
- producing curated benchmark cases with narrative and sample evidence

---

## What ERC-20 Layer 1 Does Not Cover Yet

Current ERC-20 Layer 1 does not yet do:

- swap path reconstruction
- protocol-intent decoding
- bridge attribution
- trace-aware pass-through or U-turn scoring
- 1-hop or 2-hop exposure summaries
- graph-aware clustering or entity merging

Those are separate next-step layers on top of the current transfer-row foundation.

---

## Dataset Mode vs Future Live or Graph-Aware Scoring

### Dataset mode today

ERC-20 currently scores from curated JSON artifacts that were built offline from local historical shards.

That makes it useful for:

- reproducible demos
- benchmark cases
- validator dataset-mode screenshots
- rule-family development

### Future live or graph-aware scoring

Future ERC-20 work is expected to add:

- live token-aware scoring in the EVM address path
- trace-aware contract-mediated interpretation
- pass-through and U-turn detection
- 1-hop / 2-hop exposure summaries
- richer graph-aware reasoning

---

## Relationship to EVM Calls

ERC-20 transfer rows are useful on their own, but they do not explain execution intent.

Ethereum calls and traces matter later for:

- internal value movement context
- router-mediated transfers
- contract-mediated pass-through behavior
- distinguishing genuine protocol routing from suspicious round trips

That relationship is documented separately in:

- [`docs/EVM-CALLS-INTEGRATION.md`](EVM-CALLS-INTEGRATION.md)

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/ETHEREUM-DATA-MODEL.md`](ETHEREUM-DATA-MODEL.md)
- [`docs/DATA-SOURCING-POLICY.md`](DATA-SOURCING-POLICY.md)
