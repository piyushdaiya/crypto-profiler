# Crypto Profiler Data Sourcing Policy

## Purpose

This document defines how Crypto Profiler selects, stores, and commits blockchain-related data, addresses, labels, and case artifacts.

The goal is to keep the repository:

- consistent across chains
- explainable
- legally and ethically defensible
- portfolio-safe
- reproducible for demos and tests

This policy applies to:

- Bitcoin
- Ethereum / EVM
- ERC-20 token metadata and transfer artifacts
- Solana
- future chain integrations

---

## Core Principle

Crypto Profiler should only commit blockchain data that is:

- **intentional**
- **documented**
- **relevant to the MVP**
- **traceable to a clear reason for inclusion**

The repository should **not** become a dumping ground for arbitrary public addresses, large raw datasets, or exploratory investigation notes.

---

## Data Classification

All blockchain-related data should be treated as one of the following classes.

### 1. Documented benchmark or case-study data

This includes:

- sanctioned or officially designated addresses used in deterministic test cases
- known trusted protocol addresses used for baseline or contextual examples
- known high-risk infrastructure used in curated case studies
- curated JSON artifacts intentionally created for demos and validator dataset mode

**Repository status:** may be committed

### 2. Exploratory working data

This includes:

- candidate wallet lists being explored
- one-off public addresses under investigation
- temporary address collections used during research
- local extraction targets not yet promoted to a documented case

**Repository status:** local only, do not commit

### 3. Large raw source datasets

This includes:

- raw Blockchair downloads
- BigQuery / Parquet exports
- full ERC-20 token snapshots
- large trace or transaction shards
- external SSD / cloud bucket source data

**Repository status:** external only, do not commit

### 4. Small reference metadata

This includes:

- compact label files
- chain-specific configuration
- small allowlists / denylists
- example candidate templates

**Repository status:** may be committed if intentional and documented

---

## What May Be Committed

The following are acceptable to keep in the repository.

### A. Curated case artifacts
Examples:

- curated JSON files under `data/cases/curated/`
- enriched curated JSON files under `data/cases/curated-enriched/`
- documented benchmark cases used in demos and tests

### B. Test fixtures and benchmark entities
Examples:

- sanctioned example addresses used for deterministic validation
- trusted protocol routers used for baseline comparisons
- known high-risk infrastructure used for repeated analyzer tests

### C. Small example candidate templates
Examples:

- `evm_addresses.example.txt`
- `bitcoin_addresses.example.txt`
- `solana_addresses.example.txt`

These should contain comments or clearly documented sample structure, not arbitrary public addresses.

### D. Small label/config files
Examples:

- bootstrap entity labels
- compact case manifests
- small token reference files that are intentionally part of the product

---

## What Must Not Be Committed

The following should stay out of the repository.

### A. Arbitrary public wallet lists
Do not commit:

- random public addresses
- exploratory candidate lists
- addresses collected without a documented reason
- one-off investigation targets

### B. Large raw blockchain datasets
Do not commit:

- Bitcoin raw transactions / inputs / outputs
- Ethereum raw transaction dumps
- ERC-20 raw transaction archives
- full token metadata snapshots
- BigQuery trace exports
- compressed trace shards or large local extractions

### C. Local-only working files
Do not commit:

- `*.local.txt`
- temporary exports
- SSD paths
- cloud bucket object lists
- scratch investigation outputs

### D. Unverified or weakly sourced labels
Do not commit addresses as labeled cases if they are only:

- social-media rumors
- unsourced forum claims
- general news mentions without explicit address publication
- ad hoc suspicion without provenance

---

## Public Address Inclusion Policy

Public blockchain addresses may be committed **only if all of the following are true**:

1. the address is intentionally included for a product reason
2. the address is documented in a case study, test, benchmark, or curated example
3. the address has a clear label and category
4. the source of that label is recorded or defensible
5. the case maps to behavior that Crypto Profiler can already detect or explain

### Acceptable examples

