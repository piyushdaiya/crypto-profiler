# Security Policy

## Scope

This repository contains:

- a wallet profiling CLI
- a watchlist / sanctions engine
- local bootstrap label loading
- external data lookups and sanctions-sync behavior

Security work for this project is being tracked in phases.

## Current Security Baseline

Phase 1 focuses on practical safeguards for:

- malformed and abusive input handling
- safe watchlist client behavior
- safe failure on network / parsing errors
- end-to-end sanctions decisioning
- basic API security and configuration hygiene
- curated dataset loader validation for local JSON artifacts
- local-file safety checks for dataset and trace-summary paths
- reproducible dependency and static-analysis scans via `govulncheck` and `gosec`

This baseline is informed by:

- OWASP Top Ten 2025
- OWASP API Security Top 10 2023
- OWASP ASVS
- OWASP WSTG

## Reporting a Vulnerability

Please do **not** open a public GitHub issue for a suspected security vulnerability.

Instead, report security concerns privately to the repository owner through a private channel.

When reporting, include:

- affected component
- reproduction steps
- impact
- logs or payload examples if relevant
- whether the issue is local-only, Docker-only, or both

## Supported Security Testing Areas

Current security-focused testing covers:

- address syntax validation
- watchlist client timeout / non-200 / malformed JSON handling
- sanctions short-circuit behavior
- contextual risk-scoring safety checks
- CLI handling of invalid wallet input
- dataset-mode routing across Ethereum, Solana, Bitcoin, and ERC-20 curated cases
- curated dataset loader handling for malformed JSON and missing required fields
- trace-summary path traversal regression coverage
- CI and local security scanning through `make govulncheck` and `make gosec`

## Out of Scope for Phase 1

The following are not claimed as complete in Phase 1:

- authentication / authorization hardening
- rate limiting
- production secret management
- external-facing deployment hardening
- full threat modeling
- fuzzing
- secret scanning
- SBOM generation
- formal ASVS control verification

## Security Notes

This project is an evolving portfolio and architecture showcase. Security controls and tests are improving iteratively and should not be interpreted as production certification.
