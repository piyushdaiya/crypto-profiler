# Release Notes

## Public Milestone Release

Crypto Profiler is ready for a polished public milestone release as a portfolio-grade crypto risk profiling project.

This release packages the repository as a coherent end-to-end demo of:

- multi-chain Layer 1 wallet profiling across Ethereum, ERC-20, Solana, and Bitcoin
- explainable scoring with visible reasons, grades, and review recommendations
- attribution-aware contextualization with Tier 1 sources, bounded corroboration, and actor-aware interpretation
- analyst-facing `--report` output for demos, screenshots, and interview walkthroughs
- curated case artifacts, sample reports, screenshots, and a short demo reel

## Highlights

### Multi-chain Layer 1 coverage

- Ethereum: live EVM scoring plus trace-enriched curated cases
- ERC-20: curated dataset-mode token-surface profiling
- Solana: curated dataset-mode stablecoin-flow profiling
- Bitcoin: curated dataset-mode UTXO-flow profiling

### Attribution-aware scoring

- Tier 1 attribution can escalate risky actors or suppress false positives for contextual infrastructure
- secondary sources can corroborate, raise confidence modestly, or surface conflicts without dominating the score
- actor-aware repeated interaction, concentration, sampled exposure, and bounded pass-through or U-turn findings improve analyst explanation where attribution support is strong

### Analyst-facing outputs

- `cmd/validator --report` renders a concise analyst brief on top of JSON output
- checked-in sample reports and demo media make the repo easy to review without local setup

### Validation and security posture

- unit and regression tests cover multi-chain dataset-mode paths and report rendering
- CI runs tests plus practical security tooling
- `govulncheck` and `gosec` are wired into the repository workflow

## Useful Entry Points

- [`README.md`](../README.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
- [`docs/DEMO-WALKTHROUGH.md`](DEMO-WALKTHROUGH.md)
- [`docs/sample-reports/README.md`](sample-reports/README.md)
- [`docs/media/README.md`](media/README.md)

## Current Limitations

- Solana, Bitcoin, and ERC-20 currently deliver their strongest value through curated dataset mode rather than full live scoring
- the attribution layer is bounded and explainable, not a generalized graph-resolution platform
- richer value-aware path scoring and broader live multi-chain behavior remain future work
