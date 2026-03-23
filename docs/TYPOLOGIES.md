# Crypto Profiler Typology Catalog

## Purpose

This document defines the initial typology catalog for Crypto Profiler.

It serves four purposes:

1. provide a shared vocabulary for wallet-risk patterns
2. document which patterns are already implemented versus planned
3. make scoring behavior easier to explain to reviewers
4. guide future roadmap work for exposure analysis and behavioral detection

This is not intended to be a complete financial-crime taxonomy.  
It is an MVP-focused catalog aligned to the current goals of Crypto Profiler.

---

## Typology Status Legend

- **Implemented** — currently supported in the scoring or screening engine
- **Partially Implemented** — some supporting logic exists, but the typology is not yet fully modeled
- **Planned** — explicitly intended, but not yet implemented
- **Placeholder** — documented for future design alignment only

---

## Currently Demonstrated in the Repository

The current repository already demonstrates these typology-aligned behaviors:

- sanctions short-circuit decisioning
- profiled-address high-risk label handling
- direct mixer exposure handling
- contextual mitigation for established wallets without reinforcing suspicious behavior
- fresh-wallet escalation
- velocity burst detection
- noisy inbound / dusting-like observation
- high counterparty fan-in observation
- repeated flagged-counterparty interaction
- service concentration to high-risk or trusted services
- trusted protocol contextual mitigation
- dataset-backed profiling using curated case artifacts
- address-scoped EVM trace extraction for internal-call context

This section is intentionally limited to behaviors that are already visible in the current MVP codebase, tests, or sample outputs.

---
## Bitcoin Layer 1 coverage note

The current Bitcoin MVP slice is **UTXO-flow based**.

This means current Bitcoin coverage is strongest for:

- repeated interaction with counterparties
- spend-heavy operational hub behavior
- noisy inbound broad-surface behavior
- mixed-flow broad-value wallet behavior
- high-volume address-level UTXO movement

It is currently weaker for:

- entity clustering across related addresses
- change-address attribution
- full transaction graph reconstruction
- advanced peel-chain and multi-hop flow semantics

## Solana Layer 1 coverage note

The current Solana MVP slice is **stablecoin-flow based**.

This means current Solana coverage is strongest for:

- repeated interaction with counterparties
- concentration to major counterparties
- authority-driven operational control patterns
- stablecoin-heavy value movement
- broad counterparty surface / noisy authority behavior

It is currently weaker for:

- full instruction-aware program behavior
- protocol-specific routing semantics
- deeper non-stablecoin Solana activity

## Current multi-chain Layer 1 summary

### Ethereum
- trace-aware counterparty and protocol interaction coverage

### Solana
- stablecoin-flow-aware counterparty and authority-role coverage

### Bitcoin
- UTXO-flow-aware inbound / outbound / repeated-counterparty coverage

---

## 1. Sanctions / Watchlist Hit

**Status:** Implemented

### Description
A wallet is directly identified as sanctioned or watchlisted through the watchlist engine or equivalent exact-match screening.

### Why it matters
This is a deterministic compliance event and should short-circuit heuristic scoring.

### Current behavior
- direct watchlist check through watchlist engine
- exact-match sanctioned address detection
- immediate escalation to critical outcome

### Implemented reason codes
- `direct_sanctions_match`

### Expected outcome
- `risk_score = 100`
- `risk_grade = "CRITICAL (Sanctioned)"`
- `review_recommended = true`

### Representative case study
- [`docs/case-studies/direct-sanctioned-wallet.md`](docs/case-studies/direct-sanctioned-wallet.md)

---

## 2. Direct High-Risk Counterparty Exposure

**Status:** Implemented

### Description
A wallet directly interacts with a labeled high-risk counterparty such as:
- mixer infrastructure
- exploit wallet
- scam wallet
- other flagged entity

### Why it matters
Direct interaction is usually the strongest non-sanctions signal available in transaction-led KYW analysis.

### Current behavior
- checks direct counterparties against known labels
- applies risk score adjustments based on label category
- preserves explainable evidence in `risk_reasons`

### Implemented reason codes
- `direct_mixer_interaction`
- `profiled_address_high_risk_label`
- `direct_high_risk_entity`

### Notes
This typology is already central to the current scoring model.

### Representative case study
- [`docs/case-studies/established-wallet-mixer-no-reinforcing-signals.md`](docs/case-studies/established-wallet-mixer-no-reinforcing-signals.md)

---

## 3. Indirect Exposure Within N Hops

**Status:** Planned

### Description
A wallet is not directly interacting with a risky entity, but sits within a limited number of graph hops from a known mixer, sanctioned wallet, exploit wallet, or scam cluster.

