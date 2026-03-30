
# Crypto Profiler Scoring

## Purpose

This document explains how scoring works in the repository today.

It is intentionally implementation-aware:

- what is actually scored now
- which rules are only used in curated dataset mode
- how attribution, actor/exposure refinement, and bounded graph-aware scoring fit into the pipeline
- what still remains future work

Current status:

- Ethereum native transaction scoring is implemented in the shared analyzer.
- Ethereum trace integration is implemented as dataset enrichment and observational context.
- Optimism Phase 1 tx-only scoring is implemented in validator dataset mode.
- ERC-20 scoring is implemented in validator dataset mode.
- Solana stablecoin-flow scoring is implemented in validator dataset mode.
- Bitcoin UTXO-flow scoring is implemented in validator dataset mode.
- Tier 1 attribution-aware modifiers are implemented across live and dataset-mode outputs.
- Secondary corroborating sources are implemented with bounded confidence and note-level conflict handling.
- Actor/exposure refinement is implemented where attribution support is strong enough.
- Bounded graph summary and bounded graph-aware scoring are implemented when attributed graph coverage is meaningful.

---

## Core Scoring Philosophy

### Deterministic-first

Sanctions and direct labeled high-risk exposure should dominate weaker heuristics.

### Explainability-first

Every meaningful score change should produce a visible reason code and evidence count when possible.

### Behavior first, attribution second

The behavioral model remains the primary scoring engine.

The current ordering is:

1. behavior or dataset context is scored first
2. Tier 1 attribution is resolved second
3. secondary corroboration can raise confidence or surface conflicts
4. actor/exposure refinement can be applied when attribution support is strong
5. bounded graph-aware scoring can be applied when attributed graph coverage is meaningful

This means labels and graph context improve precision, but they do not replace the underlying behavior model.

### Practical MVP weighting

The repo uses category scores and a weighted combined result:

- `FRAUD * 0.5`
- `REPUTATION * 0.3`
- `LENDING * 0.2`

Today, Solana and Bitcoin dataset-mode paths only use fraud and reputation, so their combined score is:

- `FRAUD * 0.5 + REPUTATION * 0.3`

Optimism Phase 1 also effectively uses fraud and reputation in its current dataset-mode scoring.

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

| Area                                | Implemented now | Notes                                                                                                                                        |
|-------------------------------------|-----------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| Shared live analyzer                | Yes             | Used for EVM live profiling and curated EVM cases.                                                                                           |
| Tier 1 attribution-aware modifiers  | Yes             | Applied after behavior scoring across live and dataset-mode paths.                                                                           |
| Secondary corroboration             | Yes             | Raises confidence modestly, adds bounded support, and surfaces conflicts.                                                                    |
| Actor/exposure refinement           | Yes             | Direct, near, repeated, concentration, pass-through, and U-turn findings are implemented in bounded form when attribution support is strong. |
| Bounded graph summary               | Yes             | Renders only when attributed graph coverage is meaningful.                                                                                   |
| Bounded graph-aware scoring         | Yes             | Selected motifs and concentration patterns can add bounded score refinements.                                                                |
| Ethereum trace-aware context        | Yes             | Added in dataset mode as observational context, not live scoring.                                                                            |
| Optimism Layer 2 tx-only scoring    | Yes             | Curated dataset mode only; decoded-events deferred in Phase 1 due cost/ROI.                                                                  |
| ERC-20 scoring                      | Yes             | Curated dataset mode only.                                                                                                                   |
| Solana stablecoin-flow scoring      | Yes             | Curated dataset mode only.                                                                                                                   |
| Bitcoin UTXO-flow scoring           | Yes             | Curated dataset mode only.                                                                                                                   |
| Full generalized graph scoring      | No              | Not implemented.                                                                                                                             |
| Full multi-hop live graph analytics | No              | Not implemented.                                                                                                                             |

---

## Ethereum Native and Trace-Aware Scoring

Ethereum currently uses two layers:

1. the shared analyzer for top-level wallet, label, and transfer behavior
2. optional trace-aware dataset context for curated EVM cases

### Shared analyzer rule families

