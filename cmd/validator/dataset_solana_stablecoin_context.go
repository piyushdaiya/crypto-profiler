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

func buildWalletProfileFromSolanaCuratedStablecoinCase(cc *datasets.SolanaCuratedStablecoinCase) *model.WalletProfile {
	if cc == nil {
		return &model.WalletProfile{
			Network:           "SOLANA",
			IsValid:           false,
			ValidationDetails: "Missing curated Solana case",
		}
	}

	details := []string{
		"Loaded curated Solana stablecoin-flow case",
		fmt.Sprintf("Case ID: %s", cc.CaseID),
		fmt.Sprintf("Risk Posture: %s", cc.RiskPosture),
		fmt.Sprintf("Source Rows: %d", cc.SourceRowCount),
		fmt.Sprintf("Dominant Role: %s", cc.StablecoinSummary.DominantRole),
		fmt.Sprintf("Dominant Mint: %s", cc.StablecoinSummary.DominantMint),
		fmt.Sprintf("Unique Counterparties: %d", cc.StablecoinSummary.UniqueCounterparties),
	}

	return &model.WalletProfile{
		Address:           cc.Address,
		Network:           "SOLANA",
		IsValid:           true,
		ValidationDetails: strings.Join(details, " | "),
	}
}
func applySolanaCuratedStablecoinContext(profile *model.WalletProfile, curated *datasets.SolanaCuratedStablecoinCase) {
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
			Source:        "dataset_solana_stablecoin_summary",
			EvidenceCount: evidenceCount,
		})

		switch category {
		case "FRAUD":
			fraudScore += offset
		case "REPUTATION":
			repScore += offset
		}
	}

	s := curated.StablecoinSummary
	mintCount := len(curated.MintBreakdown)

	if strings.EqualFold(s.DominantRole, "source") &&
		s.SourceTransferCount >= 100000 &&
		s.SourceCounterparties >= 500 {
		appendDatasetReason(
			"solana_source_heavy_stablecoin_distributor",
			"REPUTATION",
			fmt.Sprintf(
				"Source-heavy stablecoin distributor pattern observed (%d outbound transfers, %d source counterparties)",
				s.SourceTransferCount,
				s.SourceCounterparties,
			),
			12.0,
			s.SourceTransferCount,
		)
	}

	if strings.EqualFold(s.DominantRole, "authority") &&
		s.AuthorityTransferCount >= 10000 {
		appendDatasetReason(
			"solana_authority_heavy_stablecoin_operator",
			"FRAUD",
			fmt.Sprintf(
				"Authority-heavy stablecoin operator pattern observed (%d authority-linked transfers)",
				s.AuthorityTransferCount,
			),
			28.0,
			s.AuthorityTransferCount,
		)
	}

	// Mutually exclusive breadth rules so we do not double-count:
	// - very broad + mixed stablecoin surface gets the stronger rule
	// - otherwise broad surface gets the base rule
	if s.UniqueCounterparties >= 5000 && mintCount >= 2 {
		appendDatasetReason(
			"solana_broad_mixed_stablecoin_surface",
			"FRAUD",
			fmt.Sprintf(
				"Very broad mixed stablecoin counterparty surface observed (%d counterparties across %d mints)",
				s.UniqueCounterparties,
				mintCount,
			),
			12.0,
			s.UniqueCounterparties,
		)
	} else if s.UniqueCounterparties >= 1000 {
		appendDatasetReason(
			"solana_broad_stablecoin_counterparty_surface",
			"FRAUD",
			fmt.Sprintf(
				"Broad stablecoin counterparty surface observed (%d counterparties)",
				s.UniqueCounterparties,
			),
			8.0,
			s.UniqueCounterparties,
		)
	}

	if mintCount >= 2 {
		appendDatasetReason(
			"solana_mixed_stablecoin_activity",
			"REPUTATION",
			fmt.Sprintf(
				"Mixed stablecoin activity observed across %d mints",
				mintCount,
			),
			4.0,
			mintCount,
		)
	}

	if len(curated.TopCounterparties) > 0 &&
		curated.TopCounterparties[0].Interactions >= 10000 {
		appendDatasetReason(
			"solana_repeated_large_counterparty_interaction",
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

func clampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
func determineDatasetGrade(score float64) string {
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
