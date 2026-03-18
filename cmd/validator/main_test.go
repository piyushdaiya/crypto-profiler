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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
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

func TestRun_DatasetModeLoadsCuratedCase(t *testing.T) {
	dir := t.TempDir()

	casePath := dir + "/case.json"
	labelsPath := dir + "/bootstrap_entities.json"

	labelsRaw := `{
	  "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b": {
	    "address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
	    "name": "Tornado Cash Router",
	    "category": "MIXER",
	    "severity": "HIGH",
	    "confidence": "HIGH",
	    "source": "bootstrap_entities",
	    "trusted": false,
	    "notes": "Known mixer-related routing contract"
	  }
	}`

	if err := os.WriteFile(labelsPath, []byte(labelsRaw), 0o644); err != nil {
		t.Fatalf("failed to write bootstrap labels: %v", err)
	}

	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", labelsPath)

	raw := `{
	  "case_id": "tornado-router-high-risk",
	  "title": "High-Risk Mixer Infrastructure",
	  "risk_posture": "REVIEWABLE_HIGH_RISK",
	  "address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
	  "chain": "EVM",
	  "label": "HIGH RISK: Tornado Cash (Router)",
	  "generated_at": "2025-03-17T00:00:00Z",
	  "summary": {
	    "first_seen": "2025-02-27T00:13:23Z",
	    "last_seen": "2025-03-05T23:28:23Z",
	    "inbound_count": 1132,
	    "outbound_count": 0,
	    "unique_counterparties": 129,
	    "native_transfer_count": 1132,
	    "erc20_transfer_count": 0
	  },
	  "top_counterparties": [],
	  "sample_transfers": [
	    {
	      "chain": "EVM",
	      "asset_type": "NATIVE",
	      "tx_hash": "5689f5f1879ff174d06649192375757c813ebe799dad952188e2b451765413a7",
	      "block_id": 21934014,
	      "timestamp": "2025-02-27T00:13:23Z",
	      "from": "0xfe487df046a7e40da04a2b119d23f60987904571",
	      "to": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
	      "amount": "0",
	      "asset_name": "Ethereum",
	      "asset_symbol": "ETH",
	      "label_to": "HIGH RISK: Tornado Cash (Router)"
	    }
	  ],
	  "source_transfer_count": 1132
	}`

	if err := os.WriteFile(casePath, []byte(raw), 0o644); err != nil {
		t.Fatalf("failed to write curated case: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run([]string{"--dataset", casePath}, &out, &errOut, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, errOut.String())
	}

	var got model.WalletProfile
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode stdout json: %v\nstdout=%s", err, out.String())
	}

	if got.Address != "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b" {
		t.Fatalf("unexpected address %q", got.Address)
	}

	if got.Network != "EVM" {
		t.Fatalf("expected EVM network, got %q", got.Network)
	}
}

func setEnvForTest(t *testing.T, key, value string) {
	t.Helper()

	original, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if existed {
			err = os.Setenv(key, original)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("failed to restore env %s: %v", key, err)
		}
	})
}
