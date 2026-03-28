# Crypto Profiler Scoring

## Purpose

This document explains how scoring works in the repository today.

It is intentionally implementation-aware:

- what is actually scored now
- which rules are only used in curated dataset mode
- what is still planned for later live or graph-aware work

Current status:

- Ethereum native transaction scoring is implemented in the shared analyzer.
- Ethereum trace integration is implemented as dataset enrichment and observational context.
- ERC-20 Layer 1 scoring is implemented in validator dataset mode.
- Solana stablecoin-flow scoring is implemented in validator dataset mode.
- Bitcoin UTXO-flow scoring is implemented in validator dataset mode.
- Tier 1 attribution-aware modifiers are implemented across live and dataset-mode outputs.

---

## Core Scoring Philosophy

### Deterministic-first

Sanctions and direct labeled high-risk exposure should dominate weaker heuristics.

### Explainability-first

Every meaningful score change should produce a visible reason code and evidence count when possible.

### Behavior first, attribution second

Wave 5A keeps the behavioral model intact.

The current ordering is:

1. behavior or dataset context is scored first
2. Tier 1 attribution is resolved second
3. a bounded attribution modifier is applied

This means labels improve precision, but they do not replace the underlying behavior model.

### Practical MVP weighting

The repo uses category scores and a weighted combined result:

- `FRAUD * 0.5`
- `REPUTATION * 0.3`
- `LENDING * 0.2`

Today, Solana and Bitcoin dataset-mode paths only use fraud and reputation, so their combined score is:

- `FRAUD * 0.5 + REPUTATION * 0.3`

### Score bands

Current grade thresholds are shared across the implemented scoring paths:

- `< 5`: `MINIMAL (Observed)`
- `< 20`: `LOW (Reviewable)`
- `< 50`: `ELEVATED`
- `>= 50`: `HIGH RISK`

Sanctions short-circuit to:

- `risk_score = 100`
- `risk_grade = "CRITICAL (Sanctioned)"`

---

## What Is Implemented Today

| Area                                 | Implemented now | Notes                                                             |
|--------------------------------------|-----------------|-------------------------------------------------------------------|
| Shared live analyzer                 | Yes             | Used for EVM live profiling and curated EVM cases.                |
| Tier 1 attribution-aware modifiers   | Yes             | Applied after behavior scoring across live and dataset-mode paths. |
| Ethereum trace-aware context         | Yes             | Added in dataset mode as observational context, not live scoring. |
| ERC-20 Layer 1 scoring               | Yes             | Curated dataset mode only.                                        |
| Solana stablecoin-flow scoring       | Yes             | Curated dataset mode only.                                        |
| Bitcoin UTXO-flow scoring            | Yes             | Curated dataset mode only.                                        |
| 1-hop / 2-hop exposure scoring       | No              | Planned.                                                          |
| Graph-aware or cluster-aware scoring | No              | Planned.                                                          |

---

## Ethereum Native and Trace-Aware Scoring

Ethereum currently uses two layers:

1. the shared analyzer for top-level wallet, label, and transfer behavior
2. optional trace-aware dataset context for curated EVM cases

### Shared analyzer rule families

| Rule family                | Current reason codes                                                                                                | Implementation note                                                                     |
|----------------------------|---------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------|
| Sanctions                  | `direct_sanctions_match`, `direct_sanctions_exposure`                                                               | Direct match short-circuits; direct exposure is also scored.                            |
| Direct high-risk exposure  | `direct_mixer_interaction`, `direct_high_risk_entity`                                                               | Direct labeled contact is a major escalation path; profile-level attribution now applies afterward. |
| Wallet age                 | `fresh_wallet`, `established_history`                                                                               | New wallets are escalated; old wallets can be mitigated.                                |
| Trusted context            | `exchange_interaction`, `trusted_or_protocol_interaction`, `contextual_infrastructure_interaction`                  | Trusted or exchange context reduces score but does not override stronger fraud signals. |
| Activity burst             | `high_velocity_behavior`                                                                                            | Tx-rate threshold heuristic.                                                            |
| Repeated high-risk contact | `repeated_flagged_counterparty_interaction`                                                                         | Category-aware repeated-contact scoring.                                                |
| Service concentration      | `high_risk_service_concentration`, `exchange_concentration`, `single_service_concentration`                         | Count-based concentration to the top labeled service.                                   |
| Noisy inbound observations | `noisy_inbound_activity`, `high_counterparty_fan_in`, `zero_value_inbound_pattern`                                  | Low-severity observational rules.                                                       |
| Combination logic          | `combo_mixer_plus_fresh_wallet`, `combo_mixer_plus_high_velocity`, `combo_contextual_mitigation_established_wallet` | Multi-signal escalation and contextual mitigation.                                      |

