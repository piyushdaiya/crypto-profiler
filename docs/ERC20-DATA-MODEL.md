# Crypto Profiler ERC-20 Data Model

## Purpose

This document defines the ERC-20 data model for Crypto Profiler.

It explains:

- which ERC-20 datasets are used
- how transaction and token metadata are related
- how ERC-20 transfers should be normalized internally
- what ERC-20 behaviors the MVP can support today
- what remains out of scope until later phases

This document is intentionally MVP-focused and aligned to the current Crypto Profiler architecture.

Current status note:

- ERC-20 extraction groundwork exists in the repo.
- ERC-20 Layer 1 curated scoring and validator dataset support are not implemented yet.
- This document is therefore a design-and-groundwork reference for the next major execution wave.

---

## Scope

The ERC-20 model in Crypto Profiler is designed to support:

- token transfer profiling
- token-aware wallet activity summaries
- exact-match screening of token counterparties
- repeated interaction analysis
- concentration analysis
- stablecoin-heavy flow observations
- future internal transfer and swap-aware analysis when Ethereum calls are available

This model is **not** intended to be a full event-indexing platform or a complete DeFi decoding engine.

---

## MVP Data Window

### Canonical MVP window

The current recommended EVM MVP window is:

- **2025-03-16 → 2025-06-17**

### Why this window

This is the clean shared MVP window aligned to the current Bitcoin outputs coverage and already covered by:

- Ethereum transactions
- ERC-20 transactions
- future Ethereum calls download target

This gives Crypto Profiler one consistent cross-chain MVP slice for:
- curated case generation
- repeated interaction analysis
- concentration analysis
- future hop-based exposure work

### Token metadata window

ERC-20 token metadata does **not** need a 90-day history window.

For MVP:
- use the **latest token snapshot**
- refresh periodically if desired
- treat it as reference metadata, not behavioral history

---

## Source Datasets

Crypto Profiler uses two primary ERC-20 datasets from Blockchair.

### 1. ERC-20 Transactions

This is the transfer-event table.

Representative fields:

- `block_id`
- `transaction_hash`
- `time`
- `token_address`
- `token_name`
- `token_symbol`
- `token_decimals`
- `sender`
- `recipient`
- `value`

### What it is used for

- token transfer extraction
- token-aware activity summaries
- repeated interaction analysis
- concentration analysis
- token-level enrichment in curated cases
- stablecoin and asset-mix observations

### Important modeling note

A single Ethereum transaction may emit **multiple ERC-20 transfer rows**.

That means the ERC-20 table should be interpreted as:
- a table of **transfer events**
  not
- a table of unique transactions

This is critical for correct parsing and aggregation.

---

### 2. ERC-20 Tokens Latest Snapshot

This is the token metadata reference table.

Representative fields:

- `id`
- `address`
- `time`
- `name`
- `symbol`
- `decimals`
- `creating_block_id`
- `creating_transaction_hash`

### What it is used for

- token metadata lookup
- symbol/name/decimals normalization
- enrichment of extracted and curated case artifacts
- token-category mapping in future phases
- improved readability in analyst-facing outputs

### Important modeling note

This dataset is best treated as:
- reference metadata
- occasionally refreshed lookup material

It is **not** behavioral history.

---

## Conceptual Model

ERC-20 modeling in Crypto Profiler is based on two levels:

### 1. Transfer-event layer
This is the behavioral layer.

Questions answered:
- which token moved?
- between which addresses?
- when did it move?
- how many token transfers occurred?
- how concentrated is the wallet’s token activity?
- how often does it interact with flagged counterparties?

### 2. Token-metadata layer
This is the enrichment layer.

Questions answered:
- what token is this?
- what symbol and decimals should be used?
- is this a known stablecoin, trusted asset, or suspicious meme/spam token?
- how should raw integer token values be rendered for analysts?

---

## Join Strategy

The ERC-20 model depends on joining datasets in a precise way.

### Primary transfer event fields

A transfer event is not uniquely defined only by `transaction_hash`, because the same Ethereum transaction can contain many ERC-20 transfer rows.

For MVP, a practical event identity can be derived from:

- `transaction_hash`
- `token_address`
- `sender`
- `recipient`
- `value`

If needed later, a synthetic row identifier can be added during extraction.

