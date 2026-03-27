# Crypto Profiler Bitcoin Data Model

## Purpose

This document describes the Bitcoin model that is actually implemented in the repository today.

Wave 1 clarification:

- the current Bitcoin Layer 1 is address-scoped and UTXO-flow based
- it is delivered through curated dataset mode
- it is not yet a cluster-aware Bitcoin graph engine

---

## Current Scope

The current Bitcoin path is built from local Blockchair input/output data and reduced into address-scoped summaries.

Implemented components:

- candidate mining from local Bitcoin TSV.gz inputs/outputs
- address-scoped extracted Bitcoin Layer 1 summaries
- curated Bitcoin Layer 1 case artifacts
- validator dataset-mode scoring

Current repo paths:

- `scripts/mine_bitcoin_candidates.py`
- `scripts/extract_bitcoin_layer1.py`
- `scripts/curate_bitcoin_layer1.py`
- `data/cases/extracted-bitcoin/`
- `data/cases/curated-bitcoin/`

---

## Source Model

The current Bitcoin extractor uses:

- output rows where `recipient == tracked address` for inbound receipts
- input rows where `recipient == tracked address` and a spending transaction exists for outbound spends

It also derives approximate counterparty context from linked transaction recipients.

This makes the current Bitcoin model strongest at:

- inbound vs outbound role analysis
- broad counterparty surface summaries
- repeated counterparty interaction
- operational hub-style behavior

---

## Current Time Window

The current implemented extractor and curated fixtures are aligned to the practical Bitcoin window used in the repo:

- `2025-03-16` to `2025-06-17`

This is the current reference slice for Bitcoin Layer 1 artifacts.

---

## Extracted Summary Shape

The extracted Bitcoin summary contains the fields that the validator actually scores today.

Core summary fields:

- `first_seen`
- `last_seen`
- `inbound_receipt_count`
- `outbound_spend_count`
- `inbound_value_sats`
- `outbound_value_sats`
- `inbound_value_btc`
- `outbound_value_btc`
- `unique_counterparties`
- `counterparty_resolution_mode`
- `dominant_role`

Supporting case fields:

- `top_counterparties`
- `sample_events`

The corresponding Go types live in `internal/datasets/bitcoin_curated.go`.

---

## What Bitcoin Layer 1 Means In This Repo

In the current repository, "Bitcoin Layer 1" means:

- address-level UTXO receipt and spend behavior
- broad-surface counterparty summaries
- not wallet clustering
- not change detection
- not full path reconstruction

That means the model is useful for:

- operational behavior summaries
- repeat interaction analysis
- benchmark case creation
- explainable dataset-mode scoring

But it should not be overstated as a full Bitcoin attribution system.

---

## Current Implemented Scoring Signals

Validator dataset mode currently scores these Bitcoin patterns:

### Spend-heavy operational hub

Outbound-dominant, high-volume, broad-surface behavior.

Current reason code:

- `bitcoin_spend_heavy_operational_hub`

### Noisy inbound broad surface

Receive-heavy behavior with a very broad counterparty surface.

Current reason code:

- `bitcoin_noisy_inbound_broad_surface`

### Mixed-flow broad-value legacy wallet

Large inbound and outbound value movement on a legacy-format address with very broad surface area.

Current reason code:

- `bitcoin_legacy_mixed_flow_broad_value`

### Balanced high-volume hub

Large balanced inbound and outbound activity.

Current reason code:

- `bitcoin_balanced_high_volume_hub`

### Broad surface and repeated counterparty interaction

Current reason codes:

- `bitcoin_extremely_broad_counterparty_surface`
- `bitcoin_broad_counterparty_surface`
- `bitcoin_extreme_repeated_counterparty_interaction`
- `bitcoin_repeated_counterparty_interaction`

For exact weighting and score composition, see [`docs/SCORING.md`](SCORING.md).

---

## Validator Dataset Support

Validator dataset mode currently supports curated Bitcoin files such as:

- `data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json`
- `data/cases/curated-bitcoin/bitcoin-noisy-inbound-broad-surface.json`
- `data/cases/curated-bitcoin/bitcoin-legacy-mixed-flow-broad-value.json`

Example:

```bash
go run ./cmd/validator --dataset ./data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json
```

This is the implemented Bitcoin Layer 1 scoring path today.

---

## Current Limitations

The current Bitcoin model does not yet support:

- cluster-aware entity resolution
- change output attribution
- peel-chain detection
- rapid-spend scoring from output lifespan
- dormant-output reactivation scoring
- graph-aware hop exposure

The live Bitcoin strategy in `internal/address/bitcoin.go` is currently a basic validation and activity lookup path, not the same thing as the curated UTXO-flow Layer 1 model.

---

## Practical Interpretation

The current Bitcoin implementation is best described as:

- a realistic address-level UTXO behavior layer
- strong enough for curated benchmark cases
- intentionally limited before graph-aware expansion

That is the correct Wave 1-aligned repo story.

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/DATA-SOURCING-POLICY.md`](DATA-SOURCING-POLICY.md)