- sanctioned address benchmark
- known mixer/router benchmark
- trusted protocol router benchmark
- documented fraud or seizure benchmark
- well-explained public-wallet behavior case

### Unacceptable examples

- “this address looked suspicious”
- “this was mentioned in a tweet”
- “this is a random whale wallet”
- “this was in the news but we do not have a clear source trail”

---

## Provenance Requirements

Every committed benchmark or case-study address should have enough context to answer:

- why is this address in the repo?
- what chain is it on?
- what label does it carry?
- what typology does it represent?
- where did the label come from?
- is it a test fixture, curated case, or benchmark example?

This provenance can live in:

- curated case JSON
- case-study markdown
- label files
- manifest files
- test fixture helpers

---

## Recommended Repo Convention

### Candidate lists

Commit only templates:

```text
data/candidates/evm_addresses.example.txt
data/candidates/bitcoin_addresses.example.txt
data/candidates/solana_addresses.example.txt
```

Keep real working files local:

```text
data/candidates/evm_addresses.local.txt
data/candidates/bitcoin_addresses.local.txt
data/candidates/solana_addresses.local.txt
```

### Suggested `.gitignore`

```gitignore
data/candidates/*.local.txt
data/cases/extracted-traces/*.traces.ndjson.gz
data/eth-traces-sample/
```

### Raw data location

Large raw source data should live in:

- external SSD
- cloud bucket
- local non-git directory
- reproducible documented download/export path

---

## Chain-Specific Guidance

## Bitcoin

### Commit
- documented case-study addresses
- labeled benchmark examples
- curated artifacts

### Do not commit
- arbitrary explored addresses
- raw UTXO datasets
- temporary extraction targets

---

## Ethereum / EVM

### Commit
- documented benchmark addresses
- trusted protocol routers used in tests and cases
- high-risk infrastructure used in curated examples
- enriched curated artifacts

### Do not commit
- arbitrary public wallet lists
- temporary candidate files
- raw trace exports
- large extracted trace shards

---

## ERC-20

### Commit
- small intentional token reference/config files
- curated case artifacts that include ERC-20 context

### Do not commit
- full `erc-20_tokens_latest.tsv.gz`
- large raw ERC-20 transfer archives
- bulky token metadata dumps

---

## Solana

### Commit
- documented benchmark/case-study addresses
- curated extracted summaries once intentionally selected
- example candidate templates

### Do not commit
- exploratory public address lists
- raw address-history dumps
- large local transaction hydration outputs unless intentionally curated

---

## Promotion Path: Local to Committed

A local address or dataset should only be promoted into the repository when it has passed this bar:

1. it represents a useful benchmark or case
2. its label/category is documented
3. it supports a current typology or detector
4. it improves tests, demos, or explainability
5. it has been reduced to a curated, portfolio-safe artifact

If those conditions are not met, keep it local.

---

## Documentation Expectations

When a new benchmark address or case is committed, the repo should ideally include at least one of:

- a curated case artifact
- a case-study markdown doc
- a manifest entry
- a test fixture with a clear label
- a short README or architecture note explaining why it exists

---

## Security and Privacy Considerations

Although blockchain addresses are public, the repository should still avoid:

- casual collection of arbitrary real-user wallets
- vague or weakly justified labeling
- unexplained accusations
- unnecessary inclusion of large public-wallet datasets

The standard should be:

- minimal necessary inclusion
- clear purpose
- clear labeling
- clear provenance

---

## Working Rule of Thumb

If the data is:

- **documented, intentional, and part of the product story** → commit it
- **exploratory, temporary, or arbitrary** → keep it local
- **large and raw** → keep it external
- **small and curated** → safe to commit if documented

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/ETHEREUM-DATA-MODEL.md`](ETHEREUM-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
- [`docs/ERC20-DATA-MODEL.md`](ERC20-DATA-MODEL.md)
- [`docs/SOLANA-DATA-MODEL.md`](SOLANA-DATA-MODEL.md)
