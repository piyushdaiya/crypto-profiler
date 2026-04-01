package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type crosschainL2Member struct {
	Chain                 string  `json:"chain"`
	Address               string  `json:"address"`
	TxCount               int     `json:"tx_count"`
	InboundCount          int     `json:"inbound_count"`
	OutboundCount         int     `json:"outbound_count"`
	UniqueCounterparties  int     `json:"unique_counterparties"`
	DominantContractShare float64 `json:"dominant_contract_share"`
	FailureRatePct        float64 `json:"failure_rate_pct"`
	UniqueEmitters        int     `json:"unique_emitters"`
	UniqueTopic0s         int     `json:"unique_topic0s"`
	BridgeHitCount        int     `json:"bridge_hit_count"`
	ProtocolHitCount      int     `json:"protocol_hit_count"`
	StablecoinHitCount    int     `json:"stablecoin_hit_count"`
	ServiceHitCount       int     `json:"service_hit_count"`
	TopBridgeFamily       string  `json:"top_bridge_family"`
	TopProtocolFamily     string  `json:"top_protocol_family"`
	TopStablecoinFamily   string  `json:"top_stablecoin_family"`
}

type crosschainL2Summary struct {
	AddressCount                  int     `json:"address_count"`
	ChainCount                    int     `json:"chain_count"`
	TotalTxCount                  int     `json:"total_tx_count"`
	MaxDominantContractShare      float64 `json:"max_dominant_contract_share"`
	MaxUniqueCounterparties       int     `json:"max_unique_counterparties"`
	BridgeOrStablecoinMemberCount int     `json:"bridge_or_stablecoin_member_count"`
}

type crosschainL2Case struct {
	SchemaVersion     string               `json:"schema_version"`
	CaseFamily        string               `json:"case_family"`
	CaseID            string               `json:"case_id"`
	Title             string               `json:"title"`
	Description       string               `json:"description"`
	ChainsIncluded    []string             `json:"chains_included"`
	MemberCount       int                  `json:"member_count"`
	Members           []crosschainL2Member `json:"members"`
	CrosschainSummary crosschainL2Summary  `json:"crosschain_summary"`
	CurationNotes     struct {
		Narrative      string `json:"narrative"`
		SelectionBasis string `json:"selection_basis"`
	} `json:"curation_notes"`
}

func looksLikeCrosschainL2Case(raw string) bool {
	return strings.Contains(raw, `"case_family": "crosschain_l2"`) ||
		strings.Contains(raw, `"case_family":"crosschain_l2"`)
}

func buildWalletProfileFromCrosschainL2Case(cc *crosschainL2Case) *model.WalletProfile {
	if cc == nil {
		return nil
	}

	address := "multiple"
	if len(cc.Members) == 1 && cc.Members[0].Address != "" {
		address = cc.Members[0].Address
	}

	return &model.WalletProfile{
		Address:           address,
		Network:           "MULTI-CHAIN",
		IsValid:           true,
		ValidationDetails: fmt.Sprintf("Loaded curated cross-chain L2 case | Case ID: %s | Case Family: %s", cc.CaseID, cc.CaseFamily),
	}
}

func applyCrosschainL2Context(profile *model.WalletProfile, cc *crosschainL2Case) {
	if profile == nil || cc == nil {
		return
	}

	s := cc.CrosschainSummary

	switch cc.CaseID {
	case "crosschain-l2-repeated-contract-service-pattern":
		appendCrosschainReason(profile, model.RiskReason{
			Code:          "crosschain_multi_l2_service_pattern",
			Category:      "REPUTATION",
			Description:   fmt.Sprintf("Service-like repeated-contract pattern observed across %d L2s", s.ChainCount),
			Offset:        8,
			Source:        "dataset_crosschain_l2_summary",
			EvidenceCount: s.TotalTxCount,
		})

		if s.MaxDominantContractShare >= 95 {
			appendCrosschainReason(profile, model.RiskReason{
				Code:          "crosschain_extreme_contract_concentration",
				Category:      "REPUTATION",
				Description:   fmt.Sprintf("Extremely high dominant contract share observed across cross-chain case (max %.2f%%)", s.MaxDominantContractShare),
				Offset:        3,
				Source:        "dataset_crosschain_l2_summary",
				EvidenceCount: s.AddressCount,
			})
		}

	case "crosschain-l2-broad-operational-hub":
		appendCrosschainReason(profile, model.RiskReason{
			Code:          "crosschain_multi_l2_operational_hub",
			Category:      "FRAUD",
			Description:   fmt.Sprintf("Broad operational-hub pattern observed across %d L2s", s.ChainCount),
			Offset:        16,
			Source:        "dataset_crosschain_l2_summary",
			EvidenceCount: s.TotalTxCount,
		})

		if s.MaxUniqueCounterparties >= 100000 {
			appendCrosschainReason(profile, model.RiskReason{
				Code:          "crosschain_extremely_broad_counterparty_surface",
				Category:      "FRAUD",
				Description:   fmt.Sprintf("Extremely broad counterparty surface observed in cross-chain case (max %d counterparties)", s.MaxUniqueCounterparties),
				Offset:        6,
				Source:        "dataset_crosschain_l2_summary",
				EvidenceCount: s.MaxUniqueCounterparties,
			})
		}

		appendCrosschainReason(profile, model.RiskReason{
			Code:          "crosschain_operational_consistency_observed",
			Category:      "REPUTATION",
			Description:   fmt.Sprintf("Consistent mixed-flow operational behavior observed across %d L2s", s.ChainCount),
			Offset:        3,
			Source:        "dataset_crosschain_l2_summary",
			EvidenceCount: s.ChainCount,
		})

	case "crosschain-l2-stablecoin-bridge-operational-surface":
		appendCrosschainReason(profile, model.RiskReason{
			Code:          "crosschain_bridge_protocol_consistency",
			Category:      "FRAUD",
			Description:   fmt.Sprintf("Bridge- or stablecoin-adjacent operational surface observed across %d L2s", s.ChainCount),
			Offset:        10,
			Source:        "dataset_crosschain_l2_summary",
			EvidenceCount: s.BridgeOrStablecoinMemberCount,
		})

		appendCrosschainReason(profile, model.RiskReason{
			Code:          "crosschain_stablecoin_bridge_surface",
			Category:      "REPUTATION",
			Description:   fmt.Sprintf("Cross-chain bridge/stablecoin consistency observed across %d members", s.BridgeOrStablecoinMemberCount),
			Offset:        4,
			Source:        "dataset_crosschain_l2_summary",
			EvidenceCount: s.BridgeOrStablecoinMemberCount,
		})
	}

	profile.RiskScore = crosschainWeightedRiskScore(profile.RiskBreakdown)
	profile.RiskGrade = crosschainDetermineGrade(profile.RiskScore)
	profile.ReviewRecommended = crosschainShouldRecommendReview(profile)
}

func appendCrosschainReason(profile *model.WalletProfile, reason model.RiskReason) {
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

func crosschainWeightedRiskScore(breakdown model.RiskCategory) float64 {
	score := (breakdown.Fraud * 0.5) + (breakdown.Reputation * 0.3) + (breakdown.Lending * 0.2)
	return crosschainRound(score, 2)
}

func crosschainDetermineGrade(score float64) string {
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

func crosschainShouldRecommendReview(profile *model.WalletProfile) bool {
	if profile == nil {
		return false
	}
	return profile.RiskScore >= 5
}

func crosschainRound(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}