### Practical offsets in the shared analyzer

These are the important current offsets, as implemented:

- `direct_sanctions_match`: `+100` and short-circuit
- `direct_high_risk_entity`: `+45`
- `direct_mixer_interaction`: `+20`
- `fresh_wallet`: `+35`
- `high_velocity_behavior`: `+25`
- `high_risk_service_concentration`: `+18`
- `established_history`: `-10`
- `exchange_interaction`: `-5`
- `trusted_or_protocol_interaction`: `-5`
- `contextual_infrastructure_interaction`: `-5`
- `exchange_concentration`: `-4`
- `single_service_concentration`: `-4`
- `noisy_inbound_activity`: `+2`
- `high_counterparty_fan_in`: `+2`
- `zero_value_inbound_pattern`: `+1`
- `combo_mixer_plus_fresh_wallet`: `+20`
- `combo_mixer_plus_high_velocity`: `+20`
- `combo_contextual_mitigation_established_wallet`: `-15`

`repeated_flagged_counterparty_interaction` is dynamic:

- base offsets depend on counterparty category
- repeated interaction count increases the offset further

### Ethereum trace-aware dataset context

Trace integration currently improves explanation, not core risk weighting.

Current trace-context reasons are all `0`-offset observational reasons:

- `dataset_trace_activity_observed`
- `dataset_trace_failed_calls_observed`
- `dataset_trace_deep_routing_observed`
- `dataset_trace_broad_counterparty_surface`

This means curated EVM cases can say:

- internal traces were present
- failed calls existed
- routing depth was high
- internal counterparty surface was broad

without pretending the repo already has mature trace-driven behavioral scoring.

### Tier 1 attribution modifiers

After the shared analyzer or dataset-mode scoring runs, the resolver applies a bounded attribution modifier when a Tier 1 label is available.

Current modifier families:

| Attribution shape                             | Current reason code                    | Current effect      |
|-----------------------------------------------|----------------------------------------|---------------------|
| Sanctioned actor attribution                  | `tier1_profile_sanctioned_attribution` | `FRAUD +60`         |
| Illicit service, exploit, or scam attribution | `tier1_profile_risky_attribution`      | `FRAUD +45`         |
| Trusted protocol or exchange attribution      | `tier1_profile_contextual_attribution` | `REPUTATION -10`    |
| Mining pool or treasury attribution           | `tier1_profile_contextual_attribution` | `REPUTATION -8`     |

Important implementation notes:

- direct watchlist sanctions still short-circuit first
- Tier 1 attribution adds one resolved modifier rather than dumping all source labels into the score
- contextual labels are meant to reduce false positives, not guarantee a low-risk outcome

### What is not implemented yet for Ethereum

- trace-driven pass-through or U-turn detection
- value-weighted concentration from traces
- hop-based exposure
- graph-aware path scoring
- corroborating-source conflict resolution beyond Tier 1

---

## ERC-20 Layer 1 Scoring

ERC-20 scoring is currently implemented only in validator dataset mode for curated ERC-20 Layer 1 cases.

It is built from `erc20_summary`, `token_breakdown`, and `top_counterparties`.

### Implemented ERC-20 rule families

