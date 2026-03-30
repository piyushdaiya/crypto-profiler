package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type optimismLayer2Summary struct {
	FirstSeen                     string  `json:"first_seen"`
	LastSeen                      string  `json:"last_seen"`
	TxCount                       int     `json:"tx_count"`
	InboundCount                  int     `json:"inbound_count"`
	OutboundCount                 int     `json:"outbound_count"`
	UniqueCounterparties          int     `json:"unique_counterparties"`
	UniqueToAddresses             int     `json:"unique_to_addresses"`
	UniqueFromAddresses           int     `json:"unique_from_addresses"`
	UniqueFunctionSelectors       int     `json:"unique_function_selectors"`
	DominantDirection             string  `json:"dominant_direction"`
	DominantCounterpartyShare     float64 `json:"dominant_counterparty_share"`
	DominantContractShare         float64 `json:"dominant_contract_share"`
	DominantFunctionSelectorShare float64 `json:"dominant_function_selector_share"`
}

type optimismCounterpartySummary struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type optimismCuratedLayer2Case struct {
	Chain                string                        `json:"chain"`
	CaseID               string                        `json:"case_id"`
	Title                string                        `json:"title"`
	Description          string                        `json:"description"`
	RiskPosture          string                        `json:"risk_posture"`
	Address              string                        `json:"address"`
	WindowStart          string                        `json:"window_start"`
	WindowEnd            string                        `json:"window_end"`
	Layer2Summary        optimismLayer2Summary         `json:"layer2_summary"`
	TopCounterparties    []optimismCounterpartySummary `json:"top_counterparties"`
	TopToAddresses       []optimismCounterpartySummary `json:"top_to_addresses"`
	TopFromAddresses     []optimismCounterpartySummary `json:"top_from_addresses"`
	TopFunctionSelectors []optimismCounterpartySummary `json:"top_function_selectors"`
	CurationNotes        struct {
		Narrative      string `json:"narrative"`
		SelectionBasis string `json:"selection_basis"`
	} `json:"curation_notes"`
}

func looksLikeOptimismCase(raw string) bool {
	return strings.Contains(raw, `"chain": "OPTIMISM"`) || strings.Contains(raw, `"chain":"OPTIMISM"`)
}

func buildWalletProfileFromOptimismCuratedLayer2Case(cc *optimismCuratedLayer2Case) *model.WalletProfile {
	if cc == nil {
		return nil
	}

	return &model.WalletProfile{
		Address:           cc.Address,
		Network:           "OPTIMISM",
		IsValid:           true,
		ValidationDetails: fmt.Sprintf("Loaded curated Optimism Layer 2 case | Case ID: %s | Risk Posture: %s", cc.CaseID, cc.RiskPosture),
	}
}

