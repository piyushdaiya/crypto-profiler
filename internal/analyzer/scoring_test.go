package analyzer

import (
	"testing"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func TestDetermineGrade(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{"minimal lower bound", 0.0, "MINIMAL (Observed)"},
		{"minimal upper bound", 4.99, "MINIMAL (Observed)"},
		{"low lower bound", 5.0, "LOW (Reviewable)"},
		{"low upper bound", 19.99, "LOW (Reviewable)"},
		{"elevated lower bound", 20.0, "ELEVATED"},
		{"elevated upper bound", 49.99, "ELEVATED"},
		{"high risk lower bound", 50.0, "HIGH RISK"},
		{"high risk higher", 100.0, "HIGH RISK"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineGrade(tt.score)
			if got != tt.want {
				t.Fatalf("determineGrade(%v) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

func TestShouldRecommendReview(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		hits  []RuleHit
		want  bool
	}{
		{
			name:  "score threshold triggers review",
			score: 5.0,
			hits:  nil,
			want:  true,
		},
		{
			name:  "direct mixer interaction triggers review even with low score",
			score: 2.5,
			hits: []RuleHit{
				{Code: "direct_mixer_interaction", Category: "FRAUD"},
			},
			want: true,
		},
		{
			name:  "mitigation only does not trigger review",
			score: 0.0,
			hits: []RuleHit{
				{Code: "combo_contextual_mitigation_established_wallet", Category: "FRAUD"},
			},
			want: false,
		},
		{
			name:  "watchlist engine unavailable alone does not trigger review",
			score: 0.0,
			hits: []RuleHit{
				{Code: "watchlist_engine_unavailable", Category: "SYSTEM"},
			},
			want: false,
		},
		{
			name:  "fresh wallet triggers review",
			score: 2.0,
			hits: []RuleHit{
				{Code: "fresh_wallet", Category: "FRAUD"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRecommendReview(tt.score, tt.hits)
			if got != tt.want {
				t.Fatalf("shouldRecommendReview(%v, %+v) = %v, want %v", tt.score, tt.hits, got, tt.want)
			}
		})
	}
}

func TestApplyCombinationRules(t *testing.T) {
	tests := []struct {
		name     string
		input    []RuleHit
		wantCode string
		wantHit  bool
	}{
		{
			name: "mixer plus fresh wallet creates combo hit",
			input: []RuleHit{
				{Code: "direct_mixer_interaction", Category: "FRAUD"},
				{Code: "fresh_wallet", Category: "FRAUD"},
			},
			wantCode: "combo_mixer_plus_fresh_wallet",
			wantHit:  true,
		},
		{
			name: "mixer plus high velocity creates combo hit",
			input: []RuleHit{
				{Code: "direct_mixer_interaction", Category: "FRAUD"},
				{Code: "high_velocity_behavior", Category: "FRAUD"},
			},
			wantCode: "combo_mixer_plus_high_velocity",
			wantHit:  true,
		},
		{
			name: "established wallet with mixer and no extra fraud creates mitigation hit",
			input: []RuleHit{
				{Code: "direct_mixer_interaction", Category: "FRAUD"},
				{Code: "established_history", Category: "REPUTATION"},
			},
			wantCode: "combo_contextual_mitigation_established_wallet",
			wantHit:  true,
		},
		{
			name: "mitigation blocked by fresh wallet",
			input: []RuleHit{
				{Code: "direct_mixer_interaction", Category: "FRAUD"},
				{Code: "established_history", Category: "REPUTATION"},
				{Code: "fresh_wallet", Category: "FRAUD"},
			},
			wantCode: "combo_contextual_mitigation_established_wallet",
			wantHit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyCombinationRules(tt.input)

			found := false
			for _, hit := range got {
				if hit.Code == tt.wantCode {
					found = true
					break
				}
			}

			if found != tt.wantHit {
				t.Fatalf("applyCombinationRules() found=%v for %q, want %v; got=%+v", found, tt.wantCode, tt.wantHit, got)
			}
		})
	}
}

func TestAppendReason(t *testing.T) {
	var reasons []model.RiskReason

	hit := RuleHit{
		Code:           "direct_mixer_interaction",
		Category:       "FRAUD",
		Description:    "Direct interaction with mixer",
		Offset:         20,
		Source:         "bootstrap_entities",
		RelatedEntity:  "Tornado Cash Router",
		RelatedAddress: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
		Severity:       model.LabelSeverityHigh,
		Confidence:     model.LabelConfidenceHigh,
		EvidenceCount:  1,
	}

	appendReason(&reasons, hit)

	if len(reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d", len(reasons))
	}

	if reasons[0].Code != hit.Code {
		t.Fatalf("expected code %q, got %q", hit.Code, reasons[0].Code)
	}
}
