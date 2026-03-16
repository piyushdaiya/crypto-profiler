package core

import (
	"math/big"
	"time"
)

// ValidationResult is the standardized output for ANY blockchain
type ValidationResult struct {
	IsValid      bool      `json:"is_valid"`
	Network      string    `json:"network"`      // e.g., "EVM", "BITCOIN", "SOLANA"
	Address      string    `json:"address"`
	
	// Live Chain Data (if valid & online)
	IsActive     bool      `json:"is_active"`    // True if Balance > 0 OR Nonce > 0
	Nonce        uint64    `json:"nonce"`        // Transaction count
	Balance      *big.Int  `json:"balance"`      // Wei / Satoshis / Lamports
	LastSeen     time.Time `json:"last_seen"`    // Placeholder for Indexer data
	
	ErrorMsg     string    `json:"error,omitempty"`
}

// Config holds RPC URLs for different chains
type Config struct {
	EvmRPC     string
	SolanaRPC  string
	BitcoinRPC string // Usually an Indexer API (like Blockstream) rather than raw RPC
}