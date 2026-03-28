# Demo Walkthrough

## What This Project Is

Crypto Profiler is a portfolio-grade crypto risk profiling project focused on explainable, multi-chain Layer 1 analysis.

It demonstrates how to:

- combine live EVM scoring with curated dataset-mode analysis
- keep chain-specific Layer 1 logic honest instead of pretending every chain fits one ingestion model
- produce explainable outputs that are readable by engineers, analysts, and hiring managers

## What Problems It Demonstrates

- explainable wallet risk scoring instead of opaque labels
- multi-chain Layer 1 profiling across Ethereum, Solana, Bitcoin, and ERC-20 token flows
- practical dataset-mode benchmarking for demos, case studies, and regression testing
- a path from raw chain data to curated analyst-facing outputs

## Best Demo Path

If you have 3 to 5 minutes:

1. Start with Ethereum to establish the shared report format and the trace-aware story.
2. Show Solana or Bitcoin next to prove the repo uses chain-specific Layer 1 modeling rather than generic EVM language everywhere.
3. End with ERC-20 to show contextual token-surface scoring and the broader multi-chain dataset-mode story.

If you have 60 to 90 seconds:

- run the Ethereum report only
- point out the curated case, trace context, top reasons, and analyst-facing narrative
- mention that the same CLI/report surface also supports Solana, Bitcoin, and ERC-20 curated cases

## Recommended Demo Commands

### 1. Ethereum trace-aware curated case

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json
```

What to notice:

- the report surfaces a high-risk label plus trace-aware context
- the top reasons distinguish direct entity risk from trace observations
- the Layer 1 context shows transfer counts and internal trace breadth together

Talk track:

"This is the strongest end-to-end Ethereum example in the repo. It starts from transfer extraction, adds internal trace context, and ends in an analyst-style report rather than raw JSON."

### 2. Solana stablecoin authority case

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-solana/solana-stablecoin-authority-operator.json
```

What to notice:

- the report is explicitly Solana-specific, not generic EVM language reused everywhere
- it highlights dominant role, mint concentration, and authority-linked flow
- the top counterparties immediately show repeated operational linkage

Talk track:

"Solana is modeled differently on purpose. The current Layer 1 slice is stablecoin-flow based, and the report makes that modeling assumption visible instead of hiding it."

### 3. Bitcoin UTXO-flow operational hub

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json
```

What to notice:

- the report talks about receipts, spends, and counterparties in UTXO terms
- repeated interaction concentration is easy to spot from the top counterparties section
- the score is reviewable rather than overclaimed as automatically malicious

Talk track:

"Bitcoin uses an address-level UTXO-flow lens here. The case is useful because it looks operational and service-like, which is exactly the kind of nuance I wanted the repo to preserve."

### 4. ERC-20 trusted protocol hub

```bash
go run ./cmd/validator --report --dataset ./data/cases/curated-erc20/erc20-uniswap-v2-router-trusted-token-hub.json
```

What to notice:

- the report makes token-surface breadth visible without flattening it into a false high-risk claim
- trusted protocol context and broad token activity both appear in the reasons
- the Layer 1 context shows dominant token concentration, repeated counterparties, and token diversity

Talk track:

"This is a good demo of contextual scoring. A very broad ERC-20 surface can still be interpreted as trusted protocol routing when labels and behavior line up."

## Suggested Demo Flow

1. Start with the Ethereum case to establish the shared output contract and trace-aware story.
2. Show Solana or Bitcoin next to prove the repo is genuinely multi-chain rather than Ethereum-only.
3. End with ERC-20 to show the token-surface model and the move toward richer analyst-facing output.

## What To Capture For Screenshots Or A Short Demo Video

Recommended shots:

1. the top section of the Ethereum report showing score, grade, and top reasons
2. the Solana report area that highlights dominant role and authority-linked counterparties
3. the Bitcoin report area that shows spend-heavy UTXO context
4. the ERC-20 report area that shows token breadth plus trusted protocol context
5. the README demo-entry section or architecture diagram as the opening frame

Good sequence for a short screen recording:

1. open the README and point at the demo entry points
2. run the Ethereum report command
3. run one non-Ethereum report command
4. end on the architecture diagram or sample-report index

The goal is to show that the project is both technically real and easy to explain.

## Sample Outputs

Static sample analyst reports live in:

- [`docs/sample-reports/ethereum-tornado-router.txt`](sample-reports/ethereum-tornado-router.txt)
- [`docs/sample-reports/solana-authority-operator.txt`](sample-reports/solana-authority-operator.txt)
- [`docs/sample-reports/bitcoin-operational-hub.txt`](sample-reports/bitcoin-operational-hub.txt)
- [`docs/sample-reports/erc20-uniswap-v2-router.txt`](sample-reports/erc20-uniswap-v2-router.txt)
- [`docs/sample-reports/README.md`](sample-reports/README.md)

Generated screenshots and the demo reel live in:

- [`docs/media/README.md`](media/README.md)
- [`docs/DEMO-SHOT-LIST.md`](DEMO-SHOT-LIST.md)
