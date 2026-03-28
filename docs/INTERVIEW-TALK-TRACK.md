# Interview Talk Track

## What Problem This Project Solves

Crypto Profiler is a portfolio-grade crypto risk profiling project for AML, sanctions, fraud, and investigative-style wallet review.

The problem it tackles is not just scoring wallets. It is making wallet-risk reasoning explainable, reproducible, and chain-aware enough to discuss in a real analyst or regtech context.

## 30-Second Version

"I built Crypto Profiler as a multi-chain crypto-risk project that focuses on explainable Layer 1 profiling. Ethereum has the strongest live path, while Solana, Bitcoin, and ERC-20 currently use curated dataset mode so I can demonstrate realistic scoring and analyst-facing reports without pretending every chain is equally mature. The repo is designed to show both engineering depth and portfolio-ready presentation."

## 2-Minute Version

"The main design choice was to avoid forcing every chain through one fake universal model. Ethereum, Solana, Bitcoin, and ERC-20 each expose different useful primitives, so the repo keeps chain-specific Layer 1 adapters while still converging on one shared `WalletProfile` output contract.

That lets the same validator produce JSON for engineering workflows and a concise `--report` view for demos. I also treated curated cases as a first-class product surface rather than test junk, because they make the scoring behavior reproducible for interviews, reviews, and regression tests.

The result is a repo that shows extraction patterns, curated artifacts, dataset-mode validation, explainable scoring, CI/security checks, and a demo-friendly report layer without overclaiming graph analytics that are not built yet."

## Why The Architecture Looks Like This

- Ethereum has the strongest live scoring path today, so it uses the shared analyzer directly.
- Solana, Bitcoin, and ERC-20 currently deliver their strongest value through curated dataset mode.
- The shared output contract keeps the CLI and report layer coherent even when chain-specific scoring differs underneath.
- Trace context is used where it materially improves Ethereum explanation, but the docs do not pretend trace-native live scoring is already complete.

## What Is Implemented Today

- live EVM profiling with shared analyzer scoring
- trace-enriched Ethereum curated cases
- ERC-20 Layer 1 curated scoring in dataset mode
- Solana stablecoin-flow curated scoring in dataset mode
- Bitcoin UTXO-flow curated scoring in dataset mode
- analyst-facing report mode
- tests, CI, and practical security checks

## Tradeoffs I Chose Deliberately

- I prioritized explainability over a larger but weaker feature surface.
- I used curated dataset mode as a real delivery path instead of pretending every chain had production-ready live scoring.
- I kept the repo small and reviewable instead of turning it into a full historical chain warehouse.
- I separated implemented behavior from roadmap behavior so the project stays credible.

## What I Would Build Next

- 1-hop and 2-hop exposure summaries
- pass-through and U-turn behavior
- fresh-wallet plus immediate large-flow reasoning
- richer graph-aware scoring
- stronger live Solana and Bitcoin Layer 1 scoring

## How To Demo It In An Interview

1. Start with the Ethereum curated report to show the strongest end-to-end path.
2. Show one non-Ethereum curated report to demonstrate real multi-chain modeling.
3. Call out that the same CLI supports machine-readable JSON and analyst-facing `--report` output.
4. Use the architecture doc and sample reports to keep the walkthrough concise.

## Good Closing Line

"The project is meant to show that I can build practical risk tooling, keep the implementation honest about maturity, and package technical work in a way that is usable for analysts, reviewers, and engineering teams."
