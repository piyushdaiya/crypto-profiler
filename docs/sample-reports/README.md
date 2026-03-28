# Sample Analyst Reports

These static examples show what `cmd/validator --report` looks like on strong curated cases.

They are useful for:

- portfolio review without running commands
- quick recruiter or hiring-manager skim
- side-by-side comparison with the live demo walkthrough

## Included Samples

### Ethereum

- [`ethereum-tornado-router.txt`](ethereum-tornado-router.txt)
- Demonstrates: trace-aware Ethereum Layer 1 reporting, labeled high-risk infrastructure, and corroborating attribution with explicit actor context in the report surface.
- Demo command:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
```

### Solana

- [`solana-authority-operator.txt`](solana-authority-operator.txt)
- Demonstrates: stablecoin-flow modeling, authority-heavy role interpretation, and repeated counterparty concentration on Solana.
- Demo command:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-solana/solana-stablecoin-authority-operator.json
```

### Bitcoin

- [`bitcoin-operational-hub.txt`](bitcoin-operational-hub.txt)
- Demonstrates: UTXO-flow language, spend-heavy operational behavior, broad but structured Bitcoin counterparty surfaces, and bounded WalletExplorer-style actor grouping in the report.
- Demo command:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json
```

### ERC-20

- [`erc20-uniswap-v2-router.txt`](erc20-uniswap-v2-router.txt)
- Demonstrates: token-surface breadth, trusted protocol context, repeated counterparty activity, and actor-aware contextual findings in ERC-20 Layer 1 reporting.
- Demo command:

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-erc20/erc20-uniswap-v2-router-trusted-token-hub.json
```

## Recommended Review Order

1. Ethereum for the strongest end-to-end story.
2. Solana or Bitcoin to show genuinely chain-specific modeling.
3. ERC-20 to show token-surface reasoning and contextual scoring.
