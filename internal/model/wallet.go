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
	RiskScore         float64              `json:"risk_score"` // Combined Score (0-100)
	RiskGrade         string               `json:"risk_grade"` // EXCELLENT, NEUTRAL, FAILING, etc.
	ReviewRecommended bool                 `json:"review_recommended"`
	RiskBreakdown     RiskCategory         `json:"risk_breakdown"` // Fraud, Reputation, Lending
	RiskReasons       []RiskReason         `json:"risk_reasons"`   // Explainable offsets
	Attribution       *ResolvedAttribution `json:"attribution,omitempty"`
}

type RiskCategory struct {
	Fraud      float64 `json:"fraud_risk"`
	Reputation float64 `json:"reputation_risk"`
	Lending    float64 `json:"lending_risk"`
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
