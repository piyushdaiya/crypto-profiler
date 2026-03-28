# Crypto Profiler Data Sourcing Policy

## Purpose

This document defines what data should be committed, what should stay local, and how the current repo organizes candidate lists, extracted fixtures, and curated cases.

The policy is meant to keep the repository:

- reproducible
- portfolio-safe
- internally consistent
- small enough to work with comfortably

---

## Core Rule

Commit data when it is:

- intentional
- documented
- already part of the product story
- small enough to serve as a fixture, curated case, or reference artifact

Keep data out of git when it is:

- exploratory
- large and raw
- local-only working material
- not yet tied to a documented case or benchmark

---

## Current Repo Convention

### Candidate files

Committed templates:

- `data/candidates/*.example.txt`

Local working lists:

- `data/candidates/*.local.txt`

### Curated artifacts

Committed curated artifacts currently live under:

- `data/cases/curated/`
- `data/cases/curated-enriched/`
- `data/cases/curated-erc20/`
- `data/cases/curated-solana/`
- `data/cases/curated-bitcoin/`

### Extracted reference fixtures

The repo also currently includes small extracted reference fixtures under:

- `data/cases/extracted/`
- `data/cases/extracted-traces/`
- `data/cases/extracted-solana-stablecoin/`
- `data/cases/extracted-bitcoin/`

These should be treated as intentional reference artifacts, not as permission to commit every working extraction output.

---

## What May Be Committed

### Curated benchmark cases

Examples:

- curated Ethereum, ERC-20, Solana, and Bitcoin case JSON
- trace-enriched curated EVM cases
- benchmark cases used in validator dataset mode

### Small reference fixtures

Examples:

- compact extracted summary JSON files
- small compressed extracted samples used as reference fixtures
- small Tier 1 attribution fixture files
- label/config files

### Documented benchmark addresses

Examples:

- sanctioned deterministic test addresses
- trusted router benchmarks
- high-risk infrastructure used in curated case studies
- mining-pool or trusted-service benchmark addresses used in attribution tests and demos

### Tier 1 attribution fixtures

The repo may also commit small intentional Tier 1 attribution fixtures such as:

- `data/labels/tier1_graphsense_entities.json`
- `data/labels/tier1_bitcoin_mining_pools.json`
- `data/labels/bootstrap_entities.json`

These are committed because they are:

- small
- testable
- part of the implemented scoring story
- useful for deterministic demos and regression coverage

### Small example candidate templates

Examples:

- `data/candidates/evm_addresses.example.txt`
- `data/candidates/erc20_addresses.example.txt`
- `data/candidates/bitcoin_addresses.example.txt`
- `data/candidates/solana_addresses.example.txt`

---

## What Must Stay Local

### Working candidate lists

Do not commit:

- `data/candidates/*.local.txt`
- mined address lists that have not been curated
- exploratory wallet collections

### Large raw source data

Do not commit:

- raw Blockchair dumps
- large BigQuery / Parquet exports
- full trace shards
- large ERC-20 transfer archives
- large Solana history or transaction hydration outputs

### Temporary extraction outputs

Do not commit:

- scratch runs
- one-off debug exports
- oversized address-scoped raw subsets that are not intentional fixtures

---

## Chain-Specific Notes

### Ethereum / EVM

Safe to commit when intentional:

- curated EVM case artifacts
- trace-enriched curated case artifacts
- compact trace summary fixtures
- trusted and high-risk benchmark addresses used in tests and curated cases
- small Tier 1 structured attribution fixtures used for scoring and report demos

Keep local or external:

- raw trace exports
- large working trace subsets
- arbitrary public wallet lists

### Solana

Safe to commit when intentional:

- curated stablecoin-flow cases
- compact extracted stablecoin summary fixtures
- example candidate templates

Keep local or external:

- raw stablecoin-flow Parquet dumps
- exploratory public address lists
- general transaction hydration experiments that are not curated

### Bitcoin

Safe to commit when intentional:

- curated UTXO-flow cases
- compact extracted Bitcoin summary fixtures
- example candidate templates
- small mining-pool attribution fixtures

Keep local or external:

- raw Blockchair inputs/outputs
- temporary UTXO extraction runs
- arbitrary explored address lists

### ERC-20

Current status:

- address-scoped ERC-20 Layer 1 extraction is implemented
- ERC-20 curated Layer 1 support is implemented in dataset mode

Safe to commit when intentional:

- small token reference/config files
- curated ERC-20 case artifacts under `data/cases/curated-erc20/`
- committed example candidate templates such as `data/candidates/erc20_addresses.example.txt`
- small Tier 1 label fixtures that support trusted protocol or exchange-style contextualization

Keep local or external:

- large raw ERC-20 transfer archives
- full token metadata dumps
- working extraction outputs under `data/cases/extracted-erc20/`

---

## Promotion Rule

A local address, extracted artifact, or candidate list should only be promoted into git when all of the following are true:

1. it supports a current benchmark, demo, or documented case
2. it maps to behavior the repo can already explain
3. it is small enough to serve as a reference artifact
4. it is documented in a case, test, manifest, or doc

If not, keep it local.

---

## Documentation Expectations

When committing a new benchmark artifact, update the docs that explain it:

- `README.md`
- `ARCHITECTURE.md`
- `docs/TYPOLOGIES.md`
- `docs/SCORING.md`
- `docs/LABEL-SOURCE-HIERARCHY.md`
- the relevant chain-specific data model doc

This keeps the repo story aligned with the actual committed data.

---

## Related Documents

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/LABEL-SOURCE-HIERARCHY.md`](LABEL-SOURCE-HIERARCHY.md)
- [`docs/ETHEREUM-DATA-MODEL.md`](ETHEREUM-DATA-MODEL.md)
- [`docs/ERC20-DATA-MODEL.md`](ERC20-DATA-MODEL.md)
- [`docs/SOLANA-DATA-MODEL.md`](SOLANA-DATA-MODEL.md)
- [`docs/BITCOIN-DATA-MODEL.md`](BITCOIN-DATA-MODEL.md)
