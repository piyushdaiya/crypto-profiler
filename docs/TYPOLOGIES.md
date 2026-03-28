# Crypto Profiler Typology Catalog

## Purpose

This document catalogs the typologies represented in the current repository.

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

- trace-driven live scoring
- generalized graph-aware exposure
- trace-native pass-through or U-turn detection

### ERC-20 Layer 1

Implemented today:

- curated ERC-20 Layer 1 based on address-scoped transfer-row summaries
- validator dataset mode for ERC-20 curated cases
- token-aware scoring for trusted protocol hubs, noisy inbound token surfaces, broad token surfaces, repeated counterparties, mixed token activity, and single-token concentration

Not implemented today:

- live ERC-20 scoring inside the EVM strategy
- swap-aware or protocol-intent interpretation
- trace-aware ERC-20 pass-through or U-turn detection
- generalized graph-aware token exposure

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

- generalized cluster-aware Bitcoin modeling
- change detection
- graph-aware peel-chain or pass-through scoring

---

## Implemented Typologies

| Typology                                                    | Where it exists today  | Current implementation shape                                                                                                                                                                                                |
|-------------------------------------------------------------|------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Sanctions / watchlist hit                                   | Shared analyzer        | `direct_sanctions_match` short-circuits to a critical result.                                                                                                                                                               |
| Direct high-risk wallet or counterparty exposure            | Shared analyzer        | `direct_mixer_interaction`, `direct_high_risk_entity`, and `direct_sanctions_exposure` remain part of the live behavior layer.                                                                                              |
| Trusted / exchange contextual mitigation                    | Shared analyzer        | `exchange_interaction`, `trusted_or_protocol_interaction`, and `contextual_infrastructure_interaction` provide behavior-level mitigation before Tier 1 profile attribution is applied.                                       |
| Tier 1 risky actor attribution                              | Attribution layer      | `tier1_profile_sanctioned_attribution` and `tier1_profile_risky_attribution` now apply deterministic post-behavior escalation from Tier 1 labels.                                                                          |
| Tier 1 contextual infrastructure suppression                | Attribution layer      | `tier1_profile_contextual_attribution` reduces false positives for trusted protocols, exchanges, mining pools, or treasury-like infrastructure.                                                                            |
| Established-history mitigation                              | Shared analyzer        | `established_history` reduces score for older wallets.                                                                                                                                                                      |
| Fresh wallet                                                | Shared analyzer        | `fresh_wallet` is implemented as an age-based escalation.                                                                                                                                                                   |
| Velocity burst                                              | Shared analyzer        | `high_velocity_behavior` is threshold-based on txs per active hour.                                                                                                                                                         |
| Repeated flagged counterparty interaction                   | Shared analyzer        | `repeated_flagged_counterparty_interaction` is implemented with evidence counts and category-aware offsets.                                                                                                                 |
| Service concentration                                       | Shared analyzer        | `high_risk_service_concentration`, `exchange_concentration`, `single_service_concentration` use top labeled service concentration by interaction count.                                                                     |
| Noisy inbound / dusting-like observation                    | Shared analyzer        | `noisy_inbound_activity`, `high_counterparty_fan_in`, `zero_value_inbound_pattern` are low-severity observational rules.                                                                                                    |
| Mixer plus reinforcing context                              | Shared analyzer        | `combo_mixer_plus_fresh_wallet`, `combo_mixer_plus_high_velocity`, and `combo_contextual_mitigation_established_wallet` are implemented combination rules.                                                                  |
| Ethereum internal-call context                              | Validator dataset mode | `dataset_trace_activity_observed`, `dataset_trace_failed_calls_observed`, `dataset_trace_deep_routing_observed`, `dataset_trace_broad_counterparty_surface` add trace-aware explanation context to curated EVM cases.       |
| Solana source-heavy stablecoin distributor                  | Validator dataset mode | `solana_source_heavy_stablecoin_distributor` covers source-dominant large stablecoin distribution patterns.                                                                                                                 |
| Solana authority-heavy stablecoin operator                  | Validator dataset mode | `solana_authority_heavy_stablecoin_operator` covers authority-driven operational control behavior.                                                                                                                          |
| Solana broad stablecoin counterparty surface                | Validator dataset mode | `solana_broad_mixed_stablecoin_surface`, `solana_broad_stablecoin_counterparty_surface`, `solana_repeated_large_counterparty_interaction`, `solana_mixed_stablecoin_activity`.                                              |
| ERC-20 trusted protocol token hub                           | Validator dataset mode | `erc20_trusted_protocol_token_hub` recognizes labeled trusted protocol token hubs with large token-surface activity.                                                                                                        |
| ERC-20 exchange-style service surface                       | Validator dataset mode | `erc20_exchange_service_surface` recognizes labeled exchange or service wallets with broad token-surface activity.                                                                                                          |
| ERC-20 noisy inbound or broad token surface                 | Validator dataset mode | `erc20_noisy_token_inbound_surface` and `erc20_broad_token_counterparty_surface` score large inbound-heavy or very broad token surfaces.                                                                                    |
| ERC-20 repeated token counterparty activity                 | Validator dataset mode | `erc20_repeated_counterparty_activity` scores repeated interaction with the same counterparty set.                                                                                                                          |
| ERC-20 mixed token activity and concentration               | Validator dataset mode | `erc20_mixed_token_activity` and `erc20_single_token_operational_concentration` capture token diversity and dominant-token concentration.                                                                                   |
| Actor-aware repeated interaction and concentration          | Attribution layer      | `actor_repeated_risky_interaction`, `actor_risky_concentration`, `actor_contextual_repeated_interaction`, and `actor_contextual_concentration` refine scoring when counterparties resolve to the same actor with strong support. |
| Practical hop-aware exposure summaries                      | Attribution layer      | `attribution_insights` now include direct and near exposure summaries tied to attributed actors when sampled Layer 1 flow order supports them.                                                                              |
| Pass-through and U-turn attribution                         | Attribution layer      | `actor_pass_through_risky_exposure`, `actor_u_turn_risky_service`, and the matching report insights now surface bounded routing findings where actor support is strong enough.                                             |
| Cluster-aware attribution refinement                        | Attribution layer      | `cluster_grouping` insights now highlight sampled multi-address actor groupings when attribution links multiple addresses to the same actor.                                                                                 |
| Bitcoin spend-heavy operational hub                         | Validator dataset mode | `bitcoin_spend_heavy_operational_hub` covers outbound-dominant operational behavior.                                                                                                                                        |
| Bitcoin noisy inbound broad surface                         | Validator dataset mode | `bitcoin_noisy_inbound_broad_surface` covers receive-heavy broad-surface behavior.                                                                                                                                          |
| Bitcoin mixed-flow broad-value legacy wallet                | Validator dataset mode | `bitcoin_legacy_mixed_flow_broad_value` covers high-volume mixed legacy-format activity.                                                                                                                                    |
| Bitcoin broad surface and repeated counterparty interaction | Validator dataset mode | `bitcoin_extremely_broad_counterparty_surface`, `bitcoin_broad_counterparty_surface`, `bitcoin_extreme_repeated_counterparty_interaction`, `bitcoin_repeated_counterparty_interaction`, `bitcoin_balanced_high_volume_hub`. |