| Rule family                | Current reason codes                                                                                                | Implementation note                                                                                 |
|----------------------------|---------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------|
| Sanctions                  | `direct_sanctions_match`, `direct_sanctions_exposure`                                                               | Direct match short-circuits; direct exposure is also scored.                                        |
| Direct high-risk exposure  | `direct_mixer_interaction`, `direct_high_risk_entity`                                                               | Direct labeled contact is a major escalation path; profile-level attribution now applies afterward. |
| Wallet age                 | `fresh_wallet`, `established_history`                                                                               | New wallets are escalated; old wallets can be mitigated.                                            |
| Trusted context            | `exchange_interaction`, `trusted_or_protocol_interaction`, `contextual_infrastructure_interaction`                  | Trusted or exchange context reduces score but does not override stronger fraud signals.             |
| Activity burst             | `high_velocity_behavior`                                                                                            | Tx-rate threshold heuristic.                                                                        |
| Repeated high-risk contact | `repeated_flagged_counterparty_interaction`                                                                         | Category-aware repeated-contact scoring.                                                            |
| Service concentration      | `high_risk_service_concentration`, `exchange_concentration`, `single_service_concentration`                         | Count-based concentration to the top labeled service.                                               |
| Noisy inbound observations | `noisy_inbound_activity`, `high_counterparty_fan_in`, `zero_value_inbound_pattern`                                  | Low-severity observational rules.                                                                   |
| Combination logic          | `combo_mixer_plus_fresh_wallet`, `combo_mixer_plus_high_velocity`, `combo_contextual_mitigation_established_wallet` | Multi-signal escalation and contextual mitigation.                                                  |

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

---

## Optimism Layer 2 Phase 1 Scoring

Optimism Phase 1 is currently implemented as a transactions-only dataset-mode path.

It is intentionally conservative and cost-aware:

- uses the repo’s canonical 90-day window
- mines and exports shortlisted Optimism transactions from BigQuery
- summarizes those exports locally on the homelab
- derives the first scoring pass from transaction structure, counterparties, destination concentration, and function-selector concentration
- defers decoded-events for now because the scan-cost / ROI tradeoff was poor for the initial implementation

### Implemented Optimism Phase 1 rule families

| Rule family                            | Current reason code                              | Current effect   |
|----------------------------------------|--------------------------------------------------|------------------|
| Repeated-contract router-like behavior | `optimism_repeated_contract_router_like`         | `REPUTATION +10` |
| Low-diversity contextual concentration | `optimism_low_counterparty_diversity_context`    | `REPUTATION +4`  |
| Broad operational hub                  | `optimism_broad_operational_hub`                 | `FRAUD +14`      |
| Extremely broad counterparty surface   | `optimism_extremely_broad_counterparty_surface`  | `FRAUD +8`       |
| Mixed-flow operational pattern         | `optimism_mixed_flow_operational_pattern`        | `REPUTATION +3`  |

### Practical interpretation

The current tx-only Optimism layer is trying to answer questions like:

- Is this wallet dominated by one contract destination and one function selector?
- Does the transaction shape look like repeated router / contract-centric infrastructure usage?
- Is the wallet operating across a very broad counterparty surface?
- Is the activity mixed inbound/outbound in a way that looks operational and reviewable?
- Does the broad surface look like an operational hub rather than a concentrated service wallet?

### What is deferred

Optimism decoded-events were evaluated during Phase 1, but deferred because query-scan cost was high relative to immediate scoring value for the first implementation pass.

That means Phase 1 does not yet claim:

- decoded-event-native protocol classification
- event-driven bridge classification
- richer contract semantics from event arguments
- graph-aware Optimism motifs built from decoded event streams

Those remain good next-stage enhancements once the transactions-only path is fully documented and stabilized.

---

## Tier 1 Attribution Modifiers

After the shared analyzer or dataset-mode scoring runs, the resolver applies a bounded attribution modifier when a Tier 1 label is available.

Current modifier families:

| Attribution shape                             | Current reason code                    | Current effect   |
|-----------------------------------------------|----------------------------------------|------------------|
| Sanctioned actor attribution                  | `tier1_profile_sanctioned_attribution` | `FRAUD +60`      |
| Illicit service, exploit, or scam attribution | `tier1_profile_risky_attribution`      | `FRAUD +45`      |
| Trusted protocol or exchange attribution      | `tier1_profile_contextual_attribution` | `REPUTATION -10` |
| Mining pool or treasury attribution           | `tier1_profile_contextual_attribution` | `REPUTATION -8`  |

Important implementation notes:

- direct watchlist sanctions still short-circuit first
- Tier 1 attribution adds one resolved modifier rather than dumping all source labels into the score
- contextual labels are meant to reduce false positives, not guarantee a low-risk outcome

---

## Secondary Corroboration Behavior

Secondary corroboration adds a bounded layer on top of Tier 1.

Current practical rules:

- a lone secondary risky attribution is bounded to a small `+4` signal
- a lone secondary contextual attribution is bounded to a small `-2` contextual adjustment
- a secondary source corroborating a Tier 1 result adds only a modest `+3` or `-2` reinforcement
- conflicting secondary sources add a note-only `0`-offset reason

Current reason codes:

- `secondary_profile_risky_attribution`
- `secondary_profile_contextual_attribution`
- `secondary_corroborated_risky_attribution`
- `secondary_corroborated_contextual_attribution`
- `attribution_source_conflict_observed`

The design goal is deliberate:

- secondary sources can sharpen confidence
- secondary sources can improve analyst explanation
- secondary sources must not create large hard jumps on their own

---

## Actor and Exposure Refinements

The current actor/exposure layer adds a bounded refinement step after attribution resolution.

Current practical reason codes:

| Refinement shape                            | Current reason code                     | Current effect      |
|---------------------------------------------|-----------------------------------------|---------------------|
| Actor-aware repeated risky interaction      | `actor_repeated_risky_interaction`      | `FRAUD +4` to `+6`  |
| Actor-aware risky concentration             | `actor_risky_concentration`             | `FRAUD +5`          |
| Actor-aware repeated contextual interaction | `actor_contextual_repeated_interaction` | `REPUTATION -1.5`   |
| Actor-aware contextual concentration        | `actor_contextual_concentration`        | `REPUTATION -3`     |
| Pass-through exposure to risky actor        | `actor_pass_through_risky_exposure`     | `FRAUD +4`          |
| U-turn through risky actor                  | `actor_u_turn_risky_service`            | `FRAUD +3`          |

The current actor/exposure layer also adds `attribution_insights` for:

- direct exposure to attributed actors
- near exposure to risky actors through an intermediary
- cluster-aware grouping when multiple sampled addresses resolve to the same actor
- pass-through and U-turn findings where sampled flow order supports them

Guardrails:

- stronger score changes require strong, non-secondary attribution support
- secondary-only attribution can still appear in analyst-facing insights, but does not drive the stronger actor/exposure score modifiers
- the goal is explainable refinement, not a second opaque scoring engine

---

## Graph-aware Summary and Bounded Graph Scoring

Crypto Profiler includes a bounded graph-analysis layer on top of attribution and behavioral scoring.

This layer is intentionally conservative:

- it only activates when attributed graph coverage is meaningful
- it does not override Tier 1 attribution or existing behavioral scoring
- it is designed to improve explanation quality and add bounded score adjustments rather than simulate a full graph platform

### When graph summary is shown

The analyst-facing report renders a `Graph Summary` section only when:

- attributed graph coverage is meaningful enough to avoid misleading output
- the sampled graph can be rolled up into actor-level summaries with enough support to be useful

In low-coverage cases, graph summary is suppressed rather than overstated.

### Graph summary outputs

When present, the graph summary can include:

- attributed graph coverage over sampled interactions
- unique actor count
- direct risky actor count
- direct contextual actor count
- near risky actor count
- top actors by attributed interaction share
- graph motifs when the sampled graph supports them

### Graph motifs currently supported

The current bounded motif layer includes:

- contextual-to-risky pass-through
- risky-actor U-turn style behavior
- contextual fan-in hubs
- contextual fan-out hubs
- risky fan-out patterns

These motifs are sampled, explanation-first findings. They are not presented as full path reconstruction.