### Why it matters
Many laundering and obfuscation patterns rely on short graph distance rather than direct repeated interaction.

### Planned direction
- start with 1-hop and 2-hop exposure
- weight risk by hop distance
- preserve path evidence for explainability

### Candidate future reason codes
- `hop_1_high_risk_exposure`
- `hop_2_high_risk_exposure`
- `weighted_indirect_exposure`

### Dependencies
- stronger graph traversal support
- richer extracted datasets
- Bitcoin outputs + Ethereum calls / traces for better flow reconstruction

---

## 4. Velocity Burst

**Status:** Implemented

### Description
A wallet exhibits unusually high activity over a short time window.

### Why it matters
High-frequency short-window activity can indicate bot-like behavior, burst laundering, pass-through movement, or unusual activation patterns.

### Current behavior
- measures transaction rate relative to wallet lifetime/activity window
- applies elevated fraud scoring when thresholds are crossed

### Implemented reason codes
- `high_velocity_behavior`

### Notes
This typology is currently simple and threshold-based.
It can be improved later with rolling-window or z-score logic.

---

## 5. Peel Chain Behavior

**Status:** Planned

### Description
Funds move through a sequence of wallets where value is repeatedly split or partially forwarded, often retaining a remainder and sending a fresh fraction onward.

### Why it matters
Peel chains are a known laundering and structuring pattern used to create distance and obscure ultimate flow.

### Planned direction
- sequence detection across chained outbound transfers
- repeated partial outflow pattern recognition
- graph/path evidence retained in output

### Candidate future reason codes
- `peel_chain_pattern`
- `layering_sequence_detected`

### Dependencies
- graph/path traversal
- temporal ordering across linked wallets
- improved Bitcoin outputs and Ethereum traces support

---

## 6. Mixer / Tumbler Exposure

**Status:** Partially Implemented

### Description
A wallet directly or indirectly interacts with mixer/tumbler infrastructure.

### Why it matters
Mixer usage is one of the most important crypto-risk patterns for AML and KYW workflows.

### Current behavior
- direct mixer interaction detection is supported
- profiled-address mixer label support is supported
- contextual mitigation is supported for established wallets without reinforcing suspicious behavior

### Implemented reason codes
- `direct_mixer_interaction`
- `profiled_address_high_risk_label`
- `combo_contextual_mitigation_established_wallet`

### Candidate future reason codes
- `hop_to_mixer_exposure`
- `weighted_mixer_exposure`

### Planned future extension
- indirect mixer exposure
- hop-to-mixer scoring
- volume- and recency-aware mixer exposure weighting

### Representative case study
- [`docs/case-studies/established-wallet-mixer-no-reinforcing-signals.md`](docs/case-studies/established-wallet-mixer-no-reinforcing-signals.md)

---

## 7. Newly Created Wallet with Immediate Large Flow

**Status:** Partially Implemented

### Description
A newly created or newly observed wallet begins moving value quickly after first appearance, especially at large volume.

### Why it matters
Fresh-wallet + immediate flow is often a strong risk indicator, especially when paired with:
- high velocity
- mixer interaction
- rapid outflow
- limited historical context

### Current behavior
- fresh-wallet detection is implemented
- fresh-wallet + mixer combination logic is implemented
- fresh-wallet + velocity combination logic can already compound risk indirectly

### Implemented reason codes
- `fresh_wallet`
- `combo_mixer_plus_fresh_wallet`

### Candidate future reason codes
- `new_wallet_large_initial_flow`
- `new_wallet_immediate_cashout`

### Planned future extension
- explicit large-value thresholding
- first-seen to first-large-outflow timing
- service concentration after creation

---

## 8. Round-Trip / U-Turn Behavior

**Status:** Planned

### Description
Funds move into a wallet and back out to the same or related destination within a short interval, suggesting transit-only usage rather than organic wallet behavior.

### Why it matters
U-turn behavior can indicate pass-through laundering, routing, rapid cash movement, or flow obfuscation.

### Planned direction
- identify quick inbound → outbound relationships
- correlate counterparties and timing
- preserve temporal evidence in output

### Candidate future reason codes
- `round_trip_behavior`
- `u_turn_flow_detected`
- `rapid_passthrough_behavior`

### Dependencies
- more advanced temporal linkage logic
- counterparty correlation
- graph-aware sequencing
- trace-aware internal-flow summaries for EVM

---

## 9. Concentration Risk to a Single Service

**Status:** Partially Implemented

### Description
A large share of wallet activity is concentrated to a single exchange, mixer, protocol, or other entity.

### Why it matters
Concentration can be benign or risky depending on the service type, but it often provides strong contextual information about wallet purpose.

