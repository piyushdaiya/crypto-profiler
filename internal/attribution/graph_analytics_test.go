package attribution

import (
	"testing"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func TestBuildGraphSummaryFromWave5CInput_ContextualCoverage(t *testing.T) {
	input := Wave5CInput{
		Network: "BITCOIN",
		Counterparties: []InteractionCounterparty{
			{
				Address:       "cp1",
				Interactions:  20,
				InboundCount:  20,
				OutboundCount: 0,
			},
			{
				Address:       "cp2",
				Interactions:  10,
				InboundCount:  5,
				OutboundCount: 5,
			},
		},
	}

	resolve := func(address string) *model.ResolvedAttribution {
		switch address {
		case "cp1", "cp2":
			return &model.ResolvedAttribution{
				Address:    address,
				Label:      "Exchange Cluster",
				Actor:      "Sample Exchange",
				Category:   model.LabelCategory("EXCHANGE"),
				RiskClass:  model.AttributionRiskClass("EXCHANGE"),
				Contextual: true,
				Confidence: 0.70,
			}
		default:
			return nil
		}
	}

	summary := BuildGraphSummaryFromWave5CInput("profile", input, resolve)
	if summary == nil {
		t.Fatal("expected graph summary")
	}
	if summary.UniqueActors != 1 {
		t.Fatalf("expected 1 actor, got %d", summary.UniqueActors)
	}
	if summary.DirectContextualActorCount != 1 {
		t.Fatalf("expected 1 contextual actor, got %d", summary.DirectContextualActorCount)
	}
	if summary.AttributedInteractions == 0 {
		t.Fatal("expected attributed interactions")
	}
}

func TestApplyGraphSummaryContext_BoundedAdjustments(t *testing.T) {
	profile := &model.WalletProfile{
		Address:   "profile",
		Network:   "EVM",
		RiskScore: 10,
		RiskGrade: "LOW (Reviewable)",
		RiskBreakdown: model.RiskCategory{
			Fraud:      10,
			Reputation: 5,
			Lending:    0,
		},
		GraphSummary: &model.GraphSummary{
			TotalInteractions:          20,
			AttributedInteractions:     12,
			AttributedInteractionShare: 0.60,
			UniqueActors:               2,
			TopActorShare:              0.85,
			DirectRiskyActorCount:      1,
			TopActors: []model.GraphActorSummary{
				{
					Actor:            "Sample Mixer",
					Category:         "MIXER",
					RiskClass:        "ILLICIT_SERVICE",
					RiskEscalating:   true,
					InteractionCount: 10,
					UniqueAddresses:  2,
					Share:            0.85,
				},
			},
			Motifs: []model.GraphMotif{
				{
					Code:    "risky_actor_u_turn",
					Summary: "Sampled flows show inbound and outbound interaction with risky actor Sample Mixer (possible U-turn style pattern).",
					Count:   4,
				},
			},
		},
	}

	ApplyGraphSummaryContext(profile)

	if profile.RiskScore <= 10 {
		t.Fatalf("expected risk score increase, got %.2f", profile.RiskScore)
	}
	if len(profile.RiskReasons) == 0 {
		t.Fatal("expected graph-derived risk reasons")
	}
}
func TestApplyGraphSummaryContext_VisibleGraphScoringTriggers(t *testing.T) {
	profile := &model.WalletProfile{
		Address:           "0xprofile",
		Network:           "EVM",
		IsValid:           true,
		RiskScore:         12.0,
		RiskGrade:         "LOW (Reviewable)",
		ReviewRecommended: true,
		RiskBreakdown: model.RiskCategory{
			Fraud:      18.0,
			Reputation: 2.0,
			Lending:    0,
		},
		GraphSummary: &model.GraphSummary{
			TotalInteractions:          30,
			AttributedInteractions:     24,
			AttributedInteractionShare: 0.80,
			UniqueActors:               2,
			TopActorShare:              0.75,
			ConcentrationHHI:           0.61,
			DirectRiskyActorCount:      1,
			DirectContextualActorCount: 1,
			NearRiskyActorCount:        2,
			TopActors: []model.GraphActorSummary{
				{
					Actor:            "Sample Mixer Cluster",
					Category:         "MIXER",
					RiskClass:        "ILLICIT_SERVICE",
					RiskEscalating:   true,
					Confidence:       0.97,
					InteractionCount: 18,
					InboundCount:     9,
					OutboundCount:    9,
					UniqueAddresses:  3,
					Share:            0.75,
				},
				{
					Actor:            "Sample Exchange",
					Category:         "EXCHANGE",
					RiskClass:        "EXCHANGE",
					Contextual:       true,
					Confidence:       0.72,
					InteractionCount: 6,
					InboundCount:     6,
					OutboundCount:    0,
					UniqueAddresses:  2,
					Share:            0.25,
				},
			},
			Motifs: []model.GraphMotif{
				{
					Code:       "risky_actor_u_turn",
					Summary:    "Sampled flows show inbound and outbound interaction with risky actor Sample Mixer Cluster (possible U-turn style pattern).",
					FromActor:  "Sample Mixer Cluster",
					ToActor:    "Sample Mixer Cluster",
					Count:      6,
					Confidence: 0.95,
				},
				{
					Code:         "contextual_to_risky_pass_through",
					Summary:      "Sampled flows suggest contextual-to-risky pass-through from Sample Exchange to Sample Mixer Cluster.",
					FromActor:    "Sample Exchange",
					ToActor:      "Sample Mixer Cluster",
					FromCategory: "EXCHANGE",
					ToCategory:   "MIXER",
					Count:        4,
					Confidence:   0.82,
				},
			},
		},
	}

	beforeScore := profile.RiskScore
	beforeFraud := profile.RiskBreakdown.Fraud

	ApplyGraphSummaryContext(profile)

	if profile.RiskScore <= beforeScore {
		t.Fatalf("expected graph scoring to increase risk score, before=%.2f after=%.2f", beforeScore, profile.RiskScore)
	}
	if profile.RiskBreakdown.Fraud <= beforeFraud {
		t.Fatalf("expected graph scoring to increase fraud breakdown, before=%.2f after=%.2f", beforeFraud, profile.RiskBreakdown.Fraud)
	}
	if !profile.ReviewRecommended {
		t.Fatal("expected review to remain recommended after graph scoring")
	}

	requiredCodes := map[string]bool{
		"graph_risky_actor_concentration":        false,
		"graph_near_risky_actor_exposure":        false,
		"graph_risky_actor_u_turn":               false,
		"graph_contextual_to_risky_pass_through": false,
	}

	for _, reason := range profile.RiskReasons {
		if _, ok := requiredCodes[reason.Code]; ok {
			requiredCodes[reason.Code] = true
		}
	}

	for code, seen := range requiredCodes {
		if !seen {
			t.Fatalf("expected graph-derived reason %q to be present; reasons=%#v", code, profile.RiskReasons)
		}
	}
}
