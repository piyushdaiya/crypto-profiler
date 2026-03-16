package analyzer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func newWatchlistServer(t *testing.T, response watchlistResponse) *httptest.Server {
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

	return httptest.NewServer(handler)
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
