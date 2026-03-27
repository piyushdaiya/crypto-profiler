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
	"github.com/piyushdaiya/crypto-profiler/internal/analyzer"
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

func TestRun_DatasetModeLoadsERC20CuratedCase(t *testing.T) {
	dir := t.TempDir()

	casePath := dir + "/erc20-case.json"
	labelsPath := dir + "/bootstrap_entities.json"

	labelsRaw := `{
	  "0xe592427a0aece92de3edee1f18e0157c05861564": {
	    "address": "0xe592427a0aece92de3edee1f18e0157c05861564",
	    "name": "Uniswap V3 Router",
	    "category": "PROTOCOL",
	    "severity": "LOW",
	    "confidence": "HIGH",
	    "source": "bootstrap_entities",
	    "trusted": true,
	    "notes": "Trusted protocol router"
	  }
	}`

	if err := os.WriteFile(labelsPath, []byte(labelsRaw), 0o644); err != nil {
		t.Fatalf("failed to write bootstrap labels: %v", err)
	}

	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", labelsPath)

	raw := `{
	  "case_id": "erc20-uniswap-v3-router-token-hub",
	  "title": "ERC-20 Trusted Protocol Token Hub",
	  "risk_posture": "LOW_REVIEWABLE_PROTOCOL_HUB",
	  "address": "0xe592427a0aece92de3edee1f18e0157c05861564",
	  "chain": "EVM",
	  "label": "PROTOCOL: Uniswap V3 Router",
	  "generated_at": "2025-06-17T00:00:00Z",
	  "source_dataset_type": "erc20_layer1",
	  "source_row_count": 25000,
	  "raw_subset_file": "e592427a0aece92de3edee1f18e0157c05861564.erc20.ndjson.gz",
	  "erc20_summary": {
	    "first_seen": "2025-03-16 00:00:23",
	    "last_seen": "2025-06-17 23:59:47",
	    "inbound_transfer_count": 12000,
	    "outbound_transfer_count": 13000,
	    "inbound_counterparties": 2200,
	    "outbound_counterparties": 1800,
	    "unique_counterparties": 3100,
	    "unique_token_contracts": 85,
	    "repeated_counterparties": 36,
	    "max_counterparty_interactions": 160,
	    "dominant_direction": "outbound",
	    "dominant_token_address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	    "dominant_token_symbol": "USDC",
	    "dominant_token_transfer_share_pct": 31.25
	  },
	  "token_breakdown": [
	    {
	      "token_address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	      "token_name": "USD Coin",
	      "token_symbol": "USDC",
	      "token_decimals": 6,
	      "transfer_count": 7812,
	      "inbound_count": 3900,
	      "outbound_count": 3912,
	      "inbound_value_raw": "1250000000000",
	      "outbound_value_raw": "1255000000000",
	      "unique_counterparties": 1800
	    }
	  ],
	  "top_counterparties": [
	    {
	      "address": "0x1111111111111111111111111111111111111111",
	      "interactions": 160,
	      "inbound_count": 80,
	      "outbound_count": 80,
	      "unique_tokens": 12,
	      "top_tokens": [
	        {
	          "token_address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	          "symbol": "USDC",
	          "count": 90
	        }
	      ]
	    }
	  ],
	  "sample_transfers": [],
	  "curation_notes": {
	    "narrative": "Trusted protocol router with a broad ERC-20 surface."
	  }
	}`

	if err := os.WriteFile(casePath, []byte(raw), 0o644); err != nil {
		t.Fatalf("failed to write curated ERC-20 case: %v", err)
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

	if got.Address != "0xe592427a0aece92de3edee1f18e0157c05861564" {
		t.Fatalf("unexpected address %q", got.Address)
	}

	if got.Network != "EVM" {
		t.Fatalf("expected EVM network, got %q", got.Network)
	}

	if got.RiskScore <= 0 {
		t.Fatalf("expected positive dataset-driven risk score, got %v", got.RiskScore)
	}

	var found bool
	for _, reason := range got.RiskReasons {
		if reason.Code == "erc20_trusted_protocol_token_hub" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected erc20_trusted_protocol_token_hub reason, got %+v", got.RiskReasons)
	}
}

func setEnvForTest(t *testing.T, key, value string) {
	t.Helper()

	original, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}
	analyzer.ResetKnownEntitiesCacheForTesting()

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
		analyzer.ResetKnownEntitiesCacheForTesting()
	})
}
