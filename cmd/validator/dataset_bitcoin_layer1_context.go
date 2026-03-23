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

	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func buildWalletProfileFromBitcoinCuratedLayer1Case(cc *datasets.BitcoinCuratedLayer1Case) *model.WalletProfile {
	if cc == nil {
		return &model.WalletProfile{
			Network:           "BITCOIN",
			IsValid:           false,
			ValidationDetails: "Missing curated Bitcoin case",
		}
	}

	details := []string{
		"Loaded curated Bitcoin Layer 1 case",
		fmt.Sprintf("Case ID: %s", cc.CaseID),
		fmt.Sprintf("Risk Posture: %s", cc.RiskPosture),
		fmt.Sprintf("Source Rows: %d", cc.SourceRowCount),
		fmt.Sprintf("Dominant Role: %s", cc.UTXOSummary.DominantRole),
		fmt.Sprintf("Unique Counterparties: %d", cc.UTXOSummary.UniqueCounterparties),
	}

	return &model.WalletProfile{
		Address:           cc.Address,
		Network:           "BITCOIN",
		IsValid:           true,
		ValidationDetails: strings.Join(details, " | "),
	}
}

func applyBitcoinCuratedLayer1Context(profile *model.WalletProfile, curated *datasets.BitcoinCuratedLayer1Case) {
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
			Source:        "dataset_bitcoin_utxo_summary",
			EvidenceCount: evidenceCount,
		})

		switch category {
		case "FRAUD":
			fraudScore += offset
		case "REPUTATION":
			repScore += offset
		}
	}

	s := curated.UTXOSummary

	isLegacyMixedFlowBroadValue :=
		(strings.HasPrefix(curated.Address, "1") || strings.HasPrefix(curated.Address, "3")) &&
			s.InboundReceiptCount >= 100000 &&
			s.OutboundSpendCount >= 100000 &&
			s.UniqueCounterparties >= 50000

	// Legacy mixed-flow case gets its own stronger, more specific rule.
	// This prevents it from being flattened into the generic spend-heavy operational-hub bucket.
	if isLegacyMixedFlowBroadValue {
		appendDatasetReason(
			"bitcoin_legacy_mixed_flow_broad_value",
			"FRAUD",
			fmt.Sprintf(
				"Legacy mixed-flow broad-value pattern observed (%d inbound receipts, %d outbound spends, %d counterparties)",
				s.InboundReceiptCount,
				s.OutboundSpendCount,
				s.UniqueCounterparties,
			),
			16.0,
			s.InboundReceiptCount+s.OutboundSpendCount,
		)
	} else if strings.EqualFold(s.DominantRole, "outbound") &&
		s.OutboundSpendCount >= 100000 &&
		s.UniqueCounterparties >= 1000 {
		appendDatasetReason(
			"bitcoin_spend_heavy_operational_hub",
			"FRAUD",
			fmt.Sprintf(
				"Spend-heavy operational hub pattern observed (%d outbound spends, %d counterparties)",
				s.OutboundSpendCount,
				s.UniqueCounterparties,
			),
			18.0,
			s.OutboundSpendCount,
		)
	}

	if strings.EqualFold(s.DominantRole, "inbound") &&
		s.InboundReceiptCount >= 50000 &&
		s.UniqueCounterparties >= 10000 {
		appendDatasetReason(
			"bitcoin_noisy_inbound_broad_surface",
			"FRAUD",
			fmt.Sprintf(
				"Noisy inbound broad-surface pattern observed (%d inbound receipts, %d counterparties)",
				s.InboundReceiptCount,
				s.UniqueCounterparties,
			),
			14.0,
			s.InboundReceiptCount,
		)
	}

	if strings.EqualFold(s.DominantRole, "balanced") &&
		s.InboundReceiptCount >= 50000 &&
		s.OutboundSpendCount >= 50000 {
		appendDatasetReason(
			"bitcoin_balanced_high_volume_hub",
			"REPUTATION",
			fmt.Sprintf(
				"Balanced high-volume hub pattern observed (%d inbound, %d outbound)",
				s.InboundReceiptCount,
				s.OutboundSpendCount,
			),
			8.0,
			s.InboundReceiptCount+s.OutboundSpendCount,
		)
	}

	if s.UniqueCounterparties >= 50000 {
		appendDatasetReason(
			"bitcoin_extremely_broad_counterparty_surface",
			"FRAUD",
			fmt.Sprintf(
				"Extremely broad Bitcoin counterparty surface observed (%d counterparties)",
				s.UniqueCounterparties,
			),
			10.0,
			s.UniqueCounterparties,
		)
	} else if s.UniqueCounterparties >= 1000 {
		appendDatasetReason(
			"bitcoin_broad_counterparty_surface",
			"FRAUD",
			fmt.Sprintf(
				"Broad Bitcoin counterparty surface observed (%d counterparties)",
				s.UniqueCounterparties,
			),
			6.0,
			s.UniqueCounterparties,
		)
	}

	if len(curated.TopCounterparties) > 0 && curated.TopCounterparties[0].Interactions >= 100000 {
		appendDatasetReason(
			"bitcoin_extreme_repeated_counterparty_interaction",
			"FRAUD",
			fmt.Sprintf(
				"Extreme repeated interaction with top counterparty (%d interactions)",
				curated.TopCounterparties[0].Interactions,
			),
			10.0,
			curated.TopCounterparties[0].Interactions,
		)
	} else if len(curated.TopCounterparties) > 0 && curated.TopCounterparties[0].Interactions >= 10000 {
		appendDatasetReason(
			"bitcoin_repeated_counterparty_interaction",
			"FRAUD",
			fmt.Sprintf(
				"Heavy repeated interaction with top counterparty (%d interactions)",
				curated.TopCounterparties[0].Interactions,
			),
			4.0,
			curated.TopCounterparties[0].Interactions,
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
