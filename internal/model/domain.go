/*
 *
 *  *  Copyright (c) 2026 Piyush Daiya
 *  *  *
 *  *  * Permission is hereby granted, free of charge, to any person obtaining a copy
 *  *  * of this software and associated documentation files (the "Software"), to deal
 *  *  * in the Software without restriction, including without limitation the rights
 *  *  * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  *  * copies of the Software, and to permit persons to whom the Software is
 *  *  * furnished to do so, subject to the following conditions:
 *  *  *
 *  *  * The above copyright notice and this permission notice shall be included in all
 *  *  * copies or substantial portions of the Software.
 *  *  *
 *  *  * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  *  * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  *  * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  *  * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  *  * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  *  * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *  *  * SOFTWARE.
 *
 */

package model

import (
	"math/big"
	"time"
)

// ValidationResult is the standardized output for ANY blockchain
type ValidationResult struct {
	IsValid bool   `json:"is_valid"`
	Network string `json:"network"` // e.g., "EVM", "BITCOIN", "SOLANA"
	Address string `json:"address"`

	// Live Chain Data (if valid & online)
	IsActive bool      `json:"is_active"` // True if Balance > 0 OR Nonce > 0
	Nonce    uint64    `json:"nonce"`     // Transaction count
	Balance  *big.Int  `json:"balance"`   // Wei / Satoshis / Lamports
	LastSeen time.Time `json:"last_seen"` // Placeholder for Indexer data

	ErrorMsg string `json:"error,omitempty"`
}

// Config holds RPC URLs for different chains
type Config struct {
	EvmRPC     string
	SolanaRPC  string
	BitcoinRPC string // Usually an Indexer API (like Blockstream) rather than raw RPC
}
