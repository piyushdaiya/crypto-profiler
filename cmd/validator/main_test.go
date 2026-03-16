package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/address"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type stubStrategy struct {
	name    string
	match   bool
	profile *model.WalletProfile
	err     error
}

func (s *stubStrategy) Name() string {
	return s.name
}

func (s *stubStrategy) IsValidSyntax(address string) bool {
	return s.match && address == "0xtest"
}

func (s *stubStrategy) FetchState(ctx context.Context, address string, apiKey string) (*model.WalletProfile, error) {
	return s.profile, s.err
}

func TestRun_EmitsJSONForMatchingStrategy(t *testing.T) {
	firstSeen := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:           "0xtest",
		Network:           "EVM",
		IsValid:           true,
		ValidationDetails: "stubbed profile",
		IsActive:          true,
		Balance:           "1.0000 ETH",
		TxCount:           3,
		FirstSeen:         &firstSeen,
		LastSeen:          &lastSeen,
		RiskScore:         12.5,
		RiskGrade:         "LOW (Reviewable)",
		ReviewRecommended: true,
		RiskReasons: []model.RiskReason{
			{
				Code:        "stub_reason",
				Category:    "FRAUD",
				Description: "stub reason",
				Offset:      12.5,
			},
		},
	}

	strategies := []address.ChainStrategy{
		&stubStrategy{
			name:    "STUB",
			match:   true,
			profile: profile,
		},
	}

	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run([]string{"0xtest"}, &out, &errOut, strategies)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, errOut.String())
	}

	if !strings.Contains(errOut.String(), "Analyzing 0xtest on STUB") {
		t.Fatalf("expected analysis log on stderr, got %q", errOut.String())
	}

	var got model.WalletProfile
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode stdout json: %v\nstdout=%s", err, out.String())
	}

	if got.Address != "0xtest" {
		t.Fatalf("expected address 0xtest, got %q", got.Address)
	}

	if got.RiskScore != 12.5 {
		t.Fatalf("expected risk score 12.5, got %v", got.RiskScore)
	}

	if got.RiskGrade != "LOW (Reviewable)" {
		t.Fatalf("expected LOW (Reviewable), got %q", got.RiskGrade)
	}

	if !got.ReviewRecommended {
		t.Fatalf("expected review_recommended=true")
	}
}

func TestRun_InvalidAddressFallsBackToUnknown(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run([]string{"not-a-wallet"}, &out, &errOut, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, errOut.String())
	}

	var got model.WalletProfile
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode stdout json: %v\nstdout=%s", err, out.String())
	}

	if got.Network != "UNKNOWN" {
		t.Fatalf("expected UNKNOWN network, got %q", got.Network)
	}

	if got.IsValid {
		t.Fatalf("expected invalid wallet")
	}
}
