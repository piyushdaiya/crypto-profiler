# OWASP Phase 1 Test Matrix

## Purpose

This document maps the current Crypto Profiler codebase to a practical Phase 1 OWASP-oriented security baseline.

It is **not** a claim of full compliance.  
It is a record of:

- relevant OWASP risk areas
- current controls and tests
- gaps to address in later phases

## Reference Standards

This Phase 1 baseline is informed by:

- **OWASP Top Ten 2025** for general web application security awareness
- **OWASP API Security Top 10 2023** for watchlist-engine API risks
- **OWASP ASVS** as a security control vocabulary
- **OWASP WSTG** as a testing guide

## Repo Components in Scope

- `cmd/validator`
- `cmd/profiler`
- `internal/address`
- `internal/analyzer`
- `internal/watchlist`
- watchlist-engine `/check` path
- sanctions-sync and bootstrap label loading flows

## Phase 1 Coverage

### 1. Input Validation and Injection Resistance
**Relevant OWASP areas**
- Top 10: Injection / input-handling concerns
- API Security: unsafe input handling
- WSTG: input validation testing

**Current coverage**
- EVM, Bitcoin, and Solana syntax validation tests
- validator fallback for invalid addresses
- watchlist client query parameter escaping
- malformed JSON handling in watchlist client tests

**Status**
- Partially covered

---

### 2. Unsafe Consumption of Upstream Data
**Relevant OWASP areas**
- API Security Top 10 2023: Unsafe Consumption of APIs
- WSTG: configuration and error-handling testing

**Current coverage**
- watchlist client timeout tests
- non-200 response handling
- malformed JSON handling
- sanctions-sync operational verification through Docker logs

**Status**
- Partially covered

---

### 3. Security Misconfiguration
**Relevant OWASP areas**
- Top 10: Security Misconfiguration
- WSTG: configuration/deployment testing

**Current coverage**
- Dockerized local execution
- explicit runtime file paths for bootstrap labels
- minimal health-check validation
- no public claim of production hardening

**Status**
- Limited baseline only

---

### 4. Broken Access Control / Unnecessary Exposure
**Relevant OWASP areas**
- Top 10: Broken Access Control
- API Security: authorization and endpoint exposure

**Current coverage**
- internal-only local Docker workflow
- intended `/check` usage path
- no public auth model yet

**Status**
- Not fully addressed in Phase 1

---

### 5. Error Handling and Information Exposure
**Relevant OWASP areas**
- WSTG: error handling
- Top 10: security misconfiguration / sensitive information exposure patterns

**Current coverage**
- watchlist client returns generic connection errors
- validator falls back safely on invalid inputs
- analyzer handles watchlist-engine unavailability without crashing

**Status**
- Partially covered

---

### 6. Business Logic / Risk Decision Integrity
**Relevant OWASP areas**
- WSTG: business logic testing

**Current coverage**
- direct sanctioned wallet short-circuit tests
- contextual mitigation tests
- fresh wallet + mixer + burst escalation tests
- review recommendation logic tests

**Status**
- Covered for Phase 1 baseline

## Current Test Evidence

### Address validation
- `internal/address/validation_test.go`

### Analyzer / scoring
- `internal/analyzer/scoring_test.go`
- `internal/analyzer/labels_loader_test.go`
- `internal/analyzer/investigate_test.go`

### Watchlist client
- `internal/watchlist/client_test.go`
- `internal/watchlist/security_test.go`

### Validator CLI path
- `cmd/validator/main_test.go`
- `cmd/validator/security_test.go`

## Phase 1 Gaps

The following remain for future phases:

- rate limiting and abuse protection
- authn/authz model for engine endpoints
- structured audit logging
- fuzz testing
- dependency / SBOM scanning
- secret scanning beyond the current Go-focused checks
- threat model document
- ASVS control-by-control checklist

## End of Day 1 Definition

Phase 1 is considered complete for Day 1 when:

- core unit and integration-style tests pass
- govulncheck and gosec pass in CI and locally
- watchlist client input handling is hardened
- sanctioned and contextual risk cases are documented
- the OWASP Phase 1 baseline is documented in-repo
