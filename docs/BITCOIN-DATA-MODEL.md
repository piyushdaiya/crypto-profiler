# Crypto Profiler Bitcoin Data Model

## Purpose

This document defines the Bitcoin data model for Crypto Profiler.

It explains:

- which Blockchair Bitcoin datasets are used
- how those datasets relate to each other
- how Bitcoin data should be normalized internally
- what Bitcoin behaviors the MVP can support today
- what remains out of scope until later phases

This document is intentionally MVP-focused and aligned to the current Crypto Profiler architecture.

---

## Scope

The Bitcoin model in Crypto Profiler is designed to support:

- wallet activity profiling
- sanctions and labeled-address screening
- repeated interaction analysis
- concentration analysis
- pass-through and rapid-spend observations
- future 1-hop and 2-hop exposure logic
- future peel-chain and UTXO behavior analysis

This model is **not** intended to be a full Bitcoin graph warehouse or full blockchain indexer.

---

## MVP Data Window

### Canonical MVP window

The current recommended Bitcoin MVP window is:

- **2025-03-16 → 2025-06-17**

### Why this window

This is the clean overlapping window currently supported by:

- Bitcoin transactions
- Bitcoin inputs
- Bitcoin outputs

This gives Crypto Profiler a consistent 90+ day Bitcoin slice for:

- transaction summaries
- UTXO creation
- UTXO spend linkage
- behavior and exposure heuristics

---

## Source Datasets

Crypto Profiler uses three primary Bitcoin datasets from Blockchair.

### 1. Bitcoin Transactions

This is the transaction-level summary table.

Representative fields:

- `block_id`
- `hash`
- `time`
- `size`
- `weight`
- `version`
- `lock_time`
- `is_coinbase`
- `has_witness`
- `input_count`
- `output_count`
- `input_total`
- `output_total`
- `fee`
- `fee_per_kb`
- `fee_per_kwu`
- `cdd_total`

### What it is used for

- tx-level metadata
- fee analysis
- fan-in / fan-out summaries
- transaction density / burst analysis
- future Bitcoin behavioral rules

---

### 2. Bitcoin Outputs

This is the UTXO creation table.

Representative fields:

- `block_id`
- `transaction_hash`
- `index`
- `time`
- `value`
- `value_usd`
- `recipient`
- `type`
- `script_hex`
- `is_from_coinbase`
- `is_spendable`

### What it is used for

- recipient analysis
- output concentration
- identifying created UTXOs
- output-type context
- future service concentration modeling
- future change-like output analysis

---

### 3. Bitcoin Inputs

This is the spent-output linkage table.

Representative fields:

- `block_id`
- `transaction_hash`
- `index`
- `time`
- `value`
- `value_usd`
- `recipient`
- `type`
- `script_hex`
- `is_from_coinbase`
- `is_spendable`
- `spending_block_id`
- `spending_transaction_hash`
- `spending_index`
- `spending_time`
- `spending_value_usd`
- `spending_sequence`
- `spending_signature_hex`
- `spending_witness`
- `lifespan`
- `cdd`

### What it is used for

- UTXO spend linkage
- time-to-spend behavior
- dormant output reactivation
- rapid-spend observation
- future peel-chain and pass-through analysis

---

## Conceptual Model

Bitcoin modeling in Crypto Profiler is based on three levels:

### 1. Transaction layer
This is the summary layer.

Questions answered:
- how large was the transaction?
- how many inputs and outputs did it contain?
- what fee did it pay?
- was it coinbase?
- how dense / bursty is the wallet’s observed activity?

### 2. Output layer
This is the created-UTXO layer.

Questions answered:
- which outputs were created?
- which addresses received value?
- how much value was assigned to each output?
- what script type and recipient type were used?

### 3. Spend-linkage layer
This is the consumed-UTXO layer.

Questions answered:
- when was the output later spent?
- how long did it remain unspent?
- which transaction consumed it?
- does spending behavior suggest pass-through, dormancy, or layering?

---

## Join Strategy

The Bitcoin model depends on joining datasets in a precise way.

### Primary output identity

An output is uniquely identified by:

- `transaction_hash`
- `index`

This pair is the UTXO identity.

### Join rules

#### Transactions → Outputs
Join on:

- `transactions.hash = outputs.transaction_hash`

This links tx-level metadata to created outputs.

#### Outputs → Inputs
Join on:

- `outputs.transaction_hash = inputs.transaction_hash`
- `outputs.index = inputs.index`

This links a created output to its later spend behavior.

### Important note

In the Blockchair inputs table, the `transaction_hash` and `index` identify the **original output being spent**, not the spending transaction.

That means the input row is effectively a “spent UTXO record,” which is ideal for lifecycle modeling.

---

## Internal Normalized Model

Crypto Profiler should normalize Bitcoin data into common internal shapes.

### BitcoinTransactionSummary

Suggested conceptual fields:

- `TxHash`
- `BlockID`
- `Timestamp`
- `InputCount`
- `OutputCount`
- `InputTotalSats`
- `OutputTotalSats`
- `FeeSats`
- `IsCoinbase`
- `HasWitness`
- `Weight`
- `CDDTotal`

### BitcoinOutput

Suggested conceptual fields:

- `TxHash`
- `OutputIndex`
- `BlockID`
- `Timestamp`
- `ValueSats`
- `Recipient`
- `ScriptType`
- `ScriptHex`
- `IsCoinbase`
- `IsSpendable`

### BitcoinSpentOutput

Suggested conceptual fields:

