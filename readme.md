# Crypto Profiler & Risk Investigator

An advanced, privacy-focused crypto wallet risk scoring engine. It combines **deterministic analysis** (OFAC sanctions, government watchlists) with **heuristic behavioral analysis** (velocity, mixer interaction, age) to generate a multi-dimensional risk score.

## 🏗 Architecture

The system follows a microservices architecture split into two containers:

1. **Watchlist Engine (Server):**

   * Runs 24/7 in the background.
   * Automatically downloads and parses the **OFAC SDN List** (Sanctions).
   * Stores ~500+ sanctioned crypto addresses (BTC, ETH, XMR, etc.) in a local SQLite database.
   * Exposes a high-speed internal HTTP API for checking addresses.
2. **Validator (Client):**

   * CLI tool that accepts a wallet address.
   * Fetches on-chain data (Etherscan, CoinStats).
   * Queries the **Watchlist Engine** to check for federal sanctions.
   * Runs behavioral heuristics (Mixers, Botting, Velocity).
   * Outputs a JSON risk profile.

## 🚀 Quick Start (Docker Compose)

The easiest way to run the full stack is with Docker Compose. This ensures the Engine and Validator are on the same network.

### 1. Prerequisites

* Docker & Docker Compose installed.
* API Keys for **Etherscan** and **CoinStats** (add them to a `.env` file).

### 2. Setup `.env`

Create a file named `.env` in the root directory:

```bash
ETHERSCAN_API_KEY=your_etherscan_key_here
COINSTATS_API_KEY=your_coinstats_key_here

```

### 3. Build & Start the Engine

This starts the Watchlist Engine in the background. It will immediately begin downloading the OFAC list (~100MB).

```bash
docker compose up -d --build

```

*Wait ~30 seconds for the engine to initialize and download the initial database.*

## Real World Test Scenarios

Below are actual results from the investigator running against various networks and risk profiles.

### 1. 🚨 Critical Risk: OFAC Sanctioned Bitcoin Address

This address is on the OFAC SDN blacklist. The engine detects this immediately via the Watchlist Engine.

**Command:**

```bash
docker compose exec validator ./validator bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx

```

**Output:**

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
  "risk_breakdown": {
    "fraud_risk": 100,
    "reputation_risk": 100,
    "lending_risk": 100
  },
  "risk_reasons": [
    {
      "category": "FRAUD",
      "description": "CRITICAL: OFAC Sanctioned Address (XBT)",
      "offset": 100
    },
    {
      "category": "REPUTATION",
      "description": "Government Blacklisted Entity",
      "offset": 100
    },
    {
      "category": "LENDING",
      "description": "Prohibited: Federal Sanctions",
      "offset": 100
    }
  ]
}

```

### 2. ✅ Safe: Inactive Solana Address

A valid but empty wallet with no history.

**Command:**

```bash
docker compose exec validator ./validator 8i8UWJ1wfnU811iRtUgtn7idUpPttDM1ATt1bmHok4sP

```

**Output:**

```json
{
  "address": "8i8UWJ1wfnU811iRtUgtn7idUpPttDM1ATt1bmHok4sP",
  "network": "SOLANA",
  "is_valid": true,
  "validation_details": "Inactive Account (No Tx History)",
  "is_active": false,
  "balance": "0.000000000 SOL",
  "tx_count": 0,
  "risk_score": 0,
  "risk_grade": "EXCELLENT (Safe)",
  "risk_breakdown": {
    "fraud_risk": 0,
    "reputation_risk": 0,
    "lending_risk": 0
  },
  "risk_reasons": null
}

```

### 3. ✅ Excellent: Established EVM Address

An active Ethereum wallet with a long history (>1 Year), earning a reputation bonus.

**Command:**

```bash
docker compose exec validator ./validator 0x7dA0aEf1B75035cbf364a690411BCCa7E7859dF8

```

**Output:**

```json
{
  "address": "0x7dA0aEf1B75035cbf364a690411BCCa7E7859dF8",
  "network": "EVM",
  "is_valid": true,
  "validation_details": "Active | First Seen: 2023-10-27",
  "is_active": true,
  "balance": "0.5156 ETH",
  "tx_count": 10000,
  "first_seen": "2023-10-27T09:44:23Z",
  "last_seen": "2024-08-18T09:23:11Z",
  "risk_score": 0,
  "risk_grade": "EXCELLENT (Safe)",
  "risk_breakdown": {
    "fraud_risk": 0,
    "reputation_risk": 0,
    "lending_risk": 0
  },
  "risk_reasons": [
    {
      "category": "REPUTATION",
      "description": "Established History (>1 Year)",
      "offset": -10
    }
  ]
}

```

## 📋 Usage Examples

Run the validator against any wallet address (ETH, BTC, SOL).

```bash
# Check an Ethereum Address
docker compose exec validator ./validator 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045

# Check a Bitcoin Address
docker compose exec validator ./validator 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa

```

## 🔍 The Investigator Logic

The risk score (0-100) is calculated based on three weighted categories.

### 1. Categories


| Category       | Weight | Description                                                  |
| -------------- | ------ | ------------------------------------------------------------ |
| **FRAUD**      | 50%    | Criminal activity, Mixers (Tornado Cash), Scams, Botting.    |
| **REPUTATION** | 30%    | Account age, Connection to KYC Exchanges (Coinbase/Binance). |
| **LENDING**    | 20%    | Creditworthiness, Balance retention, History length.         |

### 2. Risk Factors & Offsets

The engine uses "Explainable AI" logic. Every score change is logged with a reason.


| Detection Type        | Impact             | Example Reason                                |
| --------------------- | ------------------ | --------------------------------------------- |
| **OFAC Sanction**     | **CRITICAL**       | `CRITICAL: Wallet is on OFAC SDN List (XBT)`  |
| **Mixer Interaction** | +55.0 (Fraud)      | `Direct Interaction with Tornado Cash Router` |
| **High Velocity**     | +25.0 (Fraud)      | `High Velocity Behavior (>20 Tx/Hour)`        |
| **Fresh Wallet**      | +35.0 (Fraud)      | `Freshly Created Wallet (<24h)`               |
| **KYC Exchange**      | -15.0 (Reputation) | `Verified Exchange Link (Likely KYC)`         |
| **Long History**      | -10.0 (Lending)    | `Established History (>1 Year)`               |

### 3. Grading Scale

* **0 - 10:** EXCELLENT (Safe)
* **10 - 35:** LOW (Neutral)
* **35 - 60:** WARNING (Elevated)
* **60 - 100:** FAILING (High Risk)

## 🧪 Testing & Verification

### Verify the Engine is Running

Check the logs to ensure the OFAC database was built successfully.

```bash
docker logs -f crypto-profiler-engine-1

```

*Expected Output:*

> `✅ [SYNC] Done. Scanned 18557 parties. Loaded 543 sanctioned addresses.`

```

```
