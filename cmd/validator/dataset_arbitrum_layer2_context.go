package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type arbitrumLayer2Summary struct {
	FirstSeen                 string  `json:"first_seen"`
	LastSeen                  string  `json:"last_seen"`
	TxCount                   int     `json:"tx_count"`
	InboundCount              int     `json:"inbound_count"`
	OutboundCount             int     `json:"outbound_count"`
	UniqueCounterparties      int     `json:"unique_counterparties"`
	UniqueToAddresses         int     `json:"unique_to_addresses"`
	UniqueFromAddresses       int     `json:"unique_from_addresses"`
	DominantDirection         string  `json:"dominant_direction"`
	DominantCounterpartyShare float64 `json:"dominant_counterparty_share"`
	DominantContractShare     float64 `json:"dominant_contract_share"`
}

type arbitrumCounterpartySummary struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type arbitrumCuratedLayer2Case struct {
	Chain             string                        `json:"chain"`
	CaseID            string                        `json:"case_id"`
	Title             string                        `json:"title"`
	Description       string                        `json:"description"`
	RiskPosture       string                        `json:"risk_posture"`
	Address           string                        `json:"address"`
	WindowStart       string                        `json:"window_start"`
	WindowEnd         string                        `json:"window_end"`
	Layer2Summary     arbitrumLayer2Summary         `json:"layer2_summary"`
	TopCounterparties []arbitrumCounterpartySummary `json:"top_counterparties"`
	TopToAddresses    []arbitrumCounterpartySummary `json:"top_to_addresses"`
	TopFromAddresses  []arbitrumCounterpartySummary `json:"top_from_addresses"`
	CurationNotes     struct {
		Narrative      string `json:"narrative"`
		SelectionBasis string `json:"selection_basis"`
	} `json:"curation_notes"`
}

func looksLikeArbitrumCase(raw string) bool {
	return strings.Contains(raw, `"chain": "ARBITRUM"`) || strings.Contains(raw, `"chain":"ARBITRUM"`)
}

func buildWalletProfileFromArbitrumCuratedLayer2Case(cc *arbitrumCuratedLayer2Case) *model.WalletProfile {
	if cc == nil {
		return nil
	}

	return &model.WalletProfile{
		Address:           cc.Address,
		Network:           "ARBITRUM",
		IsValid:           true,
		ValidationDetails: fmt.Sprintf("Loaded curated Arbitrum Layer 2 case | Case ID: %s | Risk Posture: %s", cc.CaseID, cc.RiskPosture),
	}
}

func applyArbitrumCuratedLayer2Context(profile *model.WalletProfile, cc *arbitrumCuratedLayer2Case) {
	if profile == nil || cc == nil {
		return
	}

	s := cc.Layer2Summary

	switch cc.CaseID {
	case "arbitrum-repeated-contract-service-like":
		appendArbitrumReason(profile, model.RiskReason{
			Code:          "arbitrum_repeated_contract_service_like",
			Category:      "REPUTATION",
			Description:   fmt.Sprintf("Single-contract dominated Arbitrum service-like behavior observed (dominant contract %.2f%%)", s.DominantContractShare),
			Offset:        9,
			Source:        "dataset_arbitrum_layer2_summary",
			EvidenceCount: s.TxCount,
		})

		if s.UniqueCounterparties <= 1000 {
			appendArbitrumReason(profile, model.RiskReason{
				Code:          "arbitrum_low_counterparty_diversity_context",
				Category:      "REPUTATION",
				Description:   fmt.Sprintf("Relatively low counterparty diversity for very high transaction volume observed (%d counterparties)", s.UniqueCounterparties),
				Offset:        4,
				Source:        "dataset_arbitrum_layer2_summary",
				EvidenceCount: s.UniqueCounterparties,
			})
		}

	case "arbitrum-broad-operational-hub":
		appendArbitrumReason(profile, model.RiskReason{
			Code:          "arbitrum_broad_operational_hub",
			Category:      "FRAUD",
			Description:   fmt.Sprintf("Broad mixed-flow Arbitrum operational surface observed (%d counterparties)", s.UniqueCounterparties),
			Offset:        14,
			Source:        "dataset_arbitrum_layer2_summary",
			EvidenceCount: s.UniqueCounterparties,
		})

		if s.UniqueCounterparties >= 100000 {
			appendArbitrumReason(profile, model.RiskReason{
				Code:          "arbitrum_extremely_broad_counterparty_surface",
				Category:      "FRAUD",
				Description:   fmt.Sprintf("Extremely broad Arbitrum counterparty surface observed (%d counterparties)", s.UniqueCounterparties),
				Offset:        9,
				Source:        "dataset_arbitrum_layer2_summary",
				EvidenceCount: s.UniqueCounterparties,
			})
		}

		if s.OutboundCount > 0 && s.InboundCount > 0 {
			appendArbitrumReason(profile, model.RiskReason{
				Code:          "arbitrum_mixed_flow_operational_pattern",
				Category:      "REPUTATION",
				Description:   fmt.Sprintf("Mixed inbound/outbound operational pattern observed (%d inbound, %d outbound)", s.InboundCount, s.OutboundCount),
				Offset:        3,
				Source:        "dataset_arbitrum_layer2_summary",
				EvidenceCount: s.TxCount,
			})
		}
	}

	profile.RiskScore = arbitrumWeightedRiskScore(profile.RiskBreakdown)
	profile.RiskGrade = arbitrumDetermineGrade(profile.RiskScore)
	profile.ReviewRecommended = arbitrumShouldRecommendReview(profile)
}

