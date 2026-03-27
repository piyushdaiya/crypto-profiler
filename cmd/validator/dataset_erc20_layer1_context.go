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
	"fmt"
	"math"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/analyzer"
	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func erc20CuratedLabelCategory(curated *datasets.ERC20CuratedLayer1Case) model.LabelCategory {
	if curated == nil {
		return ""
	}
	if label, ok := analyzer.LookupEntityLabel(curated.Address); ok {
		return label.Category
	}

	upperLabel := strings.ToUpper(strings.TrimSpace(curated.Label))
	switch {
	case strings.Contains(upperLabel, "EXCHANGE"):
		return model.LabelCategoryExchange
	case strings.Contains(upperLabel, "PROTOCOL"), strings.Contains(upperLabel, "TRUSTED"):
		return model.LabelCategoryProtocol
	case strings.Contains(upperLabel, "MIXER"):
		return model.LabelCategoryMixer
	default:
		return ""
	}
}

func buildWalletProfileFromERC20CuratedLayer1Case(cc *datasets.ERC20CuratedLayer1Case) *model.WalletProfile {
	if cc == nil {
		return &model.WalletProfile{
			Network:           "EVM",
			IsValid:           false,
			ValidationDetails: "Missing curated ERC-20 case",
		}
	}

	details := []string{
		"Loaded curated ERC-20 Layer 1 case",
		fmt.Sprintf("Case ID: %s", cc.CaseID),
		fmt.Sprintf("Risk Posture: %s", cc.RiskPosture),
		fmt.Sprintf("Source Rows: %d", cc.SourceRowCount),
		fmt.Sprintf("Dominant Direction: %s", cc.ERC20Summary.DominantDirection),
		fmt.Sprintf("Unique Counterparties: %d", cc.ERC20Summary.UniqueCounterparties),
		fmt.Sprintf("Unique Tokens: %d", cc.ERC20Summary.UniqueTokenContracts),
	}

	return &model.WalletProfile{
		Address:           cc.Address,
		Network:           "EVM",
		IsValid:           true,
		ValidationDetails: strings.Join(details, " | "),
	}
}

func applyERC20CuratedLayer1Context(profile *model.WalletProfile, curated *datasets.ERC20CuratedLayer1Case) {
	if profile == nil || curated == nil {
		return
	}

	var fraudScore float64
	var repScore float64

	appendDatasetReason := func(code, category, description string, offset float64, evidenceCount int) {
		profile.RiskReasons = append(profile.RiskReasons, model.RiskReason{
			Code:          code,
			Category:      category,
			Description:   description,
			Offset:        offset,
			Source:        "dataset_erc20_layer1_summary",
			EvidenceCount: evidenceCount,
		})

		switch category {
		case "FRAUD":
			fraudScore += offset
		case "REPUTATION":
			repScore += offset
		}
	}

	s := curated.ERC20Summary
	totalTransfers := s.InboundTransferCount + s.OutboundTransferCount
	topCounterpartyInteractions := 0
	if len(curated.TopCounterparties) > 0 {
		topCounterpartyInteractions = curated.TopCounterparties[0].Interactions
	}
	labelCategory := erc20CuratedLabelCategory(curated)

	switch labelCategory {
	case model.LabelCategoryProtocol, model.LabelCategoryTrusted:
		if totalTransfers >= 10000 && s.UniqueCounterparties >= 1000 {
			appendDatasetReason(
				"erc20_trusted_protocol_token_hub",
				"REPUTATION",
				fmt.Sprintf(
					"Trusted protocol ERC-20 hub observed (%d transfers across %d counterparties)",
					totalTransfers,
					s.UniqueCounterparties,
				),
				16.0,
				totalTransfers,
			)
		}

	case model.LabelCategoryExchange:
		if totalTransfers >= 1000 && s.UniqueCounterparties >= 500 {
			appendDatasetReason(
				"erc20_exchange_service_surface",
				"REPUTATION",
				fmt.Sprintf(
					"Exchange-like ERC-20 service surface observed (%d transfers across %d counterparties)",
					totalTransfers,
					s.UniqueCounterparties,
				),
				14.0,
				totalTransfers,
			)
		}
	}

	if labelCategory != model.LabelCategoryExchange &&
		labelCategory != model.LabelCategoryProtocol &&
		labelCategory != model.LabelCategoryTrusted &&
		strings.EqualFold(s.DominantDirection, "inbound") &&
		s.UniqueCounterparties >= 500 &&
		s.UniqueTokenContracts >= 25 {
		appendDatasetReason(
			"erc20_noisy_token_inbound_surface",
			"FRAUD",
			fmt.Sprintf(
				"Inbound-heavy ERC-20 surface observed (%d counterparties across %d tokens)",
				s.UniqueCounterparties,
				s.UniqueTokenContracts,
			),
			14.0,
			s.UniqueCounterparties,
		)
	}

	if s.UniqueCounterparties >= 1000 && s.UniqueTokenContracts >= 25 {
		appendDatasetReason(
			"erc20_broad_token_counterparty_surface",
			"FRAUD",
			fmt.Sprintf(
				"Broad ERC-20 counterparty surface observed (%d counterparties across %d tokens)",
				s.UniqueCounterparties,
				s.UniqueTokenContracts,
			),
			8.0,
			s.UniqueCounterparties,
		)
	}

	if s.UniqueTokenContracts >= 50 {
		appendDatasetReason(
			"erc20_mixed_token_activity",
			"REPUTATION",
			fmt.Sprintf(
				"Mixed ERC-20 activity observed across %d token contracts",
				s.UniqueTokenContracts,
			),
			4.0,
			s.UniqueTokenContracts,
		)
	}

	if s.RepeatedCounterparties >= 20 || topCounterpartyInteractions >= 100 {
		evidenceCount := s.RepeatedCounterparties
		if topCounterpartyInteractions > evidenceCount {
			evidenceCount = topCounterpartyInteractions
		}
		appendDatasetReason(
			"erc20_repeated_counterparty_activity",
			"FRAUD",
			fmt.Sprintf(
				"Repeated ERC-20 counterparty activity observed (top counterparty interactions=%d, repeated counterparties=%d)",
				topCounterpartyInteractions,
				s.RepeatedCounterparties,
			),
			4.0,
			evidenceCount,
		)
	}

	if s.DominantTokenTransferShare >= 75.0 && totalTransfers >= 1000 {
		appendDatasetReason(
			"erc20_single_token_operational_concentration",
			"REPUTATION",
			fmt.Sprintf(
				"Dominant token concentration observed (%s at %.2f%% of transfers)",
				s.DominantTokenSymbol,
				s.DominantTokenTransferShare,
			),
			3.0,
			totalTransfers,
		)
	}

	fraudScore = clampScore(fraudScore)
	repScore = clampScore(repScore)

	combinedRisk := (fraudScore * 0.5) + (repScore * 0.3)
	combinedRisk = math.Round(combinedRisk*100) / 100

	profile.RiskScore = combinedRisk
	profile.RiskGrade = determineDatasetGrade(combinedRisk)
	profile.ReviewRecommended = combinedRisk >= 5 || len(profile.RiskReasons) > 0
	profile.RiskBreakdown = model.RiskCategory{
		Fraud:      math.Round(fraudScore*100) / 100,
		Reputation: math.Round(repScore*100) / 100,
		Lending:    0,
	}

	if curated.CurationNotes.Narrative != "" {
		profile.ValidationDetails += " | Narrative: " + curated.CurationNotes.Narrative
	}
}
