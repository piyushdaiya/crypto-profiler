# Demo Shot List

This document captures the exact shot order for the current demo media package and for any future re-recording.

## Recommended Terminal Setup

- use a 16:9 capture area
- use a dark terminal theme
- use a monospace font at a readable size
- prefer a clean prompt with no unrelated shell noise
- run the curated report commands from the repository root

## Shot Order

### 1. Overview Frame

- Duration: 6 seconds
- Asset: `docs/media/screenshots/01-demo-overview.png`
- Purpose: establish the project framing before showing command output

### 2. Ethereum Curated Report

- Duration: 14 seconds
- Asset: `docs/media/screenshots/02-ethereum-curated-report.png`
- Command:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
```

- What to emphasize:
  - strongest end-to-end case
  - trace-aware Ethereum context
  - labeled high-risk infrastructure plus analyst-facing narrative

### 3. Solana Curated Report

- Duration: 14 seconds
- Asset: `docs/media/screenshots/03-solana-curated-report.png`
- Command:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-solana/solana-stablecoin-authority-operator.json
```

- What to emphasize:
  - chain-specific stablecoin-flow modeling
  - authority-role interpretation
  - repeated counterparty concentration

### 4. Bitcoin Curated Report

- Duration: 14 seconds
- Asset: `docs/media/screenshots/04-bitcoin-curated-report.png`
- Command:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json
```

- What to emphasize:
  - UTXO-flow language
  - spend-heavy operational behavior
  - reviewable, non-overclaimed risk framing

### 5. ERC-20 Curated Report

- Duration: 14 seconds
- Asset: `docs/media/screenshots/05-erc20-curated-report.png`
- Command:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-erc20/erc20-uniswap-v2-router-trusted-token-hub.json
```

- What to emphasize:
  - token-surface breadth
  - trusted protocol context
  - contextual scoring rather than blanket high-risk labeling

### 6. Closing Frame

- Duration: 6 seconds
- Purpose: end with the repo takeaway and documentation path

## Total Runtime

- 68 seconds

## Regeneration

To rebuild the packaged screenshots and MP4:

```bash
python3 scripts/generate_demo_media.py
```
