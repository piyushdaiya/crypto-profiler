# Crypto Profiler

A robust, strategy-based Go application that validates crypto wallet addresses across multiple chains (EVM, Solana, Bitcoin). It performs **offline syntax checks** (Regex/Checksum) and **online state verification** (API/RPC) to determine if a wallet is active, its age, and its balance.

## 🏗 Architecture & Design

This project uses the **Strategy Design Pattern** to ensure scalability. Adding a new blockchain is as simple as creating a new file that satisfies the `ChainStrategy` interface; no changes to the core logic are required.

### Key Components

1. **The Interface (`contract.go`)**: Defines the standard behavior every blockchain strategy must implement:

   * `IsValidSyntax(address)`: Does this string *look* like a valid address? (Offline, Fast)
   * `FetchState(address, config)`: Query the blockchain for balance and history. (Online, Slower)
2. **The Strategies (`internal/validator/`)**:

   * **EVM (Ethereum)**: Uses **Etherscan V2 API**. Checks regex, checksum, balance, and transaction history.
   * **Solana**: Uses **CoinStats API**. Syncs wallet history, checks curve validity, balance, and first/last seen dates.
   * **Bitcoin**: Uses **Blockchain.com Explorer API**. Checks Legacy/SegWit/Taproot formats and transaction history.
3. **The Orchestrator (`main.go`)**:

   * Loads configuration (`.env`).
   * Iterates through registered strategies.
   * Detects the correct chain automatically based on syntax.
   * Injects the correct API credentials (API Key or RPC URL).
   * Returns a normalized JSON response.

### Folder Structure

```text
.
├── Dockerfile              # Multi-stage build (Builder -> Alpine Runtime)
├── README.md               # Documentation
├── go.mod / go.sum         # Go dependencies
├── main.go                 # Entry point (Orchestrator)
└── internal
    └── validator
        ├── contract.go     # Interface definition & Data structs
        ├── evm.go          # Ethereum Strategy (Etherscan)
        ├── solana.go       # Solana Strategy (CoinStats)
        ├── bitcoin.go      # Bitcoin Strategy (Blockchain.com)
        └── utils.go        # HTTP/RPC helpers
```


## 🚀 How to Run

### Prerequisites

* **Docker** installed on your machine.
* (Optional) **Go 1.23+** if running locally without Docker.

### 1. Configuration

Create a `.env` file in the root directory with your API keys:

**Code snippet**

```
# Get free keys from Etherscan.io and Coinstats.app
ETHERSCAN_API_KEY=YourEtherscanKeyHere
COINSTATS_API_KEY=YourCoinStatsKeyHere
```

### 2. Build with Docker

Build the lightweight Alpine image:

**Bash**

```
docker build -t crypto-validator .
```

### 3. Usage

Run the container, passing your `.env` file and the wallet address you want to check.

**Check Ethereum:**

**Bash**

```
docker run --env-file .env crypto-validator 0x7dA0aEf1B75035cbf364a690411BCCa7E7859dF8
```

**Check Solana:**

**Bash**

```
docker run --env-file .env crypto-validator LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo 
```

**Check Bitcoin:**

**Bash**

```
docker run --env-file .env crypto-validator bc1q70pfj9ymvrd4rgdkfk0xsch4t7xenna0n0lmdf
```

## 📦 Output Format

The tool returns a standardized JSON object regardless of the chain:

**JSON**

```
{
  "address": "0x7dA...",
  "network": "EVM",
  "is_valid": true,
  "validation_details": "Active | First Seen: 2015-08-07",
  "is_active": true,
  "balance": "450.2312 ETH",
  "tx_count": 1082,
  "first_seen": "2015-08-07T00:00:00Z",
  "last_seen": "2024-05-20T14:30:00Z"
}
```