func applyOptimismCuratedLayer2Context(profile *model.WalletProfile, cc *optimismCuratedLayer2Case) {
	if profile == nil || cc == nil {
		return
	}

	s := cc.Layer2Summary

	switch cc.CaseID {
	case "optimism-repeated-contract-router-like":
		appendOptimismReason(profile, model.RiskReason{
			Code:          "optimism_repeated_contract_router_like",
			Category:      "REPUTATION",
			Description:   fmt.Sprintf("Single-contract and single-selector dominated Optimism behavior observed (dominant contract %.2f%%, dominant selector %.2f%%)", s.DominantContractShare, s.DominantFunctionSelectorShare),
			Offset:        10,
			Source:        "dataset_optimism_layer2_summary",
			EvidenceCount: s.TxCount,
		})

		if s.UniqueCounterparties <= 500 {
			appendOptimismReason(profile, model.RiskReason{
				Code:          "optimism_low_counterparty_diversity_context",
				Category:      "REPUTATION",
				Description:   fmt.Sprintf("Relatively low counterparty diversity for very high transaction volume observed (%d counterparties)", s.UniqueCounterparties),
				Offset:        4,
				Source:        "dataset_optimism_layer2_summary",
				EvidenceCount: s.UniqueCounterparties,
			})
		}

	case "optimism-broad-operational-hub":
		appendOptimismReason(profile, model.RiskReason{
			Code:          "optimism_broad_operational_hub",
			Category:      "FRAUD",
			Description:   fmt.Sprintf("Broad mixed-flow operational Optimism surface observed (%d counterparties, %d selectors)", s.UniqueCounterparties, s.UniqueFunctionSelectors),
			Offset:        14,
			Source:        "dataset_optimism_layer2_summary",
			EvidenceCount: s.UniqueCounterparties,
		})

		if s.UniqueCounterparties >= 25000 {
			appendOptimismReason(profile, model.RiskReason{
				Code:          "optimism_extremely_broad_counterparty_surface",
				Category:      "FRAUD",
				Description:   fmt.Sprintf("Extremely broad Optimism counterparty surface observed (%d counterparties)", s.UniqueCounterparties),
				Offset:        8,
				Source:        "dataset_optimism_layer2_summary",
				EvidenceCount: s.UniqueCounterparties,
			})
		}

		if s.OutboundCount > 0 && s.InboundCount > 0 {
			appendOptimismReason(profile, model.RiskReason{
				Code:          "optimism_mixed_flow_operational_pattern",
				Category:      "REPUTATION",
				Description:   fmt.Sprintf("Mixed inbound/outbound operational pattern observed (%d inbound, %d outbound)", s.InboundCount, s.OutboundCount),
				Offset:        3,
				Source:        "dataset_optimism_layer2_summary",
				EvidenceCount: s.TxCount,
			})
		}
	}

	profile.RiskScore = optimismWeightedRiskScore(profile.RiskBreakdown)
	profile.RiskGrade = optimismDetermineGrade(profile.RiskScore)
	profile.ReviewRecommended = optimismShouldRecommendReview(profile)
}

func buildReportContextFromOptimismCase(cc *optimismCuratedLayer2Case) *reportContext {
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
		DatasetType: "Optimism Layer 2 dataset",
		CaseID:      cc.CaseID,
		CaseTitle:   cc.Title,
		Narrative:   cc.CurationNotes.Narrative,
		Interpretation: fmt.Sprintf(
			"Optimism Layer 2 case showing %s behavior with %d counterparties and %d function selectors.",
			blankFallback(s.DominantDirection, "mixed"),
			s.UniqueCounterparties,
			s.UniqueFunctionSelectors,
		),
		ChainContext: []string{
			fmt.Sprintf("Transactions: %d", s.TxCount),
			fmt.Sprintf("Inbound transfers: %d", s.InboundCount),
			fmt.Sprintf("Outbound transfers: %d", s.OutboundCount),
			fmt.Sprintf("Unique counterparties: %d", s.UniqueCounterparties),
			fmt.Sprintf("Unique function selectors: %d", s.UniqueFunctionSelectors),
			fmt.Sprintf("Dominant direction: %s", blankFallback(s.DominantDirection, "unknown")),
			fmt.Sprintf("Dominant counterparty share: %.2f%%", s.DominantCounterpartyShare),
			fmt.Sprintf("Dominant contract share: %.2f%%", s.DominantContractShare),
			fmt.Sprintf("Dominant function selector share: %.2f%%", s.DominantFunctionSelectorShare),
		},
		TopCounterparties: topCounterparties,
	}
}
func appendOptimismReason(profile *model.WalletProfile, reason model.RiskReason) {
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

func optimismWeightedRiskScore(breakdown model.RiskCategory) float64 {
	score := (breakdown.Fraud * 0.5) + (breakdown.Reputation * 0.3) + (breakdown.Lending * 0.2)
	return optimismRound(score, 2)
}

func optimismDetermineGrade(score float64) string {
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

func optimismShouldRecommendReview(profile *model.WalletProfile) bool {
	if profile == nil {
		return false
	}
	return profile.RiskScore >= 5
}

func optimismRound(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}