| Rule family                            | Trigger shape                                                                      | Current reason code                            | Offset           |
|----------------------------------------|------------------------------------------------------------------------------------|------------------------------------------------|------------------|
| Trusted protocol token hub             | Trusted/protocol label with very large transfer count and broad counterparty reach | `erc20_trusted_protocol_token_hub`             | `REPUTATION +16` |
| Exchange-style service surface         | Exchange label with large transfer count and broad counterparty reach              | `erc20_exchange_service_surface`               | `REPUTATION +14` |
| Noisy inbound token surface            | Inbound-heavy activity with broad counterparties across many token contracts       | `erc20_noisy_token_inbound_surface`            | `FRAUD +14`      |
| Broad token counterparty surface       | Very broad token counterparty surface across many token contracts                  | `erc20_broad_token_counterparty_surface`       | `FRAUD +8`       |
| Mixed token activity                   | Activity across many token contracts                                               | `erc20_mixed_token_activity`                   | `REPUTATION +4`  |
| Repeated counterparty activity         | Many repeated counterparties or very high interaction with one counterparty        | `erc20_repeated_counterparty_activity`         | `FRAUD +4`       |
| Single-token operational concentration | One token dominates a large share of transfer activity                             | `erc20_single_token_operational_concentration` | `REPUTATION +3`  |

### Practical interpretation

Today, ERC-20 dataset-mode scoring is trying to answer questions like:

- is this wallet behaving like a trusted protocol token hub?
- is this token surface broad enough to be operationally significant or reviewable?
- is the activity inbound-noisy across many counterparties and tokens?
- is there repeated counterparty structure worth highlighting?
- is token activity mixed across many contracts or heavily concentrated to one asset?
- does Tier 1 attribution explain a broad token surface as trusted infrastructure instead of generic noise?

### What is not implemented yet for ERC-20

- live ERC-20 scoring inside the EVM address strategy
- swap-aware or protocol-intent interpretation
- trace-aware ERC-20 pass-through or U-turn detection
- hop-based or graph-aware token exposure
- corroborating-source attribution beyond Tier 1

---

## Solana Stablecoin-Flow Scoring

Solana scoring is currently implemented only in validator dataset mode for curated stablecoin-flow cases.

It is built from summary fields in `stablecoin_summary`, `mint_breakdown`, and `top_counterparties`.

### Implemented Solana rule families

| Rule family                             | Trigger shape                                                                             | Current reason code                              | Offset           |
|-----------------------------------------|-------------------------------------------------------------------------------------------|--------------------------------------------------|------------------|
| Source-heavy stablecoin distribution    | Dominant role is `source`, with very large outbound count and broad source counterparties | `solana_source_heavy_stablecoin_distributor`     | `REPUTATION +12` |
| Authority-heavy operator behavior       | Dominant role is `authority` with very large authority-linked transfer count              | `solana_authority_heavy_stablecoin_operator`     | `FRAUD +28`      |
| Very broad mixed stablecoin surface     | Very broad counterparty surface across multiple mints                                     | `solana_broad_mixed_stablecoin_surface`          | `FRAUD +12`      |
| Broad stablecoin counterparty surface   | Broad counterparty surface without the stronger mixed-surface case                        | `solana_broad_stablecoin_counterparty_surface`   | `FRAUD +8`       |
| Mixed stablecoin activity               | Activity across multiple stablecoin mints                                                 | `solana_mixed_stablecoin_activity`               | `REPUTATION +4`  |
| Repeated large counterparty interaction | Heavy repeated interaction with the top counterparty                                      | `solana_repeated_large_counterparty_interaction` | `FRAUD +4`       |

### Practical interpretation

Today, Solana dataset-mode scoring is trying to answer questions like:

- is this wallet acting like a large stablecoin distributor?
- is this wallet authority-heavy in a way that looks operational or reviewable?
- is the stablecoin surface broad enough to warrant attention?
- is there unusual concentration to a single repeated counterparty?
- does Tier 1 attribution provide contextual operator or service identity for a curated case?

### What is not implemented yet for Solana

- live scoring from RPC-hydrated Solana history
- instruction-aware or program-aware scoring
- non-stablecoin Layer 1 coverage
- bridge-aware or graph-aware scoring
- corroborating-source attribution beyond Tier 1

---

## Bitcoin UTXO-Flow Scoring

Bitcoin scoring is currently implemented only in validator dataset mode for curated UTXO-flow cases.

