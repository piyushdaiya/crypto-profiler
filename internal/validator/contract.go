package validator

import (
	"context"
	"time"
)

type WalletProfile struct {
	Address           string     `json:"address"`
	Network           string     `json:"network"`
	IsValid           bool       `json:"is_valid"`
	ValidationDetails string     `json:"validation_details"`
	IsActive          bool       `json:"is_active"`
	Balance           string     `json:"balance"`
	TxCount           int        `json:"tx_count"`
	FirstSeen         *time.Time `json:"first_seen,omitempty"`
	LastSeen          *time.Time `json:"last_seen,omitempty"`

	// --- NEW: Advanced Risk Scoring ---
	RiskScore     float64      `json:"risk_score"`     // Combined Score (0-100)
	RiskGrade     string       `json:"risk_grade"`     // EXCELLENT, NEUTRAL, FAILING, etc.
	RiskBreakdown RiskCategory `json:"risk_breakdown"` // Fraud, Reputation, Lending
	RiskReasons   []RiskReason `json:"risk_reasons"`   // Explainable offsets
}

type RiskCategory struct {
	Fraud      float64 `json:"fraud_risk"`
	Reputation float64 `json:"reputation_risk"`
	Lending    float64 `json:"lending_risk"`
}

type RiskReason struct {
	Category    string  `json:"category"` // "FRAUD", "REPUTATION"
	Description string  `json:"description"`
	Offset      float64 `json:"offset"`    // e.g. +15.5 or -5.0
}

type Transaction struct {
	TimeStamp int64  `json:"timeStamp"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Hash      string `json:"hash"`
}

type ChainStrategy interface {
	Name() string
	IsValidSyntax(address string) bool
	FetchState(ctx context.Context, address string, apiKey string) (*WalletProfile, error)
}