### Current behavior
- measures concentration to the top labeled service counterparty
- distinguishes high-risk service concentration from trusted or exchange concentration
- applies fraud or reputation impact depending on service type
- trace-aware summaries are now merged into curated cases and surfaced in dataset mode, although live concentration scoring is still transaction-driven


### Implemented reason codes
- `high_risk_service_concentration`
- `exchange_concentration`
- `single_service_concentration`

### Planned future extension
- value-weighted concentration, not just interaction-count concentration
- recency-aware service concentration
- concentration using trace-aware internal-call counterparties
- concentration by cluster/entity, not only by exact address

### Dependencies
- better service/entity labeling
- counterparty aggregation across extracted datasets
- later trace-aware or graph-aware weighting improvements

---

## 10. Repeated Interaction with Flagged Counterparties

**Status:** Implemented

### Description
A wallet repeatedly interacts with one or more flagged counterparties over time, rather than showing a single isolated exposure.

### Why it matters
Repeated interaction is often stronger than a one-off contact and can indicate persistent operational linkage.

### Current behavior
- counts repeated interactions to flagged entities
- applies stronger scoring than one-off isolated interaction
- supports explainable evidence counts in `risk_reasons`

### Implemented reason codes
- `repeated_flagged_counterparty_interaction`

### Candidate future reason codes
- `persistent_high_risk_counterparty_exposure`

### Planned future extension
- recency-aware repeated interaction scoring
- cross-address cluster/entity repeated interaction
- trace-aware repeated internal-call interaction patterns

---

## 11. High Counterparty Fan-In

**Status:** Implemented

### Description
A wallet receives inbound transfers from a large number of unique counterparties.

### Why it matters
High fan-in can indicate spam targeting, dusting-like activity, public-address exposure, or service-like behavior.

### Current behavior
- used as a low-severity observed signal
- intended to surface unusual inbound structure without over-escalating benign public-wallet cases

### Implemented reason codes
- `high_counterparty_fan_in`

### Notes
This is currently an observational signal, not an automatic review trigger.

### Representative case study
- `docs/case-studies/public-wallet-noisy-inbound.md` *(recommended next addition)*

---

## 12. Noisy Inbound / Dusting-Like Observation

**Status:** Implemented

### Description
A wallet shows mostly inbound activity from many senders, often with frequent zero-value or negligible-value transfers.

### Why it matters
This can reflect:
- public-address spam
- airdrop / dusting behavior
- low-signal noisy activity that analysts may still want to understand

### Current behavior
- detects high inbound ratio
- detects many unique senders
- detects repeated zero-value inbound transfers
- keeps output at observed / low-severity posture unless reinforced by stronger risk signals

### Implemented reason codes
- `noisy_inbound_activity`
- `zero_value_inbound_pattern`

### Notes
This typology intentionally improves explainability without creating unnecessary false positives.

### Representative case study
- `docs/case-studies/public-wallet-noisy-inbound.md` *(recommended next addition)*

---

## 13. Trusted Protocol or Exchange Context

**Status:** Implemented

### Description
A wallet is labeled as a known exchange or trusted protocol, or directly interacts with one.

### Why it matters
Trusted context helps reduce false positives and gives necessary business context to otherwise high-volume activity.

### Current behavior
- trusted/protocol and exchange labels can reduce risk
- profiled address trust context is supported
- trusted context functions as **contextual mitigation**, not as an override of sanctions or stronger fraud signals

### Implemented reason codes
- `profiled_address_trusted_label`
- `exchange_interaction`
- `trusted_or_protocol_interaction`

### Notes
A trusted or exchange label does **not** make a wallet automatically safe.  
It is contextual rather than absolute.

### Representative case study
- `docs/case-studies/trusted-protocol-high-activity-router.md` *(recommended next addition)*

---

## 14. Cross-Chain / Chain-Hopping Obfuscation

**Status:** Placeholder

### Description
Funds are moved across multiple chains, bridges, swaps, or settlement rails to create investigative distance from the originating wallet or event.

### Why it matters
Cross-chain movement is increasingly relevant to sanctions evasion, exploit laundering, and scam cash-out behavior.

### Placeholder direction
- bridge-aware flow modeling
- cross-chain risk handoff
- path evidence across chains

### Candidate future reason codes
- `cross_chain_obfuscation`
- `bridge_hop_risk`
- `cross_chain_settlement_pattern`

---

## 15. Stablecoin Sanctions-Evasion Corridor

**Status:** Placeholder

### Description
A wallet or flow path relies heavily on stablecoins as the settlement mechanism for sanctions evasion, laundering, or rapid cross-service movement.

### Why it matters
Stablecoins can compress settlement time and are commonly used in higher-risk flows because of their liquidity and transfer efficiency.

