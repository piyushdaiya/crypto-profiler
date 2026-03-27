# Crypto Profiler Typology Catalog

## Purpose

This document is the Wave 1 typology cleanup for the current repository state.

It separates:

- typologies that are implemented in code today
- typologies that are partially wired or supported only as groundwork
- typologies that remain backlog-only

In this repo, "implemented" can mean either:

- live analyzer support in `internal/analyzer`, or
- chain-specific validator dataset-mode scoring in `cmd/validator`

It does not mean "fully graph-aware" or "production-complete."

---

## Current Layer 1 Status

### Ethereum Layer 1

Implemented today:

- top-level EVM transaction-driven scoring in the shared analyzer
- curated Ethereum cases built from extracted native and ERC-20 transfer rows
- optional trace enrichment merged into curated cases
- validator dataset mode that surfaces trace-aware observational context

Not implemented today:

- ERC-20-specific validator scoring
- trace-driven live scoring
- hop-based exposure
- pass-through or U-turn detection

### Solana Layer 1

Implemented today:

- curated Solana Layer 1 based on large-value USDC/USDT stablecoin-flow summaries
- validator dataset mode for Solana curated cases
- role-aware scoring for source, destination, and authority-heavy activity

Not implemented today:

- general Solana instruction-aware profiling
- program-aware live scoring
- non-stablecoin Solana Layer 1 coverage

### Bitcoin Layer 1

Implemented today:

- curated Bitcoin Layer 1 based on address-scoped UTXO-flow summaries
- validator dataset mode for Bitcoin curated cases
- role-aware scoring for inbound-heavy, outbound-heavy, mixed-flow, and broad-surface cases

Not implemented today:

- cluster-aware Bitcoin modeling
- change detection
- graph-aware peel-chain or pass-through scoring

---

## Implemented Typologies

| Typology | Where it exists today | Current implementation shape |
| --- | --- | --- |
| Sanctions / watchlist hit | Shared analyzer | `direct_sanctions_match` short-circuits to a critical result. |
| Direct high-risk wallet or counterparty exposure | Shared analyzer | `profiled_address_high_risk_label`, `direct_mixer_interaction`, `direct_high_risk_entity`, `direct_sanctions_exposure`. |
| Trusted / exchange contextual mitigation | Shared analyzer | `profiled_address_trusted_label`, `exchange_interaction`, `trusted_or_protocol_interaction`. |
| Established-history mitigation | Shared analyzer | `established_history` reduces score for older wallets. |
| Fresh wallet | Shared analyzer | `fresh_wallet` is implemented as an age-based escalation. |
| Velocity burst | Shared analyzer | `high_velocity_behavior` is threshold-based on txs per active hour. |
| Repeated flagged counterparty interaction | Shared analyzer | `repeated_flagged_counterparty_interaction` is implemented with evidence counts and category-aware offsets. |
| Service concentration | Shared analyzer | `high_risk_service_concentration`, `exchange_concentration`, `single_service_concentration` use top labeled service concentration by interaction count. |
| Noisy inbound / dusting-like observation | Shared analyzer | `noisy_inbound_activity`, `high_counterparty_fan_in`, `zero_value_inbound_pattern` are low-severity observational rules. |
| Mixer plus reinforcing context | Shared analyzer | `combo_mixer_plus_fresh_wallet`, `combo_mixer_plus_high_velocity`, and `combo_contextual_mitigation_established_wallet` are implemented combination rules. |
| Ethereum internal-call context | Validator dataset mode | `dataset_trace_activity_observed`, `dataset_trace_failed_calls_observed`, `dataset_trace_deep_routing_observed`, `dataset_trace_broad_counterparty_surface` add trace-aware explanation context to curated EVM cases. |
| Solana source-heavy stablecoin distributor | Validator dataset mode | `solana_source_heavy_stablecoin_distributor` covers source-dominant large stablecoin distribution patterns. |
| Solana authority-heavy stablecoin operator | Validator dataset mode | `solana_authority_heavy_stablecoin_operator` covers authority-driven operational control behavior. |
| Solana broad stablecoin counterparty surface | Validator dataset mode | `solana_broad_mixed_stablecoin_surface`, `solana_broad_stablecoin_counterparty_surface`, `solana_repeated_large_counterparty_interaction`, `solana_mixed_stablecoin_activity`. |
| Bitcoin spend-heavy operational hub | Validator dataset mode | `bitcoin_spend_heavy_operational_hub` covers outbound-dominant operational behavior. |
| Bitcoin noisy inbound broad surface | Validator dataset mode | `bitcoin_noisy_inbound_broad_surface` covers receive-heavy broad-surface behavior. |
| Bitcoin mixed-flow broad-value legacy wallet | Validator dataset mode | `bitcoin_legacy_mixed_flow_broad_value` covers high-volume mixed legacy-format activity. |
| Bitcoin broad surface and repeated counterparty interaction | Validator dataset mode | `bitcoin_extremely_broad_counterparty_surface`, `bitcoin_broad_counterparty_surface`, `bitcoin_extreme_repeated_counterparty_interaction`, `bitcoin_repeated_counterparty_interaction`, `bitcoin_balanced_high_volume_hub`. |

---

## Partial / In-Progress Typologies

These have real code support or data groundwork, but not a complete end-to-end detector yet.

| Typology | Why it is only partial today |
| --- | --- |
| Mixer / tumbler exposure beyond direct contact | Direct mixer interaction is implemented, but hop-based or weighted mixer exposure is not. |
| Newly created wallet with immediate large flow | `fresh_wallet` exists, but there is no explicit large-value or immediate cash-out rule yet. |
| Contract-mediated pass-through behavior | Ethereum traces are extracted and surfaced in dataset mode, but no pass-through rule consumes them yet. |
| U-turn / round-trip behavior | Trace depth and counterparty context exist, but no inbound-then-back-out detector is implemented. |
| Value-weighted service concentration | Concentration works on interaction count today, not value-weighted flow. |
| Entity-aware concentration and repeated interaction | Exact-address labels work today; cluster-level or entity-merged scoring does not. |
| Bitcoin rapid spend / dormant reactivation | The UTXO data model supports these concepts, but the validator does not score them yet. |
| Solana protocol or instruction semantics | Stablecoin-flow role scoring exists, but full instruction-aware or program-aware Solana semantics do not. |
| Trace-aware Ethereum scoring | Trace context is visible in dataset mode, but the shared analyzer still scores from top-level transfers and labels. |

---

## Backlog-Only Typologies

These are explicitly part of the roadmap, but are not implemented in this repo today.

| Typology | Current status |
| --- | --- |
| 1-hop / 2-hop exposure | Backlog only. |
| Peel-chain behavior | Backlog only. |
| Graph-aware pass-through or layering | Backlog only. |
| Cross-chain / bridge-aware obfuscation | Backlog only. |
| ERC-20 Layer 1 case scoring | Backlog for Wave 2. |
| Stablecoin corridor or sanctions-evasion path modeling | Backlog only. |
| Cluster-aware graph scoring | Backlog only. |

---

## Practical Reading Guide

If you want the current repo story in one sentence:

- Ethereum is the most mature live-scored path, with trace-aware dataset enrichment.
- Solana is currently a stablecoin-flow Layer 1 dataset-mode implementation.
- Bitcoin is currently a UTXO-flow Layer 1 dataset-mode implementation.
- ERC-20 extraction groundwork exists, but ERC-20 Layer 1 scoring and validator support are still Wave 2 work.

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/EVM-CALLS-INTEGRATION.md`](EVM-CALLS-INTEGRATION.md)
- [`docs/SOLANA-DATA-MODEL.md`](SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
