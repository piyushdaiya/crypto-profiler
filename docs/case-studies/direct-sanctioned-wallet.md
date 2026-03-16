# Case Study 2: Direct Sanctioned Wallet

## Summary

This case demonstrates the clearest high-risk outcome in Crypto Profiler:

> A direct sanctions match should result in immediate escalation, maximum risk scoring, and mandatory review.

Unlike contextual mixer or heuristic-only cases, this scenario is deterministic. The wallet is identified as a sanctioned entity through the watchlist engine, and Crypto Profiler short-circuits to a critical result.

---

## Objective

Validate that Crypto Profiler can:

- detect a direct sanctions match through the watchlist engine
- immediately assign maximum risk
- bypass lower-priority heuristic scoring
- return a clear, explainable, compliance-oriented result

---

## Expected Behavior

When the wallet is confirmed as sanctioned:

- `risk_score` is set to `100`
- `risk_grade` is set to `CRITICAL (Sanctioned)`
- `review_recommended` is set to `true`
- the profiler returns a sanctions-specific reason
- lower-priority scoring logic is skipped

---

## Why This Matters

A direct sanctions hit is not just another risk signal. It is a policy-critical event.

Crypto Profiler intentionally treats this as a deterministic control outcome rather than a weighted heuristic. This improves:

- compliance defensibility
- reviewer confidence
- explainability
- operational consistency

---

## Expected Output Shape

```json
{
  "address": "SANCTIONED_WALLET_ADDRESS",
  "network": "EVM",
  "is_valid": true,
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
      "description": "CRITICAL: OFAC sanctioned address (ETH)",
      "offset": 100,
      "source": "watchlist_engine",
      "related_entity": "OFAC",
      "related_address": "SANCTIONED_WALLET_ADDRESS",
      "severity": "CRITICAL",
      "confidence": "HIGH",
      "evidence_count": 1
    }
  ]
}
```

---

## Interpretation

This result represents the **clearly high risk** end of the scoring spectrum.

It contrasts with cases where:
- a signal is present but context mitigates the outcome
- multiple weak signals require correlation before escalation
- trusted or exchange interactions remain low risk

This case confirms that Crypto Profiler can distinguish between:

- **observed risk**
- **reviewable risk**
- **clearly high risk**

---

## Product Design Insight

This case validates another core design principle:

> Deterministic sanctions outcomes should short-circuit heuristic scoring.

That keeps the system aligned with real compliance workflows and avoids ambiguity when the required action is already clear.