### Placeholder direction
- stablecoin-heavy flow ratios
- stablecoin settlement to flagged services
- repeated stablecoin transfers across high-risk corridors

### Candidate future reason codes
- `stablecoin_sanctions_evasion_corridor`
- `stablecoin_high_risk_settlement`
- `stablecoin_dominant_flagged_flow`

---

## 16. Drainer / Approval-Phishing Cluster

**Status:** Placeholder

### Description
Wallets or clusters repeatedly appear in token-drain or approval-phishing patterns, often linked to repeated approval abuse and rapid asset extraction from many victims.

### Why it matters
Approval-based drainers remain a major crypto-fraud pattern and can create repeatable downstream cash-out behavior.

### Placeholder direction
- cluster-level repeated victim interaction
- suspicious approval-driven outflow chains
- scam settlement wallet linkage

### Candidate future reason codes
- `approval_phishing_cluster`
- `drainer_cashout_pattern`
- `repeat_victim_settlement_wallet`

---

## 17. Scam Settlement / Pig-Butchering Cash-Out Cluster

**Status:** Placeholder

### Description
Wallets receive repeated inbound funds from many victims and route them toward cash-out or laundering services in patterns consistent with scam settlement infrastructure.

### Why it matters
Many scam operations use repeatable settlement and cash-out infrastructure, which can become detectable through fan-in, concentration, and flagged service exposure.

### Placeholder direction
- high fan-in scam settlement patterns
- repeated scam-linked counterparty flows
- structured cash-out routes

### Candidate future reason codes
- `scam_settlement_cluster`
- `pig_butchering_cashout_pattern`
- `repeat_scam_counterparty_flow`

---

## 18. Terrorism-Financing / Sanctions-Evasion Proximity

**Status:** Placeholder

### Description
A wallet is not itself sanctioned, but shows proximity or behavioral overlap with addresses, services, or corridors relevant to terrorism financing or sanctions-evasion typologies.

### Why it matters
Not all risk appears as an exact sanctions match. Proximity and repeated association can still matter in higher-scrutiny review contexts.

### Placeholder direction
- proximity scoring to sanctioned clusters
- repeated interaction with sanctioned-adjacent services
- stronger watchlist + exposure correlation

### Candidate future reason codes
- `terrorism_financing_proximity`
- `sanctions_evasion_proximity`
- `high_risk_sanctions_adjacent_exposure`

---

## Current Typology Coverage Summary

### Implemented
- sanctions / watchlist hit
- direct high-risk counterparty exposure
- velocity burst
- repeated flagged-counterparty interaction
- noisy inbound / dusting-like observation
- high counterparty fan-in
- trusted protocol / exchange context

### Partially Implemented
- newly created wallet with immediate flow
- mixer / tumbler exposure broader than direct contact
- concentration risk to a single service

### Planned
- indirect exposure within N hops
- peel chain behavior
- round-trip / U-turn behavior

### Placeholder
- cross-chain / chain-hopping obfuscation
- stablecoin sanctions-evasion corridor
- drainer / approval-phishing cluster
- scam settlement / pig-butchering cash-out cluster
- terrorism-financing / sanctions-evasion proximity

---

## Typology Design Principles

The typology catalog is guided by the following principles:

1. **Deterministic-first where appropriate**  
   Sanctions and exact-match watchlist hits should override weaker heuristic interpretation.

2. **Explainability over opacity**  
   Every typology should map cleanly to visible reason codes and evidence.

3. **Context matters**  
   A single mixer interaction does not always imply a failing outcome.  
   Repeated, correlated, or reinforced signals matter more.

4. **Observed does not always mean escalated**  
   Some behaviors should be surfaced as context without forcing review.

5. **Portfolio-grade realism**  
   The catalog should support practical investigator storytelling, not just synthetic scoring logic.

---

## Future Expansion

This catalog is expected to grow with:

- Bitcoin output-aware flow typologies
- Ethereum trace-aware internal-flow typologies
- trace-aware dataset enrichment for curated case reasoning
- stronger 1-hop / 2-hop graph exposure
- recency-aware scoring
- value-aware service concentration
- cluster/entity-level repeated interaction analysis
- cross-chain and bridge-aware behavior in later phases

---

## Related Documents

- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`README.md`](../README.md)
- [`docs/case-studies/established-wallet-mixer-no-reinforcing-signals.md`](case-studies/established-wallet-mixer-no-reinforcing-signals.md)
- [`docs/case-studies/direct-sanctioned-wallet.md`](case-studies/direct-sanctioned-wallet.md)
- `docs/case-studies/public-wallet-noisy-inbound.md` *(recommended next addition)*
- `docs/case-studies/trusted-protocol-high-activity-router.md` *(recommended next addition)*
