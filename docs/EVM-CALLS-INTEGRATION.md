# EVM Calls / Traces Integration

## Purpose

This document explains why Ethereum calls and traces matter in Crypto Profiler, and what the repo actually does with them today.

Current status:

- trace export and address-scoped trace extraction are implemented
- curated EVM cases can be enriched with trace summaries
- validator dataset mode surfaces trace-aware context
- trace-driven scoring logic is not fully implemented yet

---

## Why Transfer Rows Alone Are Not Enough

Plain transaction or transfer rows only show the outer shell of an EVM transaction.

They usually answer:

- who submitted the top-level transaction
- which top-level address received it
- which ERC-20 transfer events were emitted

They do not fully answer:

- which contracts routed value internally
- whether value moved through several internal calls before settling
- whether internal execution failed partway through
- how deep the internal call stack became

For wallet profiling, that internal execution context matters.

---

## What Traces Add

### Internal value movement context

Ethereum traces make it possible to see value-bearing internal calls that never appear as separate top-level transactions.

That helps distinguish:

- direct user-to-user transfers
- contract-mediated settlement
- internal value fan-out
- internal value fan-in

### Router and protocol-mediated flows

A top-level transaction to a router can hide many meaningful internal steps.

Trace data provides:

- internal recipients
- internal senders
- call depth
- broader internal counterparty surface

This is important for protocol-heavy addresses such as routers and contract-controlled services.

### Contract-mediated transfers

Some behavior only makes sense when you can see contract execution:

- a contract receives funds and forwards them internally
- a contract touches many internal counterparties in one transaction
- some internal legs fail while others succeed

Transfer rows alone do not tell that story cleanly.

---

## Why This Matters for Profiling

### Better context for protocol-heavy wallets

Without traces, high-activity protocol addresses can look like they only interact with a small outer shell of counterparties.

With traces, we can see:

- deeper routing behavior
- broader internal surfaces
- internal failure patterns

### Better context for contract-mediated high-risk behavior

For mixer-adjacent or router-like flows, traces help answer:

- was this simply a top-level submission to a contract?
- did value pass through multiple internal recipients?
- does the wallet show deep routed behavior rather than simple peer transfers?

### Better groundwork for future pass-through and U-turn detection

Pass-through and U-turn logic depends on seeing more than outer transfer rows.

Trace summaries preserve useful groundwork for later rules such as:

- inbound value that quickly routes back out internally
- transit-only contract behavior
- repeated deep routing patterns

Crypto Profiler is not fully scoring those behaviors yet, but the trace layer is the reason that future work is realistic.

---

## What The Repository Implements Today

### 1. Trace export

The repo assumes raw Ethereum traces are exported separately, currently via BigQuery into Parquet shards stored outside git.

### 2. Address-scoped trace extraction

`scripts/extract_traces.py`:

- scans Parquet shards
- filters rows where sender or recipient matches tracked addresses
- writes per-address compressed NDJSON trace subsets
- writes per-address summary JSON files

Current summary fields include:

- inbound trace count
- outbound trace count
- self trace count
- failed trace count
- value-bearing trace count
- unique counterparties
- max depth
- top counterparties

### 3. Curated-case enrichment

`cmd/enrichcases` merges extracted trace summaries into curated EVM case files.

That adds:

- `trace_summary`
- `trace_top_counterparties`
- `trace_source_count`
- `trace_raw_file`

### 4. Validator dataset-mode surfacing

When a curated EVM case includes trace data, validator dataset mode adds observational reasons such as:

- `dataset_trace_activity_observed`
- `dataset_trace_failed_calls_observed`
- `dataset_trace_deep_routing_observed`
- `dataset_trace_broad_counterparty_surface`

These reasons currently improve explanation without changing the core EVM score.

---

## Current Practical Benefits

Today, trace integration gives the repo three concrete improvements:

1. better Ethereum case-study realism for protocol-heavy or router-heavy wallets
2. richer validator output for curated EVM cases
3. a clean path toward trace-aware behavioral heuristics later

That is the current implementation story. It is stronger than plain transfer-only cases, but it is not yet full trace-native scoring.

---

## What Is Still Future Work

Trace integration is not yet used for:

- live trace-driven scoring in the shared analyzer
- value-weighted internal concentration scoring
- pass-through detection
- U-turn / round-trip detection
- hop-based exposure using internal call paths
- protocol semantic decoding

Those are the next-stage benefits, not current shipped behavior.

---

## Relationship to Ethereum Layer 1

The current Ethereum Layer 1 story in this repo is:

- shared analyzer scoring uses top-level EVM activity and labels
- curated EVM cases can include trace-aware internal-call context
- trace context improves profiling quality even before it directly changes the score

That is why calls and traces matter beyond plain transfer rows.

---

## Related Documents

- [`docs/ETHEREUM-DATA-MODEL.md`](ETHEREUM-DATA-MODEL.md)
- [`docs/SCORING.md`](SCORING.md)
- [`docs/TYPOLOGIES.md`](TYPOLOGIES.md)
- [`ARCHITECTURE.md`](../ARCHITECTURE.md)