### Join rules

#### ERC-20 Transactions → Token Metadata
Join on:

- `erc20_transactions.token_address = erc20_tokens.address`

This enriches transfers with:
- canonical token symbol
- canonical token name
- token decimals

### Priority rule for metadata

For display and enrichment:

1. prefer token metadata snapshot values
2. fall back to inline transaction row metadata
3. preserve raw values even if token metadata is missing or malformed

---

## Address and Token Normalization

### EVM address normalization

Blockchair ERC-20 data uses lowercase hexadecimal addresses without `0x`.

Crypto Profiler should normalize all ERC-20 addresses to:

- lowercase
- trimmed
- `0x`-prefixed form

This applies to:
- `token_address`
- `sender`
- `recipient`

### Token address normalization

Token contract addresses should be normalized the same way:
- lowercase
- trimmed
- `0x` prefix added

### Missing / malformed metadata

The model must tolerate:
- missing token name
- missing token symbol
- invalid or missing decimals
- unusual or spammy token metadata

The parser should never fail just because token metadata is imperfect.

---

## Value Semantics

### Raw value

The `value` field in ERC-20 transfers is the raw integer amount in base units.

Examples:
- USDC with `decimals = 6`
- WETH with `decimals = 18`

### Display value

For analyst-facing output, Crypto Profiler should compute:

- `display_value = raw_value / 10^decimals`

### Modeling rule

Always preserve:
- raw integer value
- decimals used for interpretation
- optionally a normalized human-readable value for output

The raw value is the authoritative source.

---

## Internal Normalized Model

Crypto Profiler should normalize ERC-20 data into common internal shapes.

### ERC20Transfer

Suggested conceptual fields:

- `BlockID`
- `TransactionHash`
- `Timestamp`
- `TokenAddress`
- `TokenName`
- `TokenSymbol`
- `TokenDecimals`
- `Sender`
- `Recipient`
- `RawValue`
- `DisplayValue`

### ERC20TokenMetadata

Suggested conceptual fields:

- `TokenAddress`
- `Name`
- `Symbol`
- `Decimals`
- `CreatedAt`
- `CreatingBlockID`
- `CreatingTransactionHash`

### ERC20AddressActivity

Suggested conceptual fields:

- `Address`
- `FirstSeen`
- `LastSeen`
- `InboundTransferCount`
- `OutboundTransferCount`
- `UniqueCounterparties`
- `UniqueTokensSeen`
- `TopTokensByTransferCount`
- `TopCounterparties`
- `StablecoinTransferRatio`
- `FlaggedCounterpartyInteractionCount`

---

## Activity Semantics

ERC-20 activity is event-driven and multi-asset.

### Important implications

A wallet can:
- receive many token transfers in one block
- interact with many tokens in one tx hash
- appear as both sender and recipient in complex router/swaps
- show stablecoin-heavy activity that differs from ETH-native behavior

This means ERC-20 activity should be summarized using:
- transfer-event counts
- token diversity
- top-token concentration
- top-counterparty concentration
- stablecoin exposure

---

## What the MVP Can Support Today

With ERC-20 transactions + token metadata, the ERC-20 model can support:

### Implementable soon
- token-aware wallet activity summaries
- repeated interaction with flagged counterparties
- concentration to a single service
- concentration to a single token
- stablecoin-heavy flow observation
- token metadata enrichment for curated cases

### Partially supportable
- suspicious token spam / airdrop patterns
- dusting-like token fan-in
- token settlement corridors

### Not yet supportable at production quality
- swap path reconstruction
- router-aware transfer interpretation
- contract-call intent analysis
- DeFi protocol semantic decoding
- bridge attribution
- internal transfer correlation without Ethereum calls

---

## Recommended ERC-20 Heuristics

The following ERC-20 heuristics are good candidates for the next implementation phases.

### 1. Repeated flagged counterparty interaction
A profile repeatedly sends tokens to or receives tokens from a flagged entity.

### 2. Concentration to a single service
A large share of ERC-20 transfer activity is concentrated to one labeled service.

### 3. Stablecoin-heavy flow
A wallet’s transfer activity is dominated by stablecoins or stablecoin settlement paths.

