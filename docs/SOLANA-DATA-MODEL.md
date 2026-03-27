# Crypto Profiler Solana Data Model

## Purpose

This document describes the Solana model that is actually implemented in the repository today.

Wave 1 clarification:

- the current Solana Layer 1 is stablecoin-flow based
- it is delivered through curated dataset mode
- it is not yet a general RPC-hydrated, instruction-aware Solana profiling engine

---

## Current Scope

The current Solana path is built around large-value USDC/USDT transfer flow summaries.

Implemented components:

- candidate mining from local stablecoin-flow Parquet exports
- address-scoped extracted stablecoin summaries
- curated Solana Layer 1 case artifacts
- validator dataset-mode scoring

Current repo paths:

- `scripts/mine_solana_whale_candidates.py`
- `scripts/extract_solana_stablecoin.py`
- `scripts/curate_solana_stablecoin.py`
- `data/cases/extracted-solana-stablecoin/`
- `data/cases/curated-solana/`

---

## Source Model

The Solana Layer 1 slice currently depends on historical whale-flow style stablecoin exports collected outside git.

Tracked roles in the current extractor:

- `source`
- `destination`
- `authority`

Tracked stablecoins in the current extractor:

- USDC
- USDT

This means the current Solana model is strongest at describing:

- who is sending large stablecoin transfers
- who is receiving them
- which authority is controlling the movement
- how broad the stablecoin counterparty surface is

---

## Extracted Summary Shape

The extracted Solana summary contains the fields that the validator actually scores today.

Core summary fields:

- `first_seen`
- `last_seen`
- `source_transfer_count`
- `destination_transfer_count`
- `authority_transfer_count`
- `source_value_raw`
- `destination_value_raw`
- `authority_value_raw`
- `unique_counterparties`
- `source_counterparties`
- `destination_counterparties`
- `authority_counterparties`
- `dominant_role`
- `dominant_mint`

Supporting case fields:

- `mint_breakdown`
- `transfer_type_breakdown`
- `top_counterparties`
- `top_authority_pairs`
- `sample_transfers`

The corresponding Go types live in `internal/datasets/solana_curated.go`.

---

## Current Implemented Scoring Signals

Validator dataset mode currently scores these Solana patterns:

### Source-heavy stablecoin distributor

This captures very large outbound stablecoin distribution from a source-dominant address.

Current reason code:

- `solana_source_heavy_stablecoin_distributor`

### Authority-heavy stablecoin operator

This captures authority-dominant behavior that looks operational and reviewable.

Current reason code:

- `solana_authority_heavy_stablecoin_operator`

### Broad stablecoin surface

This captures a large and potentially noisy stablecoin counterparty footprint.

Current reason codes:

- `solana_broad_mixed_stablecoin_surface`
- `solana_broad_stablecoin_counterparty_surface`

### Mixed stablecoin activity

This captures activity across multiple major stablecoin mints.

Current reason code:

- `solana_mixed_stablecoin_activity`

### Repeated large counterparty interaction

This captures heavy repeated interaction with a dominant counterparty.

Current reason code:

- `solana_repeated_large_counterparty_interaction`

For exact weighting and score composition, see [`docs/SCORING.md`](SCORING.md).

---

## What Solana Layer 1 Means In This Repo

In the current repository, "Solana Layer 1" means:

- address-scoped stablecoin-flow summaries
- not full instruction decoding
- not full program semantics
- not general wallet-wide Solana history

That is a deliberate scope choice for the current MVP.

It keeps the Solana implementation:

- practical
- dataset-backed
- explainable
- cheap enough to iterate on

---

## Validator Dataset Support

Validator dataset mode currently supports curated Solana files such as:

- `data/cases/curated-solana/solana-usdc-distributor-treasury-like.json`
- `data/cases/curated-solana/solana-stablecoin-authority-operator.json`
- `data/cases/curated-solana/solana-broad-surface-authority-mixed-stablecoin.json`

Example:

```bash
go run ./cmd/validator --dataset ./data/cases/curated-solana/solana-stablecoin-authority-operator.json
```

This is the implemented Solana scoring path today.

---

## Current Limitations

The current Solana model does not yet support:

- general instruction-aware profiling
- full program-aware behavioral scoring
- non-stablecoin Layer 1 coverage
- graph-aware exposure
- bridge-aware analysis
- live Solana Layer 1 scoring from extracted local history

The live Solana strategy in `internal/address/solana.go` is currently a basic validation and activity lookup path, not the same thing as the stablecoin-flow Layer 1 model.

---

## Wave 2 and Beyond

Logical follow-on work after Wave 1:

- expand beyond stablecoin-only Solana coverage
- add richer program-aware and instruction-aware summaries
- add graph-aware exposure work
- add more direct integration between extracted Solana summaries and broader scoring logic

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/DATA-SOURCING-POLICY.md`](DATA-SOURCING-POLICY.md)
