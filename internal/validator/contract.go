package validator

import (
	"context"
	"time"
)

// WalletProfile is the standardized output for ALL chains
type WalletProfile struct {
	Address           string     `json:"address"`
	Network           string     `json:"network"`
	IsValid           bool       `json:"is_valid"`
	ValidationDetails string     `json:"validation_details"`
	
	IsActive          bool       `json:"is_active"`
	Balance           string     `json:"balance"` // String to handle different precisions (Wei/Satoshis)
	TxCount           int        `json:"tx_count"`

	// Rich Data (Available via Etherscan/Indexers)
	FirstSeen         *time.Time `json:"first_seen,omitempty"`
	LastSeen          *time.Time `json:"last_seen,omitempty"`
}

type ChainStrategy interface {
	Name() string
	IsValidSyntax(address string) bool
	// configParam will accept either an API Key (for Etherscan) or an RPC URL (for Sol/BTC)
	FetchState(ctx context.Context, address string, configParam string) (*WalletProfile, error)
}