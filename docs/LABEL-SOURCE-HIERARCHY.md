# Attribution Source Hierarchy

## Purpose

This document explains the attribution layer implemented in the repository today.

The goal is practical precision:

- keep the behavioral model as the primary scoring engine
- add a deterministic Tier 1 attribution layer on top
- allow bounded secondary corroboration without letting weak sources dominate
- distinguish risk-escalating labels from contextual or benign labels
- improve analyst explanations without pretending the repo already has full entity resolution
- add bounded actor-aware and exposure-aware refinement where attribution support is strong

---

## What Tier 1 Means In This Repo

Tier 1 in Crypto Profiler means high-trust attribution inputs that are treated as primary context for scoring and reports.

Primary attribution in this repo means Tier 1 sources that can anchor the resolved decision.

Primary attribution currently includes:

- GraphSense-style structured entity labels
- Bitcoin mining-pool context
- repo-local bootstrap labels used as local continuity and demo overrides

Secondary corroboration currently includes:

- WalletExplorer-style secondary attribution support
- repo-safe corroborating fixture inputs
- confidence boosts from corroboration
- conflict visibility in resolved attribution and report output

The current actor/exposure layer adds:

- actor-aware repeated-interaction and concentration refinement
- practical cluster-aware grouping when multiple sampled addresses resolve to the same actor
- direct and near exposure summaries tied to attributed actors
- bounded pass-through and U-turn findings from sampled Layer 1 flows

The current implementation still does not include:

- broad corroborating-source ingestion
- large-scale conflict arbitration across many providers
- generalized cluster-wide entity resolution

---

## Internal Concepts

The attribution layer normalizes source data into a shared shape:

- `AttributionSourceMetadata`
- `AttributionRecord`
- `ResolvedAttribution`

Each normalized record carries:

- source name
- source tier
- source type
- category
- risk class
- confidence
- supporting vs conflicting evidence
- actor, when available
- whether the label is contextual or risk-escalating

This keeps the code practical and testable while still making room for later expansion.

---

## Source Hierarchy

### 1. Local override

Local override is the highest-priority source tier.

Current use:

- `data/labels/bootstrap_entities.json`

Why it exists:

- continuity with earlier repo behavior
- deterministic demo behavior
- ability to keep a small number of repo-local benchmark labels authoritative

Local override can suppress or replace a primary structured label when the repo intentionally wants a different final interpretation.

### 2. Primary structured

Primary structured sources are high-trust, structured attribution inputs.

Current primary sources:

- `data/labels/tier1_graphsense_entities.json`
- `data/labels/tier1_bitcoin_mining_pools.json`

These are the first non-demo attribution inputs used by the resolver.

### 3. Secondary corroborating

Secondary corroborating sources sit below Tier 1.

Current secondary sources:

- `data/labels/tier2_wallet_explorer_entities.json`
- `data/labels/tier2_corroborating_entities.json`

These sources may:

- support a Tier 1 decision
- raise confidence modestly
- add analyst-facing context
- surface conflicts

They do not override a stronger Tier 1 result on their own.

---

## Current Source Types

### GraphSense-style structured labels

Used for:

- labeled actors such as trusted protocols, exchanges, mixers, or other named infrastructure
- explicit actor names where available
- structured categories that can map to risk classes

### Bitcoin mining-pool context

Used for:

- mining-pool attribution that is operationally important but usually contextual rather than risk-escalating
- reducing false positives for Bitcoin addresses that look operational but are better explained as pool infrastructure

### Bootstrap local labels

Used for:

- continuity with the existing repo story
- small deterministic overrides for demos and tests
- trusted or high-risk benchmark addresses already used elsewhere in the codebase

### WalletExplorer-style secondary labels

Used for:

- lower-trust corroborating service or exchange-style context
- bounded secondary-only attribution when no Tier 1 source exists
- analyst-facing explanation without large score jumps

### Repo-safe corroborating fixture labels

Used for:

- small cross-chain corroboration scenarios in tests and demos
- confidence uplift and conflict visibility
- deterministic secondary-source examples that are safe to commit

---

## Resolution Rules

The resolver is deterministic.

It chooses the final attribution decision by preferring:

1. higher source tier
2. higher confidence
3. risk-escalating over contextual when all else is equal
4. stable lexical tie-breaks

The resolver returns:

- resolved label
- actor
- category
- risk class
- confidence
- source metadata
- whether the result is contextual or risk-escalating
- supporting source list
- corroborating source list
- conflicting source list

This means the project does not collapse all labels into one generic “known entity” flag.

Secondary-source behavior is intentionally bounded:

- a lone secondary source can resolve, but remains low-confidence
- a secondary source can corroborate a Tier 1 result and raise confidence modestly
- a secondary source can conflict with Tier 1 and appear in the report
- a secondary source does not override a stronger Tier 1 source

---

## Contextual vs Risk-Escalating Labels

This distinction is central to the current scoring design.

Risk-escalating examples:

- sanctioned actor
- mixer
- exploit or scam infrastructure

Contextual or benign examples:

- trusted protocol router
- known exchange service wallet
- mining pool
- treasury-like operational address

The scoring layer uses that distinction to avoid a common false-positive problem:

- broad or noisy activity should not always escalate if a strong Tier 1 contextual explanation exists

---

## How Attribution Affects Scoring

The existing behavioral model remains intact.

The order is:

1. analyze behavior
2. resolve Tier 1 attribution
3. consider bounded secondary corroboration or conflicts
4. apply bounded actor-aware or exposure-aware refinement when attribution support is strong
5. apply a controlled attribution modifier
6. render a report that shows both behavior and attribution context

Current practical effects:

- sanctioned or illicit-service Tier 1 attribution strongly escalates the final score
- trusted protocol, exchange, mining-pool, or treasury Tier 1 attribution reduces false positives
- corroborating secondary sources can slightly raise confidence or add modest bounded adjustments
- conflicting secondary sources can surface note-level caution without overriding Tier 1
- actor-aware rollups can merge repeated interaction or concentration to the actor level when attribution confidence is strong enough
- pass-through or U-turn findings remain bounded and explanation-first
- attribution does not replace behavioral reasoning or remove visible risk reasons

---

## What Remains Future Work

The current attribution layer is a practical MVP, not a full graph platform.

Still deferred:

- generalized actor-aware graph scoring across arbitrary hops
- large-scale cluster resolution beyond the current sampled actor grouping
- richer value-aware path scoring and protocol-intent reconstruction
- broader conflict arbitration beyond the current bounded source rules

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/DATA-SOURCING-POLICY.md`](DATA-SOURCING-POLICY.md)