func buildReportContextFromArbitrumCase(cc *arbitrumCuratedLayer2Case) *reportContext {
	if cc == nil {
		return nil
	}

	s := cc.Layer2Summary
	topCounterparties := make([]reportCounterparty, 0, len(cc.TopCounterparties))
	for _, cp := range cc.TopCounterparties {
		topCounterparties = append(topCounterparties, reportCounterparty{
			Address: cp.Key,
			Detail:  fmt.Sprintf("%d interactions", cp.Count),
		})
	}

	return &reportContext{
		Mode:        "dataset",
		DatasetType: "Arbitrum Layer 2 dataset",
		CaseID:      cc.CaseID,
		CaseTitle:   cc.Title,
		Narrative:   cc.CurationNotes.Narrative,
		Interpretation: fmt.Sprintf(
			"Arbitrum Layer 2 case showing %s behavior with %d counterparties.",
			blankFallback(s.DominantDirection, "mixed"),
			s.UniqueCounterparties,
		),
		ChainContext: []string{
			fmt.Sprintf("Transactions: %d", s.TxCount),
			fmt.Sprintf("Inbound transfers: %d", s.InboundCount),
			fmt.Sprintf("Outbound transfers: %d", s.OutboundCount),
			fmt.Sprintf("Unique counterparties: %d", s.UniqueCounterparties),
			fmt.Sprintf("Dominant direction: %s", blankFallback(s.DominantDirection, "unknown")),
			fmt.Sprintf("Dominant counterparty share: %.2f%%", s.DominantCounterpartyShare),
			fmt.Sprintf("Dominant contract share: %.2f%%", s.DominantContractShare),
		},
		TopCounterparties: topCounterparties,
	}
}

func appendArbitrumReason(profile *model.WalletProfile, reason model.RiskReason) {
	if profile == nil {
		return
	}

	profile.RiskReasons = append(profile.RiskReasons, reason)

	switch strings.ToUpper(reason.Category) {
	case "FRAUD":
		profile.RiskBreakdown.Fraud += reason.Offset
	case "REPUTATION":
		profile.RiskBreakdown.Reputation += reason.Offset
	case "LENDING":
		profile.RiskBreakdown.Lending += reason.Offset
	}
}

func arbitrumWeightedRiskScore(breakdown model.RiskCategory) float64 {
	score := (breakdown.Fraud * 0.5) + (breakdown.Reputation * 0.3) + (breakdown.Lending * 0.2)
	return arbitrumRound(score, 2)
}

func arbitrumDetermineGrade(score float64) string {
	switch {
	case score < 5:
		return "MINIMAL (Observed)"
	case score < 20:
		return "LOW (Reviewable)"
	case score < 50:
		return "ELEVATED"
	default:
		return "HIGH RISK"
	}
}

func arbitrumShouldRecommendReview(profile *model.WalletProfile) bool {
	if profile == nil {
		return false
	}
	return profile.RiskScore >= 5
}

func arbitrumRound(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}
