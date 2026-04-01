Optimism Phase 1 goal

Add Optimism dataset-mode support to Crypto Profiler using:
- 90-day canonical window
- BigQuery as temporary query engine only
- homelab as durable storage
- no persistent mart yet
- no live scoring yet

Repo files to add

scripts/
mine_optimism_candidates.py
extract_optimism_layer2.py
curate_optimism_layer2.py

cmd/validator/
dataset_optimism_layer2_context.go

docs/
OPTIMISM-DATA-MODEL.md   (optional in phase 1, but recommended)

data/
candidates/optimism_addresses.txt
cases/extracted-optimism/
cases/curated-optimism/

Local storage layout on homelab
```
<your-homelab-path>/
optimism/
manifests/
optimism-window.json
optimism-query-log.json
candidates/
optimism_candidates_90d.csv
extracted/
<address>.json
curated/
optimism-bridge-connected-operational-hub.json
optimism-contextual-protocol-router.json
raw-subsets/
<address>_sample_events.parquet
<address>_sample_transfers.parquet
```
Suggested canonical window manifest

{
"chain": "OPTIMISM",
"window_name": "canonical_90d",
"window_start": "YYYY-MM-DDT00:00:00Z",
"window_end": "YYYY-MM-DDT00:00:00Z",
"notes": "Matches the repo-wide 90-day benchmark window used for dataset-mode cases."
}

Phase 1 Optimism case families

1. optimism-bridge-connected-operational-hub
   Purpose:
    - broad bridge/protocol surface
    - repeated interaction
    - operational routing interpretation

2. optimism-contextual-protocol-router
   Purpose:
    - large contextual protocol usage
    - broad surface that should suppress false positives
    - attribution-aware contextual interpretation


Optimism extraction philosophy

Candidate mining:
- high tx count
- broad unique counterparties
- repeated interaction with same contract
- repeated decoded-event hits for bridge/protocol contracts
- stablecoin-heavy transfer activity

Per-address extraction:
- summary
- top counterparties
- contract breakdown
- protocol breakdown
- bridge breakdown
- stablecoin breakdown
- sample transfers
- sample events
- dataset notes

```sql


-- Candidate mining query skeleton (Optimism)
-- Use LOWER() on all addresses because the dataset indexes addresses in lowercase.

DECLARE window_start TIMESTAMP DEFAULT TIMESTAMP("2025-12-30 00:00:00+00");
DECLARE window_end   TIMESTAMP DEFAULT TIMESTAMP("2026-03-30 00:00:00+00");

WITH tx_90d AS (
SELECT
LOWER(from_address) AS from_address,
LOWER(to_address) AS to_address,
block_timestamp,
hash,
receipt_gas_used,
effective_gas_price
FROM `PROJECT.DATASET.transactions`
WHERE block_timestamp >= window_start
AND block_timestamp < window_end
),
addr_flows AS (
SELECT
from_address AS address,
to_address AS counterparty,
"outbound" AS direction,
hash
FROM tx_90d
WHERE from_address IS NOT NULL
AND to_address IS NOT NULL

UNION ALL

SELECT
to_address AS address,
from_address AS counterparty,
"inbound" AS direction,
hash
FROM tx_90d
WHERE from_address IS NOT NULL
AND to_address IS NOT NULL
),
agg AS (
SELECT
address,
COUNT(*) AS interaction_count,
COUNT(DISTINCT counterparty) AS unique_counterparties,
COUNTIF(direction = "inbound") AS inbound_count,
COUNTIF(direction = "outbound") AS outbound_count
FROM addr_flows
WHERE address IS NOT NULL
AND counterparty IS NOT NULL
AND address != counterparty
GROUP BY address
)
SELECT *
FROM agg
WHERE interaction_count >= 200
AND unique_counterparties >= 25
ORDER BY interaction_count DESC
LIMIT 1000;
```

```sql
-- Protocol / bridge / stablecoin enrichment skeleton
-- Build small per-address extracts only for shortlisted candidates.

DECLARE target_address STRING DEFAULT LOWER("0x...");
DECLARE window_start TIMESTAMP DEFAULT TIMESTAMP("2025-12-30 00:00:00+00");
DECLARE window_end   TIMESTAMP DEFAULT TIMESTAMP("2026-03-30 00:00:00+00");

WITH txs AS (
  SELECT
    hash,
    block_timestamp,
    LOWER(from_address) AS from_address,
    LOWER(to_address) AS to_address,
    value
  FROM `PROJECT.DATASET.transactions`
  WHERE block_timestamp >= window_start
    AND block_timestamp < window_end
    AND (
      LOWER(from_address) = target_address OR
      LOWER(to_address) = target_address
    )
),
events AS (
  SELECT
    transaction_hash,
    block_timestamp,
    LOWER(address) AS contract_address,
    event_name
  FROM `PROJECT.DATASET.decoded_events`
  WHERE block_timestamp >= window_start
    AND block_timestamp < window_end
    AND transaction_hash IN (SELECT hash FROM txs)
)
SELECT
  *
FROM events;
```
What mine_optimism_candidates.py should do