### 4. Token spam / airdrop fan-in
A wallet receives many inbound token transfers from many counterparties, especially low-signal or suspicious assets.

### 5. Single-token dominance
A wallet’s activity is heavily dominated by one asset, which can support concentration or corridor reasoning.

---

## Token Category Extension

The MVP should keep token-category logic simple, but the model should leave room for categorization such as:

- stablecoin
- wrapped native asset
- protocol governance token
- meme / speculative token
- gold / commodity-backed token
- unknown / uncategorized token

### Why this matters

Category tags can later support:
- stablecoin corridor detection
- protocol-context mitigation
- suspicious token spam detection
- more interpretable outputs

This can be implemented incrementally as a reference lookup layer.

---

## Data Quality and Modeling Notes

### 1. Inline metadata can be inconsistent
Transaction rows may contain token name/symbol/decimals, but the latest token snapshot should be treated as the preferred enrichment source.

### 2. Missing metadata is normal
Not all tokens will have clean symbols or names. The system should degrade gracefully.

### 3. Multi-transfer transactions are normal
A single tx hash may contain many transfer events across the same or different tokens.

### 4. Token value display must be cautious
Analyst-facing formatting should use decimals, but the engine should preserve the raw integer value for fidelity.

### 5. Stablecoin identification should be explicit
Do not guess stablecoins only from symbol text in scoring logic. Use a known reference list or metadata map later.

---

## Recommended Extraction Strategy

For the MVP, the ERC-20 extraction path should:

1. limit itself to the canonical window:
    - `2025-03-16 → 2025-06-17`

2. load:
    - ERC-20 transaction events
    - latest ERC-20 token metadata snapshot

3. normalize:
    - token addresses
    - sender/recipient addresses
    - raw values
    - decimals and display values

4. construct:
    - token-aware address summaries
    - top counterparties
    - top tokens
    - stablecoin and concentration candidates

5. emit:
    - extracted EVM case datasets
    - curated case artifacts
    - explanation-ready ERC-20 enriched summaries

---

## Suggested Repo Outputs

### Raw extracted layer
Examples:
- `data/cases/extracted/<address>.json`

### Curated layer
Examples:
- `data/cases/curated/<case-id>.json`

### Future docs/case studies
Examples:
- future ERC-20 spam / noisy inbound case study
- future stablecoin-heavy settlement-flow case study
- future protocol-mediated ERC-20 routing case study

---

## Relationship to Scoring Engine

The ERC-20 data model is designed to feed the same scoring engine concepts already used for EVM wallet profiling.

That means ERC-20-derived reasons should eventually map into familiar categories like:

- `FRAUD`
- `REPUTATION`
- `LENDING`

And reason codes should remain explainable, for example:

- `repeated_flagged_counterparty_interaction`
- `single_service_concentration`
- `stablecoin_dominant_flow`
- `token_spam_fan_in`
- `single_token_dominance`

This keeps source-specific extraction separate from shared scoring conventions and explanation.

---

## Relationship to Ethereum Calls

ERC-20 transfers alone are useful but incomplete for full EVM behavior modeling.

Ethereum calls are needed later for:
- internal transfer reasoning
- router / swap interpretation
- round-trip and pass-through logic
- contract-mediated flow understanding

### MVP rule

ERC-20 transactions + token metadata are enough for:
- token-aware summaries
- concentration logic
- repeated flagged interaction logic

But they are not enough for:
- full swap path decoding
- robust DeFi semantic reasoning

---

## Out of Scope for the MVP

The following are intentionally deferred:

- full event-indexed EVM warehouse
- complete DeFi decoding
- protocol-specific swap reconstruction
- advanced bridge attribution
- full cross-chain stablecoin flow modeling
- token reputation scoring at production scale

---

## Next Steps

### Immediate
- implement repeated interaction with flagged counterparties
- implement concentration risk to a single service
- add token metadata enrichment into extracted / curated cases
- define stablecoin reference list strategy

### Near-term
- add ERC-20-aware case studies
- add stablecoin-heavy flow observation
- integrate Ethereum calls once downloaded

### Later
- router-aware behavior modeling
- token spam / airdrop clustering
- cross-token corridor reasoning
- bridge-aware and cross-chain token flow logic

---

## Related Documents

- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
- [`README.md`](../README.md)
