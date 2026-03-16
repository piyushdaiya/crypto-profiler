
# Crypto Profiler

[![Status](https://img.shields.io/badge/status-active%20mvp-blue)](#roadmap)
[![Language](https://img.shields.io/badge/language-Go-00ADD8)](#tech-stack)
[![Docker](https://img.shields.io/badge/runtime-Docker-2496ED)](#quick-start)
[![Focus](https://img.shields.io/badge/focus-AML%20%7C%20Fraud%20%7C%20Crypto%20Risk-red)](#overview)
[![Architecture](https://img.shields.io/badge/architecture-Explainable%20Risk%20Scoring-orange)](#architecture)

**Wallet Risk & Exposure Intelligence for AML, Fraud, Sanctions, and Crypto Surveillance**

Crypto Profiler is a Go-based platform for profiling cryptocurrency wallets using deterministic checks, graph-based exposure analysis, and behavioral heuristics.

It is designed for financial institutions, compliance teams, investigators, RegTech product teams, and solutions architects who need a practical way to perform **Know Your Wallet (KYW)** and crypto-risk analysis with explainable results.

---

## Overview

Blockchain data is transparent, but identifying meaningful risk is not.

A wallet can appear benign at first glance while still being:
- directly linked to a sanctioned or risky entity
- only 1–2 hops away from a mixer, exploiter, or laundering route
- behaving like a pass-through mule or cash-out wallet
- showing patterns consistent with layering, smurfing, dusting, or rapid depletion after dormancy

Crypto Profiler helps transform raw wallet activity into an explainable risk assessment by combining:
- **wallet validation**
- **watchlist and risky-entity screening**
- **direct and indirect exposure analysis**
- **behavioral pattern detection**
- **weighted explainable scoring**
- **investigator-friendly outputs**

---

## Why this project exists

Crypto-risk and financial-crime teams need more than a simple blacklist check.

They need tools that can answer questions such as:
- Is this wallet valid and active?
- Is it linked to known risky entities?
- How close is it to a mixer, scam, exploit, or sanctioned actor?
- Does its behavior resemble money laundering, structuring, or rapid cash-out activity?
- Why did the system assign this risk score?

Crypto Profiler is being built to answer those questions in a portfolio-grade, explainable, and extensible way.

---

## Core capabilities

### 1. Wallet validation
- chain-aware address validation
- checksum verification where applicable
- normalized wallet representation

### 2. Risk screening
- exact-match screening against labeled wallets and risky entities
- support for sanctions/watchlist integration
- internal label support for exchanges, mixers, scams, exploit wallets, and trusted entities

### 3. Exposure analysis
- direct counterparty checks
- 1-hop and 2-hop proximity analysis
- weighted exposure scoring
- transaction graph traversal for fund-flow reasoning

### 4. Behavioral detection
Initial and planned heuristics include:
- peeling-chain style layering
- smurfing / structured transfers
- hop-to-mixer proximity
- dusting and sweep patterns
- high-velocity burst activity
- pass-through / rapid outflow behavior

### 5. Explainable scoring
- weighted score from 0–100
- severity bands: Low / Medium / High
- triggered rules and evidence
- rationale string for analyst review

### 6. Investigator-ready output
- structured JSON reports
- CLI-readable summaries
- case-study friendly sample outputs
- architecture designed for future analyst workflows

---

## Architecture

> **Architecture image placeholder**
>
> Add an architecture diagram here later, for example:
>
> `docs/images/crypto-profiler-architecture.png`

### High-level flow

1. Accept wallet address and chain context
2. Validate and normalize the address
3. Retrieve transaction and label context
4. Build wallet exposure graph
5. Apply deterministic and heuristic rules
6. Compute weighted explainable risk score
7. Generate JSON and analyst-friendly output

### Design principles

- **Go-first implementation**
- **Explainable scoring over black-box decisions**
- **Deterministic-first risk detection**
- **Graph-aware exposure analysis**
- **Modular rule engine**
- **Docker-friendly local execution**
- **Designed to integrate with a future shared watchlist engine**

---

## v0.1 MVP scope

The first public milestone is intentionally focused.

### In scope
- Bitcoin and Ethereum / EVM-first wallet model
- wallet validation and normalization
- exact-match risky wallet/entity checks
- direct and limited hop-based exposure analysis
- a small set of high-value behavioral heuristics
- explainable scoring with reason codes
- JSON and CLI outputs
- reproducible demo data and case-study examples

### Out of scope for v0.1
- full-chain ingestion
- production UI dashboard
- mempool surveillance
- ML-first scoring
- fuzzy entity resolution / name matching
- full cross-chain attribution
- complete market-manipulation surveillance engine
- full case management workflows

---

## Example use cases

Crypto Profiler is being designed for scenarios such as:

- screening a wallet before a transfer or onboarding decision
- triaging a wallet linked to suspicious inbound funds
- tracing whether a wallet is 1–2 hops from a mixer or exploit wallet
- identifying laundering-style behaviors such as peeling chains or rapid cash-out
- generating structured risk evidence for analyst review
- demonstrating wallet intelligence architecture in regulated environments

---

## Sample output

```json
{
  "wallet": "0x1234...abcd",
  "chain": "ethereum",
  "risk_score": 82,
  "severity": "high",
  "risk_categories": [
    "sanctions_risk",
    "money_laundering_risk"
  ],
  "triggered_rules": [
    "direct_risky_counterparty",
    "hop_to_mixer_proximity",
    "high_velocity_burst"
  ],
  "evidence": [
    {
      "type": "counterparty_exposure",
      "detail": "1-hop exposure to known mixer-associated wallet"
    },
    {
      "type": "behavioral_pattern",
      "detail": "47 outgoing transactions observed within 52 minutes after prolonged dormancy"
    }
  ],
  "rationale": "Score 82: direct risky counterparty exposure, 1-hop mixer proximity, and abnormal burst activity detected."
}