It is built from `utxo_summary` and `top_counterparties`.

### Implemented Bitcoin rule families

| Rule family                          | Trigger shape                                                                     | Current reason code                                 | Offset          |
|--------------------------------------|-----------------------------------------------------------------------------------|-----------------------------------------------------|-----------------|
| Legacy mixed-flow broad-value wallet | Legacy-format address with very large inbound, outbound, and counterparty breadth | `bitcoin_legacy_mixed_flow_broad_value`             | `FRAUD +16`     |
| Spend-heavy operational hub          | Outbound-dominant wallet with very large spend count and broad surface            | `bitcoin_spend_heavy_operational_hub`               | `FRAUD +18`     |
| Noisy inbound broad surface          | Inbound-dominant wallet with very large inbound count and broad surface           | `bitcoin_noisy_inbound_broad_surface`               | `FRAUD +14`     |
| Balanced high-volume hub             | Large balanced inbound and outbound activity                                      | `bitcoin_balanced_high_volume_hub`                  | `REPUTATION +8` |
| Extremely broad surface              | Very high unique counterparty count                                               | `bitcoin_extremely_broad_counterparty_surface`      | `FRAUD +10`     |
| Broad surface                        | Broad, but not extreme, unique counterparty count                                 | `bitcoin_broad_counterparty_surface`                | `FRAUD +6`      |
| Extreme repeated interaction         | Very high interaction count with top counterparty                                 | `bitcoin_extreme_repeated_counterparty_interaction` | `FRAUD +10`     |
| Repeated interaction                 | Heavy interaction with top counterparty                                           | `bitcoin_repeated_counterparty_interaction`         | `FRAUD +4`      |

### Practical interpretation

Today, Bitcoin dataset-mode scoring is trying to answer questions like:

- is this address mostly receiving or mostly spending?
- is the counterparty surface unusually broad?
- is this more like an operational hub than a normal wallet?
- is there extreme repeated interaction with one counterparty?
- does mining-pool context explain a broad operational-looking address more safely?

### What is not implemented yet for Bitcoin

- rapid-spend scoring from output lifespan
- dormant-output reactivation
- peel-chain logic
- change-aware or cluster-aware modeling
- hop-based exposure scoring
- WalletExplorer or other corroborating attribution sources

---

## What Tier 1 Attribution Changes Today

Wave 5A improves score precision in three ways:

1. risky actors can escalate scores without replacing behavioral reasons
2. contextual infrastructure can reduce false positives for broad-surface or high-volume wallets
3. reports can explain why a label mattered, including actor, confidence, and source tier

This is deliberately narrower than a full attribution platform.

Tier 1 today means:

- GraphSense-style structured labels
- Bitcoin mining-pool context
- repo-local bootstrap overrides

Deferred to later waves:

- WalletExplorer
- secondary or corroborating-source ingestion
- cluster-aware actor reasoning
- graph-aware attribution paths
- WalletExplorer or other corroborating attribution sources

---

## Dataset Mode vs Future Live or Graph-Aware Scoring

### Dataset mode today

Dataset mode is intentionally practical and reproducible.

It uses:

- curated JSON artifacts committed in the repo
- extracted summaries that were already reduced offline
- fixed case context, not live graph traversal

That makes it good for:

- demos
- repeatable screenshots
- case studies
- scoring-rule development

### What dataset mode does not do

It does not currently:

- recompute chain history live from the full raw source
- build 1-hop or 2-hop exposure graphs
- merge cluster identities
- compute value-aware path risk across protocols

### Future live or graph-aware scoring

The roadmap after the current ERC-20 implementation is to add:

- graph-aware 1-hop and 2-hop exposure
- trace-aware pass-through and U-turn logic
- fresh-wallet plus immediate large-flow reasoning
- value-weighted concentration
- richer Solana and Bitcoin live-flow reasoning

---

## Related Documents

- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/EVM-CALLS-INTEGRATION.md`](EVM-CALLS-INTEGRATION.md)
- [`docs/ERC20-DATA-MODEL.md`](ERC20-DATA-MODEL.md)
- [`docs/SOLANA-DATA-MODEL.md`](SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
