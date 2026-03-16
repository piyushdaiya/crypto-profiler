# Case Study 2: Direct Sanctioned Wallet

## Summary

This case demonstrates the deterministic high-risk path in Crypto Profiler.

A Bitcoin wallet is identified as a sanctioned address through the watchlist engine, and Crypto Profiler immediately short-circuits to a critical result. Unlike heuristic-only scenarios, this is a direct policy-relevant outcome with maximum risk and mandatory review.

---

## Objective

Validate that Crypto Profiler can:

- detect a direct sanctions match through the watchlist engine
- immediately assign maximum risk
- bypass lower-priority heuristic scoring
- return a clear, explainable, compliance-oriented result

---

## Test Wallet

- **Address:** `bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx`
- **Network:** `BITCOIN`

---

## Wallet Profile Snapshot

- **Valid wallet:** Yes
- **Network:** BITCOIN
- **Active:** Yes
- **Balance:** 0.00000000 BTC
- **Transaction count:** 2
- **First seen:** 2023-04-07T07:28:31Z
- **Last seen:** 2023-07-16T18:19:48Z

---

## Triggered Outcome

Crypto Profiler received a direct sanctions hit from the watchlist engine and intentionally short-circuited to a critical result.

### Expected behavior
- `risk_score = 100`
- `risk_grade = "CRITICAL (Sanctioned)"`
- `review_recommended = true`
- all risk breakdown categories set to `100`
- sanctions-specific reason returned
- heuristic scoring skipped

---

## Final Output

```json
{
  "address": "bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx",
  "network": "BITCOIN",
  "is_valid": true,
  "validation_details": "Active Account (History Found) | Last Active: 2023-07-16",
  "is_active": true,
  "balance": "0.00000000 BTC",
  "tx_count": 2,
  "first_seen": "2023-04-07T07:28:31Z",
  "last_seen": "2023-07-16T18:19:48Z",
  "risk_score": 100,
  "risk_grade": "CRITICAL (Sanctioned)",
  "review_recommended": true,
  "risk_breakdown": {
    "fraud_risk": 100,
    "reputation_risk": 100,
    "lending_risk": 100
  },
  "risk_reasons": [
    {
      "code": "direct_sanctions_match",
      "category": "FRAUD",
      "description": "CRITICAL: OFAC sanctioned address (XBT)",
      "offset": 100,
      "source": "watchlist_engine",
      "related_entity": "OFAC",
      "related_address": "bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx",
      "severity": "CRITICAL",
      "confidence": "HIGH",
      "evidence_count": 1
    }
  ]
}
```

---

## Interpretation

This result represents the **clearly high risk** end of the Crypto Profiler scoring model.

Unlike contextual or heuristic cases, this scenario is deterministic:
- the address is directly identified as sanctioned
- the watchlist engine provides the decisive signal
- Crypto Profiler escalates immediately without depending on additional behavioral evidence

This is the correct behavior for sanctions-first compliance workflows.

---

## Why This Matters

A direct sanctions match is not just another score input. It is a control-critical event that should drive immediate escalation.

This case demonstrates that Crypto Profiler can distinguish between:

- **observed risk**
- **reviewable risk**
- **clearly high risk**

and apply the appropriate response for each.

---

## Product Design Insight

This case validates a core architectural principle:

> Deterministic sanctions outcomes should short-circuit heuristic scoring.

That improves:
- compliance defensibility
- reviewer trust
- explainability
- operational consistency

---

## Classification

- **Case type:** Deterministic sanctions hit
- **Primary category:** Sanctions / watchlist screening
- **Risk outcome:** Clearly high risk
- **Portfolio value:** Demonstrates watchlist-engine integration, explainable short-circuit logic, and compliance-grade decisioning