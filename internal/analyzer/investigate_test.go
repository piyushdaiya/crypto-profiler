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

package analyzer

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type watchlistResponse struct {
	Sanctioned bool   `json:"sanctioned"`
	Currency   string `json:"currency"`
	Source     string `json:"source"`
}

type localWatchlistServer struct {
	URL   string
	close func() error
}

func (s *localWatchlistServer) Close() {
	if s != nil && s.close != nil {
		_ = s.close()
	}
}

func writeBootstrapLabelsForTest(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap_entities.json")

	content := `{
	  "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b": {
	    "address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
	    "name": "Tornado Cash Router",
	    "category": "MIXER",
	    "severity": "HIGH",
	    "confidence": "HIGH",
	    "source": "bootstrap_entities",
	    "trusted": false,
	    "notes": "Known mixer-related routing contract"
	  },
	  "0xe592427a0aece92de3edee1f18e0157c05861564": {
	    "address": "0xe592427a0aece92de3edee1f18e0157c05861564",
	    "name": "Uniswap V3 Router",
	    "category": "PROTOCOL",
	    "severity": "LOW",
	    "confidence": "HIGH",
	    "source": "bootstrap_entities",
	    "trusted": true,
	    "notes": "Known trusted DeFi routing contract"
	  },
	  "0x1234567890abcdef1234567890abcdef12345678": {
	    "address": "0x1234567890abcdef1234567890abcdef12345678",
	    "name": "Trusted Protocol",
	    "category": "TRUSTED",
	    "severity": "LOW",
	    "confidence": "HIGH",
	    "source": "bootstrap_entities",
	    "trusted": true,
	    "notes": "Trusted protocol example"
	  }
	}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write bootstrap labels: %v", err)
	}

	return path
}

func newWatchlistServer(t *testing.T, response watchlistResponse) *localWatchlistServer {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/check" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on local test port: %v", err)
	}

	server := &http.Server{
		Handler: handler,
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("watchlist test server stopped unexpectedly: %v", err)
		}
	}()

	ts := &localWatchlistServer{
		URL: "http://" + listener.Addr().String(),
		close: func() error {
			defer listener.Close()
			return server.Close()
		},
	}

	t.Cleanup(ts.Close)

	return ts
}

func TestInvestigate_DirectSanctionedWalletShortCircuits(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	server := newWatchlistServer(t, watchlistResponse{
		Sanctioned: true,
		Currency:   "XBT",
		Source:     "OFAC",
	})
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", writeBootstrapLabelsForTest(t))

	firstSeen := time.Date(2023, 4, 7, 7, 28, 31, 0, time.UTC)
	lastSeen := time.Date(2023, 7, 16, 18, 19, 48, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:   "bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx",
		Network:   "BITCOIN",
		IsValid:   true,
		IsActive:  true,
		TxCount:   2,
		FirstSeen: &firstSeen,
		LastSeen:  &lastSeen,
	}

	Investigate(profile, nil)

	if profile.RiskScore != 100 {
		t.Fatalf("expected risk score 100, got %v", profile.RiskScore)
	}

	if profile.RiskGrade != "CRITICAL (Sanctioned)" {
		t.Fatalf("expected critical sanctions grade, got %q", profile.RiskGrade)
	}

	if !profile.ReviewRecommended {
		t.Fatalf("expected review_recommended=true")
	}

	if len(profile.RiskReasons) != 1 {
		t.Fatalf("expected 1 risk reason, got %d", len(profile.RiskReasons))
	}

	if profile.RiskReasons[0].Code != "direct_sanctions_match" {
		t.Fatalf("expected direct_sanctions_match, got %q", profile.RiskReasons[0].Code)
	}
}

func TestInvestigate_EstablishedWalletWithMixerInteractionGetsMitigation(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	server := newWatchlistServer(t, watchlistResponse{
		Sanctioned: false,
		Currency:   "ETH",
		Source:     "OFAC",
	})
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", writeBootstrapLabelsForTest(t))

	firstSeen := time.Date(2015, 9, 28, 8, 24, 43, 0, time.UTC)
	lastSeen := time.Date(2024, 10, 12, 17, 5, 11, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:   "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		Network:   "EVM",
		IsValid:   true,
		IsActive:  true,
		TxCount:   10000,
		FirstSeen: &firstSeen,
		LastSeen:  &lastSeen,
	}

	txs := []model.Transaction{
		{
			From: "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
			To:   "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
		},
	}

	Investigate(profile, txs)

	if profile.RiskGrade != "MINIMAL (Observed)" {
		t.Fatalf("expected MINIMAL (Observed), got %q", profile.RiskGrade)
	}

	if !profile.ReviewRecommended {
		t.Fatalf("expected review_recommended=true for mixer interaction")
	}

	var foundMixer bool
	var foundMitigation bool

	for _, reason := range profile.RiskReasons {
		if reason.Code == "direct_mixer_interaction" {
			foundMixer = true
		}
		if reason.Code == "combo_contextual_mitigation_established_wallet" {
			foundMitigation = true
		}
	}

	if !foundMixer {
		t.Fatalf("expected direct_mixer_interaction reason")
	}

	if !foundMitigation {
		t.Fatalf("expected contextual mitigation reason")
	}
}

func TestInvestigate_FreshWalletMixerAndBurstEscalates(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	server := newWatchlistServer(t, watchlistResponse{
		Sanctioned: false,
		Currency:   "ETH",
		Source:     "OFAC",
	})
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", writeBootstrapLabelsForTest(t))

	firstSeen := time.Now().Add(-1 * time.Hour)
	lastSeen := time.Now()

	profile := &model.WalletProfile{
		Address:   "0xfeed000000000000000000000000000000000001",
		Network:   "EVM",
		IsValid:   true,
		IsActive:  true,
		TxCount:   100,
		FirstSeen: &firstSeen,
		LastSeen:  &lastSeen,
	}

	txs := []model.Transaction{
		{
			From: "0xfeed000000000000000000000000000000000001",
			To:   "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
		},
	}

	Investigate(profile, txs)

	if profile.RiskGrade != "HIGH RISK" && profile.RiskGrade != "ELEVATED" {
		t.Fatalf("expected HIGH RISK or ELEVATED, got %q", profile.RiskGrade)
	}

	if !profile.ReviewRecommended {
		t.Fatalf("expected review_recommended=true")
	}

	var foundFresh bool
	var foundVelocity bool
	var foundComboFresh bool
	var foundComboVelocity bool

	for _, reason := range profile.RiskReasons {
		switch reason.Code {
		case "fresh_wallet":
			foundFresh = true
		case "high_velocity_behavior":
			foundVelocity = true
		case "combo_mixer_plus_fresh_wallet":
			foundComboFresh = true
		case "combo_mixer_plus_high_velocity":
			foundComboVelocity = true
		}
	}

	if !foundFresh {
		t.Fatalf("expected fresh_wallet reason")
	}
	if !foundVelocity {
		t.Fatalf("expected high_velocity_behavior reason")
	}
	if !foundComboFresh {
		t.Fatalf("expected combo_mixer_plus_fresh_wallet reason")
	}
	if !foundComboVelocity {
		t.Fatalf("expected combo_mixer_plus_high_velocity reason")
	}
}

func TestInvestigate_PublicWalletNoisyInboundObserved(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	server := newWatchlistServer(t, watchlistResponse{
		Sanctioned: false,
		Currency:   "ETH",
		Source:     "OFAC",
	})
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", writeBootstrapLabelsForTest(t))

	firstSeen := time.Date(2015, 9, 28, 8, 24, 43, 0, time.UTC)
	lastSeen := time.Date(2025, 3, 5, 23, 53, 35, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:   "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		Network:   "EVM",
		IsValid:   true,
		IsActive:  true,
		TxCount:   1795,
		FirstSeen: &firstSeen,
		LastSeen:  &lastSeen,
	}

	txs := []model.Transaction{
		{From: "0x1000000000000000000000000000000000000001", To: profile.Address, Value: "0", Hash: "tx01"},
		{From: "0x1000000000000000000000000000000000000002", To: profile.Address, Value: "0", Hash: "tx02"},
		{From: "0x1000000000000000000000000000000000000003", To: profile.Address, Value: "0", Hash: "tx03"},
		{From: "0x1000000000000000000000000000000000000004", To: profile.Address, Value: "0", Hash: "tx04"},
		{From: "0x1000000000000000000000000000000000000005", To: profile.Address, Value: "0", Hash: "tx05"},
		{From: "0x1000000000000000000000000000000000000006", To: profile.Address, Value: "0", Hash: "tx06"},
		{From: "0x1000000000000000000000000000000000000007", To: profile.Address, Value: "0", Hash: "tx07"},
		{From: "0x1000000000000000000000000000000000000008", To: profile.Address, Value: "0", Hash: "tx08"},
		{From: "0x1000000000000000000000000000000000000009", To: profile.Address, Value: "0", Hash: "tx09"},
		{From: "0x1000000000000000000000000000000000000010", To: profile.Address, Value: "0", Hash: "tx10"},
		{From: "0x1000000000000000000000000000000000000011", To: profile.Address, Value: "0", Hash: "tx11"},
		{From: "0x1000000000000000000000000000000000000012", To: profile.Address, Value: "0", Hash: "tx12"},
		{From: "0x1000000000000000000000000000000000000013", To: profile.Address, Value: "0", Hash: "tx13"},
		{From: "0x1000000000000000000000000000000000000014", To: profile.Address, Value: "0", Hash: "tx14"},
		{From: "0x1000000000000000000000000000000000000015", To: profile.Address, Value: "0", Hash: "tx15"},
		{From: "0x1000000000000000000000000000000000000016", To: profile.Address, Value: "0", Hash: "tx16"},
		{From: "0x1000000000000000000000000000000000000017", To: profile.Address, Value: "0", Hash: "tx17"},
		{From: "0x1000000000000000000000000000000000000018", To: profile.Address, Value: "0", Hash: "tx18"},
		{From: "0x1000000000000000000000000000000000000019", To: profile.Address, Value: "0", Hash: "tx19"},
		{From: "0x1000000000000000000000000000000000000020", To: profile.Address, Value: "0", Hash: "tx20"},
		{From: "0x1000000000000000000000000000000000000021", To: profile.Address, Value: "10000000000000", Hash: "tx21"},
		{From: "0x1000000000000000000000000000000000000022", To: profile.Address, Value: "0", Hash: "tx22"},
		{From: "0x1000000000000000000000000000000000000023", To: profile.Address, Value: "0", Hash: "tx23"},
		{From: "0x1000000000000000000000000000000000000024", To: profile.Address, Value: "0", Hash: "tx24"},
		{From: "0x1000000000000000000000000000000000000025", To: profile.Address, Value: "0", Hash: "tx25"},
		{From: "0x1000000000000000000000000000000000000026", To: profile.Address, Value: "0", Hash: "tx26"},
		{From: "0x1000000000000000000000000000000000000027", To: profile.Address, Value: "0", Hash: "tx27"},
		{From: "0x1000000000000000000000000000000000000028", To: profile.Address, Value: "0", Hash: "tx28"},
		{From: "0x1000000000000000000000000000000000000029", To: profile.Address, Value: "0", Hash: "tx29"},
		{From: "0x1000000000000000000000000000000000000030", To: profile.Address, Value: "0", Hash: "tx30"},
	}

	Investigate(profile, txs)

	if profile.RiskGrade != "MINIMAL (Observed)" {
		t.Fatalf("expected MINIMAL (Observed), got %q", profile.RiskGrade)
	}

	if profile.ReviewRecommended {
		t.Fatalf("expected review_recommended=false for noisy inbound observation")
	}

	var foundNoisyInbound bool
	var foundFanIn bool
	var foundZeroValue bool

	for _, reason := range profile.RiskReasons {
		switch reason.Code {
		case "noisy_inbound_activity":
			foundNoisyInbound = true
		case "high_counterparty_fan_in":
			foundFanIn = true
		case "zero_value_inbound_pattern":
			foundZeroValue = true
		}
	}

	if !foundNoisyInbound {
		t.Fatalf("expected noisy_inbound_activity reason")
	}
	if !foundFanIn {
		t.Fatalf("expected high_counterparty_fan_in reason")
	}
	if !foundZeroValue {
		t.Fatalf("expected zero_value_inbound_pattern reason")
	}
}
func TestInvestigate_RepeatedFlaggedCounterpartyInteractionEscalates(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	server := newWatchlistServer(t, watchlistResponse{
		Sanctioned: false,
		Currency:   "ETH",
		Source:     "OFAC",
	})
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", writeBootstrapLabelsForTest(t))

	firstSeen := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2025, 3, 5, 23, 53, 35, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:   "0x1111111111111111111111111111111111111111",
		Network:   "EVM",
		IsValid:   true,
		IsActive:  true,
		TxCount:   12,
		FirstSeen: &firstSeen,
		LastSeen:  &lastSeen,
	}

	txs := []model.Transaction{
		{From: profile.Address, To: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", Value: "100", Hash: "tx01"},
		{From: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", To: profile.Address, Value: "50", Hash: "tx02"},
		{From: profile.Address, To: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", Value: "75", Hash: "tx03"},
		{From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", To: profile.Address, Value: "10", Hash: "tx04"},
	}

	Investigate(profile, txs)

	var repeated bool
	for _, reason := range profile.RiskReasons {
		if reason.Code == "repeated_flagged_counterparty_interaction" {
			repeated = true
			if reason.EvidenceCount != 3 {
				t.Fatalf("expected evidence_count=3, got %d", reason.EvidenceCount)
			}
		}
	}

	if !repeated {
		t.Fatalf("expected repeated_flagged_counterparty_interaction reason")
	}

	if profile.RiskScore <= 0 {
		t.Fatalf("expected risk score > 0, got %v", profile.RiskScore)
	}
}
func TestInvestigate_HighRiskServiceConcentrationEscalates(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	server := newWatchlistServer(t, watchlistResponse{
		Sanctioned: false,
		Currency:   "ETH",
		Source:     "OFAC",
	})
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", writeBootstrapLabelsForTest(t))

	firstSeen := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2025, 3, 5, 23, 53, 35, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:   "0x3333333333333333333333333333333333333333",
		Network:   "EVM",
		IsValid:   true,
		IsActive:  true,
		TxCount:   8,
		FirstSeen: &firstSeen,
		LastSeen:  &lastSeen,
	}

	txs := []model.Transaction{
		{From: profile.Address, To: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", Hash: "tx01"},
		{From: profile.Address, To: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", Hash: "tx02"},
		{From: profile.Address, To: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", Hash: "tx03"},
		{From: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", To: profile.Address, Hash: "tx04"},
		{From: profile.Address, To: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", Hash: "tx05"},
		{From: profile.Address, To: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Hash: "tx06"},
		{From: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", To: profile.Address, Hash: "tx07"},
		{From: profile.Address, To: "0xcccccccccccccccccccccccccccccccccccccccc", Hash: "tx08"},
	}

	Investigate(profile, txs)

	var found bool
	for _, reason := range profile.RiskReasons {
		if reason.Code == "high_risk_service_concentration" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected high_risk_service_concentration reason")
	}

	if !profile.ReviewRecommended {
		t.Fatalf("expected review recommended")
	}
}

func TestInvestigate_TrustedServiceConcentrationIsContextual(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	server := newWatchlistServer(t, watchlistResponse{
		Sanctioned: false,
		Currency:   "ETH",
		Source:     "OFAC",
	})
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", writeBootstrapLabelsForTest(t))

	firstSeen := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2025, 3, 5, 23, 53, 35, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:   "0x4444444444444444444444444444444444444444",
		Network:   "EVM",
		IsValid:   true,
		IsActive:  true,
		TxCount:   8,
		FirstSeen: &firstSeen,
		LastSeen:  &lastSeen,
	}

	txs := []model.Transaction{
		{From: profile.Address, To: "0xe592427a0aece92de3edee1f18e0157c05861564", Hash: "tx01"},
		{From: profile.Address, To: "0xe592427a0aece92de3edee1f18e0157c05861564", Hash: "tx02"},
		{From: "0xe592427a0aece92de3edee1f18e0157c05861564", To: profile.Address, Hash: "tx03"},
		{From: profile.Address, To: "0xe592427a0aece92de3edee1f18e0157c05861564", Hash: "tx04"},
		{From: profile.Address, To: "0xe592427a0aece92de3edee1f18e0157c05861564", Hash: "tx05"},
		{From: profile.Address, To: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Hash: "tx06"},
		{From: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", To: profile.Address, Hash: "tx07"},
		{From: profile.Address, To: "0xcccccccccccccccccccccccccccccccccccccccc", Hash: "tx08"},
	}

	Investigate(profile, txs)

	var found bool
	for _, reason := range profile.RiskReasons {
		if reason.Code == "single_service_concentration" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected single_service_concentration reason")
	}

	if profile.ReviewRecommended {
		t.Fatalf("did not expect review recommended")
	}
}