Input:
- BigQuery project + dataset/table refs
- canonical window start/end
- max bytes billed
- output path on homelab

Output:
- optimism_candidates_90d.csv

Columns:
- address
- interaction_count
- unique_counterparties
- inbound_count
- outbound_count
- repeated_counterparty_ratio
- top_contract_share
- protocol_event_count
- bridge_event_count
- stablecoin_event_count

Rules:
- never scan outside the 90-day window
- always set maximum_bytes_billed
- always use LOWER()
- write query metadata to manifests/optimism-query-log.json

What extract_optimism_layer2.py should do

Input:
- candidate address list
- canonical window
- max bytes billed
- output directory on homelab

For each selected address:
- query transactions within 90d
- query decoded_events within 90d for tx hashes touching the address
- compute:
    - summary
    - top counterparties
    - contract breakdown
    - protocol breakdown
    - bridge breakdown
    - stablecoin breakdown
    - sample transfers (small capped sample)
    - sample events (small capped sample)

Write:
- extracted/<address>.json

Do not:
- create giant raw mirrors
- keep large temp tables around
  What curate_optimism_layer2.py should do

Input:
- extracted/<address>.json files

Output:
- curated case JSON in data/cases/curated-optimism/

Case selection logic:
1. bridge-connected-operational-hub
    - broad counterparties
    - repeated contract or bridge interaction
    - meaningful bridge breakdown
    - operational-looking outbound or mixed behavior

2. contextual-protocol-router
    - strong protocol concentration
    - broad surface
    - repeated contextual contract usage
    - candidate likely to benefit from attribution-aware suppression

Suggested curated Optimism case shape

{
"chain": "OPTIMISM",
"case_id": "optimism-bridge-connected-operational-hub",
"title": "Optimism Bridge-Connected Operational Hub",
"description": "Address-scoped Optimism Layer 2 case showing repeated bridge/protocol-connected operational routing.",
"risk_posture": "REVIEWABLE_OPERATIONAL_BRIDGE_SURFACE",
"address": "0x...",
"window_start": "YYYY-MM-DDTHH:MM:SSZ",
"window_end": "YYYY-MM-DDTHH:MM:SSZ",
"layer2_summary": {},
"top_counterparties": [],
"sample_transfers": [],
"sample_events": [],
"curation_notes": {
"narrative": "",
"selection_basis": ""
}
}

What dataset_optimism_layer2_context.go should score first

Initial rule families:
- optimism_bridge_connected_operational_hub
- optimism_contextual_protocol_router
- optimism_broad_counterparty_surface
- optimism_repeated_contract_interaction
- optimism_stablecoin_operational_surface
- optimism_mixed_bridge_protocol_surface

Initial scoring behavior:
- fraud offsets for suspicious broad / repeated / bridge-connected operational surfaces
- reputation/contextual offsets for protocol-router style benign infrastructure
- then let existing attribution layer refine the result
- then let actor/exposure and bounded graph summary participate if the extracted data supports it

Cost controls to enforce in code and workflow

1. Stay on free trial and do not activate full paid billing yet.
2. Set a custom daily BigQuery query quota.
3. Set maximum bytes billed on every query job.
4. Filter every query to the canonical 90-day window.
5. Export results immediately to homelab.
6. Delete any temporary BigQuery tables after export.
7. Avoid long-lived GCS storage unless absolutely necessary.

Optimism Phase 1 acceptance criteria

Done means:
- candidate mining script works for Optimism
- address extraction script writes compact JSON to homelab
- curation script creates at least 2 curated Optimism cases
- validator routes those cases in dataset mode
- report output works for curated Optimism cases
- docs updated to mention Optimism dataset-mode support
- no dependency on keeping BigQuery storage alive after export

Suggested local workflow

# 1. Set canonical 90-day window in one manifest
# 2. Mine candidates
python3 scripts/mine_optimism_candidates.py \
--window-start 2025-12-30T00:00:00Z \
--window-end 2026-03-30T00:00:00Z \
--out /homelab/crypto-profiler/optimism/candidates/optimism_candidates_90d.csv

# 3. Extract shortlisted addresses
python3 scripts/extract_optimism_layer2.py \
--candidates /homelab/crypto-profiler/optimism/candidates/optimism_candidates_90d.csv \
--window-start 2025-12-30T00:00:00Z \
--window-end 2026-03-30T00:00:00Z \
--out-dir /homelab/crypto-profiler/optimism/extracted

# 4. Curate high-signal cases
python3 scripts/curate_optimism_layer2.py \
--input-dir /homelab/crypto-profiler/optimism/extracted \
--out-dir /Users/piyushdaiya/Documents/projects/crypto-profiler/data/cases/curated-optimism

# 5. Validate curated report output
go run ./cmd/validator --report --dataset ./data/cases/curated-optimism/optimism-bridge-connected-operational-hub.json
go run ./cmd/validator --report --dataset ./data/cases/curated-optimism/optimism-contextual-protocol-router.json