# Tier 1 Label Source Hierarchy

## Purpose

This document explains the attribution layer implemented in Wave 5A.

The goal is practical precision:

- keep the behavioral model as the primary scoring engine
- add a deterministic Tier 1 attribution layer on top
- distinguish risk-escalating labels from contextual or benign labels
- improve analyst explanations without pretending the repo already has full entity resolution

---

## What Tier 1 Means In This Repo

Tier 1 in Crypto Profiler means high-trust attribution inputs that are treated as primary context for scoring and reports.

Wave 5A includes only:

- GraphSense-style structured entity labels
- Bitcoin mining-pool context
- repo-local bootstrap labels used as local continuity and demo overrides

Wave 5A does not include:

- WalletExplorer
- broad corroborating-source ingestion
- large-scale conflict arbitration across many providers
- cluster-wide entity resolution

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

Current Wave 5A sources:

- `data/labels/tier1_graphsense_entities.json`
- `data/labels/tier1_bitcoin_mining_pools.json`

These are the first non-demo attribution inputs used by the resolver.

---

## Source Types In Wave 5A

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

This means the project does not collapse all labels into one generic “known entity” flag.

---

## Contextual vs Risk-Escalating Labels

This distinction is central to Wave 5A.

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

Wave 5A keeps the existing behavioral model intact.

The order is:

1. analyze behavior
2. resolve Tier 1 attribution
3. apply a controlled attribution modifier
4. render a report that shows both behavior and attribution context

Current practical effects:

- sanctioned or illicit-service attribution strongly escalates the final score
- trusted protocol, exchange, mining-pool, or treasury attribution reduces false positives
- attribution does not replace behavioral reasoning or remove visible risk reasons

---

## What Is Deferred To Wave 5B And 5C

Wave 5A intentionally stops short of:

- WalletExplorer integration
- secondary or corroborating-source ingestion
- multi-source conflict arbitration beyond the current tier rules
- cluster-aware or actor-aware graph scoring
- path-aware exposure, pass-through, or U-turn attribution

Those are the next layers, not part of the current Tier 1 implementation.

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/DATA-SOURCING-POLICY.md`](DATA-SOURCING-POLICY.md)