---

## Partial / In-Progress Typologies

These have real code support or data groundwork, but not a complete end-to-end detector yet.

| Typology                                            | Why it is only partial today                                                                                        |
|-----------------------------------------------------|---------------------------------------------------------------------------------------------------------------------|
| Mixer / tumbler exposure beyond direct contact      | Direct mixer interaction is implemented, but hop-based or weighted mixer exposure is not.                           |
| Newly created wallet with immediate large flow      | `fresh_wallet` exists, but there is no explicit large-value or immediate cash-out rule yet.                         |
| Contract-mediated trace-native pass-through behavior | Ethereum traces are extracted and surfaced in dataset mode, but the current pass-through logic still works from sampled Layer 1 flow order rather than full trace semantics. |
| ERC-20 trace-aware routing interpretation           | ERC-20 transfer-row scoring and sampled actor-aware routing insight exist, but traces are not yet used to distinguish swaps or protocol intent. |
| Value-weighted service concentration                | Concentration works on interaction count today, not value-weighted flow.                                            |
| Generalized entity-wide concentration and repeated interaction | Actor-aware refinement exists for sampled, strongly attributed actors, but not as a generalized graph-wide entity engine. |
| Bitcoin rapid spend / dormant reactivation          | The UTXO data model supports these concepts, but the validator does not score them yet.                             |
| Solana protocol or instruction semantics            | Stablecoin-flow role scoring exists, but full instruction-aware or program-aware Solana semantics do not.           |
| Trace-aware Ethereum scoring                        | Trace context is visible in dataset mode, but the shared analyzer still scores from top-level transfers and labels. |

---

## Backlog-Only Typologies

These are future-work areas, but they are not implemented in this repo today.

| Typology                                               | Current status |
|--------------------------------------------------------|----------------|
| Peel-chain behavior                                    | Backlog only.  |
| Graph-aware 1-hop / 2-hop exposure                     | Backlog only.  |
| Graph-aware pass-through or layering                   | Backlog only.  |
| Cross-chain / bridge-aware obfuscation                 | Backlog only.  |
| Stablecoin corridor or sanctions-evasion path modeling | Backlog only.  |
| Cluster-aware graph scoring                            | Backlog only.  |

---

## Practical Reading Guide

If you want the current repo story in one sentence:

- Ethereum is the most mature live-scored path, with trace-aware dataset enrichment.
- ERC-20 is now a transfer-row Layer 1 dataset-mode implementation.
- Solana is currently a stablecoin-flow Layer 1 dataset-mode implementation.
- Bitcoin is currently a UTXO-flow Layer 1 dataset-mode implementation.

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/EVM-CALLS-INTEGRATION.md`](EVM-CALLS-INTEGRATION.md)
- [`docs/LABEL-SOURCE-HIERARCHY.md`](LABEL-SOURCE-HIERARCHY.md)
- [`docs/ERC20-DATA-MODEL.md`](ERC20-DATA-MODEL.md)
- [`docs/SOLANA-DATA-MODEL.md`](SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