- `SourceTxHash`
- `SourceOutputIndex`
- `SourceTimestamp`
- `ValueSats`
- `Recipient`
- `ScriptType`
- `SpendingTxHash`
- `SpendingBlockID`
- `SpendingTimestamp`
- `LifespanSeconds`
- `CDD`

### BitcoinAddressActivity

Suggested conceptual fields:

- `Address`
- `FirstSeen`
- `LastSeen`
- `InboundCount`
- `OutboundCount`
- `UniqueCounterparties`
- `TotalReceivedSats`
- `TotalSentSats`
- `LinkedOutputs`
- `LinkedSpends`

---

## Address Semantics

Bitcoin differs from account-based chains like Ethereum.

### Important constraints

Bitcoin addresses are not wallet identities.

An address may be:
- reused
- single-use
- change
- part of a cluster not visible from address-only data

### MVP interpretation rule

For MVP, Crypto Profiler will treat Bitcoin recipient addresses as **address-level entities**, not guaranteed wallet-level identities.

That means:
- exact-match screening is valid
- concentration analysis is still useful
- repeated interaction analysis is still useful
- but deeper wallet clustering should be treated as future work

---

## What the MVP Can Support Today

With transactions + outputs + inputs, the Bitcoin model can support:

### Implementable soon
- labeled address exact-match screening
- repeated interaction with flagged counterparties
- concentration to one recipient/service
- newly active address with immediate flow
- rapid spend / short-lifespan outputs
- dormant output reactivation observations
- simple fan-in / fan-out summaries

### Partially supportable
- pass-through behavior
- round-trip candidate logic
- simple peel-chain candidate detection

### Not yet supportable at production quality
- full wallet clustering
- strong change detection
- large-scale graph expansion
- confidence-weighted multi-hop attribution

---

## Recommended Bitcoin Heuristics

The following Bitcoin heuristics are good candidates for the next implementation phases.

### 1. Repeated flagged recipient interaction
A profile repeatedly creates or spends outputs linked to flagged addresses.

### 2. Concentration to a single recipient/service
A large portion of observed value goes to one address or one labeled service.

### 3. Newly active address with immediate spend
Outputs are created and spent rapidly soon after first observation.

### 4. Short-lifespan spend pattern
Outputs are repeatedly spent within a very small time window.

### 5. Dormant reactivation
Aged outputs are suddenly spent after a long inactive period.

### 6. Peel-chain candidate behavior
Repeated partial forwarding or structured output splitting appears across linked spends.

---

## Data Quality and Modeling Notes

### 1. Output identity is reliable
The `transaction_hash + index` pair is a strong primary key for UTXO-level modeling.

### 2. USD values are auxiliary
`value_usd` and `spending_value_usd` are useful for explanation and demos, but Crypto Profiler should treat satoshi values as the primary data source.

### 3. Coinbase outputs need special handling
Coinbase-related outputs should not be interpreted the same way as normal transactional behavior.

### 4. Null / nonstandard script handling
Script types should be preserved, but the MVP should avoid overfitting behavior to rare output types too early.

### 5. Address reuse assumptions must be cautious
Repeated appearance of the same address is useful operationally, but should not automatically imply one real-world entity cluster.

---

## Recommended Extraction Strategy

For the MVP, the Bitcoin extraction path should:

1. limit itself to the canonical window:
    - `2025-03-16 → 2025-06-17`

2. load:
    - transactions
    - outputs
    - inputs

3. normalize:
    - transaction summaries
    - created outputs
    - spend linkages

4. construct:
    - address-level summaries
    - top counterparties
    - spend timing observations
    - service concentration candidates

5. emit:
    - extracted Bitcoin case datasets
    - curated Bitcoin case artifacts
    - explanation-ready summaries

---

## Suggested Repo Outputs

### Raw extracted layer
Examples:
- `data/cases/extracted-btc/<address>.json`

### Curated layer
Examples:
- `data/cases/curated/<case-id>.json`

### Future docs/case studies
Examples:
- `docs/case-studies/bitcoin-rapid-spend-observation.md`
- `docs/case-studies/bitcoin-service-concentration.md`

---

## Relationship to Scoring Engine

The Bitcoin data model is designed to feed the same scoring engine concepts already used by EVM cases.

That means Bitcoin-derived reasons should eventually map into familiar categories like:

- `FRAUD`
- `REPUTATION`
- `LENDING`

And reason codes should remain explainable, for example:

- `repeated_flagged_counterparty_interaction`
- `single_service_concentration`
- `rapid_spend_pattern`
- `dormant_output_reactivation`
- `new_address_immediate_flow`

This keeps chain-specific ingestion separate from chain-agnostic scoring and explanation.

---

## Out of Scope for the MVP

The following are intentionally deferred:

- full wallet clustering
- industrial-scale UTXO graph analytics
- transaction graph persistence in a dedicated database
- advanced coin control / change detection
- entity resolution beyond exact-match labels
- automated cross-chain Bitcoin bridge attribution

---

## Next Steps

### Immediate
- implement repeated interaction with flagged counterparties
- implement concentration risk to a single service
- design Bitcoin extracted dataset shape
- add Bitcoin case-study candidates

### Near-term
- add Bitcoin-specific extracted and curated cases
- add rapid-spend and dormant-reactivation heuristics
- add 1-hop / 2-hop Bitcoin exposure summaries

### Later
- peel-chain logic
- stronger temporal flow reasoning
- service clustering
- wallet clustering confidence model

---

## Related Documents

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`README.md`](README.md)