### Bounded graph-aware score modifiers

Graph-aware score changes are deliberately modest.

Examples of bounded modifiers:

- concentration on a risky actor when attributed graph coverage is meaningful
- near-risky actor exposure
- risky-actor U-turn motif
- contextual-to-risky pass-through motif
- contextual concentration reducing false-positive pressure in some cases

These modifiers:

- require meaningful attributed graph coverage
- remain bounded in size
- augment existing reasons instead of replacing them

### What graph-aware scoring does not claim

The current implementation does not claim:

- full graph reconstruction
- generalized path search over arbitrary hops
- value-weighted graph scoring
- comprehensive cluster resolution
- trace-native path decoding across all chains
- complete live graph analytics

It is better understood as:

- sampled graph intelligence
- explainable actor-level rollups
- bounded motif detection
- cautious score refinement where the data supports it

### What is not implemented yet for Ethereum

- trace-driven live pass-through or U-turn detection
- value-weighted concentration from traces
- generalized graph-aware path scoring
- full live multi-hop graph scoring

---

## ERC-20 Scoring

ERC-20 scoring is currently implemented only in validator dataset mode for curated ERC-20 cases.

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
- does meaningful attributed graph coverage support bounded graph summary or motif-based refinement?

### What is not implemented yet for ERC-20

- live ERC-20 scoring inside the EVM address strategy
- swap-aware or protocol-intent interpretation
- trace-aware ERC-20 swap or protocol-intent decoding
- generalized hop-based or graph-aware token exposure
- generalized corroborating-source attribution beyond Tier 1

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
- non-stablecoin coverage
- bridge-aware or generalized graph scoring
- generalized corroborating-source attribution beyond Tier 1

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
- does meaningful attributed graph coverage support a bounded graph rollup?

### What is not implemented yet for Bitcoin

- rapid-spend scoring from output lifespan
- dormant-output reactivation
- peel-chain logic
- change-aware or generalized cluster-aware modeling
- generalized multi-hop exposure scoring
- broader corroborating attribution coverage beyond the current bounded fixtures

---

## What Attribution and Graph Context Change Today

The current attribution, actor/exposure, and bounded graph layer improves score precision in five ways:

1. risky actors can escalate scores without replacing behavioral reasons
2. contextual infrastructure can reduce false positives for broad-surface or high-volume wallets
3. reports can explain why a label mattered, including actor, confidence, and source tier
4. strong actor support can refine repeated interaction, concentration, and sampled exposure narratives
5. meaningful attributed graph coverage can add bounded graph summary and bounded motif-based score refinement

This is deliberately narrower than a full attribution or graph platform.

Tier 1 today means:

- GraphSense-style structured labels
- Bitcoin mining-pool context
- repo-local bootstrap overrides

Secondary corroboration adds:

- WalletExplorer-style secondary attribution support
- repo-safe corroborating fixture sources
- explicit corroborating vs conflicting source handling

The current actor/exposure and graph layer adds:

- actor-aware repeated-interaction and concentration refinement
- practical direct and near exposure summaries
- bounded pass-through and U-turn findings tied to attributed actors
- bounded graph summary rollups
- bounded graph motifs and bounded graph-aware score changes

Still future work:

- generalized graph-aware attribution paths
- broader corroborating-source ingestion and conflict arbitration
- richer graph coverage for currently sparse-attribution cases

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
- build generalized arbitrary-hop exposure graphs
- merge cluster identities across arbitrary address sets
- compute value-aware path risk across protocols

### Future live or graph-aware scoring

Future work includes:

- broader graph coverage and richer graph-derived scoring
- value-weighted concentration
- fresh-wallet plus immediate large-flow reasoning
- richer Solana and Bitcoin live-flow reasoning
- Optimism event-aware or selector-plus-event enrichment once cost/ROI supports it

---

## Related Documents

- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/EVM-CALLS-INTEGRATION.md`](EVM-CALLS-INTEGRATION.md)
- [`docs/ERC20-DATA-MODEL.md`](ERC20-DATA-MODEL.md)
- [`docs/SOLANA-DATA-MODEL.md`](SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
