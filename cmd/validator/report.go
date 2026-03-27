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
	"sort"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type reportContext struct {
	Mode              string
	DatasetType       string
	CaseID            string
	CaseTitle         string
	Narrative         string
	Interpretation    string
	ChainContext      []string
	TopCounterparties []reportCounterparty
}

type reportCounterparty struct {
	Address string
	Label   string
	Detail  string
}

func buildLiveReportContext(profile *model.WalletProfile) *reportContext {
	if profile == nil {
		return nil
	}

	interpretation := "Live validator result using the current chain strategy."
	if !profile.IsValid {
		interpretation = "Input did not resolve to a supported wallet format for the current validator strategies."
	}

	chainContext := make([]string, 0, 3)
	if profile.Balance != "" {
		chainContext = append(chainContext, "Balance: "+profile.Balance)
	}
	if profile.TxCount > 0 {
		chainContext = append(chainContext, fmt.Sprintf("Observed transaction count: %d", profile.TxCount))
	}
	if detail := strings.TrimSpace(profile.ValidationDetails); detail != "" {
		chainContext = append(chainContext, "Validation context: "+detail)
	}

	return &reportContext{
		Mode:           "live",
		Interpretation: interpretation,
		ChainContext:   chainContext,
	}
}

func buildReportContextFromCuratedCase(cc *datasets.CuratedCase) *reportContext {
	if cc == nil {
		return nil
	}

	chainContext := []string{
		fmt.Sprintf("Inbound transfers: %d", cc.Summary.InboundCount),
		fmt.Sprintf("Outbound transfers: %d", cc.Summary.OutboundCount),
		fmt.Sprintf("Unique counterparties: %d", cc.Summary.UniqueCounterparties),
		fmt.Sprintf("Native transfers: %d", cc.Summary.NativeTransferCount),
	}
	if cc.Summary.ERC20TransferCount > 0 {
		chainContext = append(chainContext, fmt.Sprintf("ERC-20 transfers: %d", cc.Summary.ERC20TransferCount))
	}
	if cc.TraceSummary != nil {
		chainContext = append(chainContext,
			fmt.Sprintf("Trace activity: %d internal traces", cc.TraceSourceCount),
			fmt.Sprintf("Max trace depth: %d", cc.TraceSummary.MaxDepth),
			fmt.Sprintf("Trace counterparties: %d", cc.TraceSummary.UniqueCounterparties),
		)
	}

	return &reportContext{
		Mode:              "dataset",
		DatasetType:       "Ethereum curated Layer 1 dataset",
		CaseID:            cc.CaseID,
		CaseTitle:         cc.Title,
		Narrative:         firstNonEmpty(cc.Description, cc.RiskPosture),
		Interpretation:    evmInterpretation(cc),
		ChainContext:      chainContext,
		TopCounterparties: renderEVMCounterparties(cc.TopCounterparties),
	}
}

func buildReportContextFromSolanaCase(cc *datasets.SolanaCuratedStablecoinCase) *reportContext {
	if cc == nil {
		return nil
	}

	s := cc.StablecoinSummary
	return &reportContext{
		Mode:        "dataset",
		DatasetType: "Solana stablecoin-flow Layer 1 dataset",
		CaseID:      cc.CaseID,
		CaseTitle:   cc.Title,
		Narrative:   firstNonEmpty(cc.CurationNotes.Narrative, cc.Description),
		Interpretation: fmt.Sprintf(
			"Solana stablecoin case centered on a %s-dominant flow pattern with %d counterparties.",
			blankFallback(s.DominantRole, "mixed"),
			s.UniqueCounterparties,
		),
		ChainContext: []string{
			fmt.Sprintf("Dominant role: %s", blankFallback(s.DominantRole, "unknown")),
			fmt.Sprintf("Dominant mint: %s", blankFallback(s.DominantMint, "unknown")),
			fmt.Sprintf("Authority-linked transfers: %d", s.AuthorityTransferCount),
			fmt.Sprintf("Source transfers: %d", s.SourceTransferCount),
			fmt.Sprintf("Destination transfers: %d", s.DestinationTransferCount),
			fmt.Sprintf("Unique counterparties: %d", s.UniqueCounterparties),
		},
		TopCounterparties: renderSolanaCounterparties(cc.TopCounterparties),
	}
}

func buildReportContextFromBitcoinCase(cc *datasets.BitcoinCuratedLayer1Case) *reportContext {
	if cc == nil {
		return nil
	}

	s := cc.UTXOSummary
	return &reportContext{
		Mode:        "dataset",
		DatasetType: "Bitcoin UTXO-flow Layer 1 dataset",
		CaseID:      cc.CaseID,
		CaseTitle:   cc.Title,
		Narrative:   firstNonEmpty(cc.CurationNotes.Narrative, cc.Description),
		Interpretation: fmt.Sprintf(
			"Bitcoin Layer 1 case built from address-level UTXO flow with a %s role and %d counterparties.",
			blankFallback(s.DominantRole, "mixed"),
			s.UniqueCounterparties,
		),
		ChainContext: []string{
			fmt.Sprintf("Dominant role: %s", blankFallback(s.DominantRole, "unknown")),
			fmt.Sprintf("Inbound receipts: %d", s.InboundReceiptCount),
			fmt.Sprintf("Outbound spends: %d", s.OutboundSpendCount),
			fmt.Sprintf("Unique counterparties: %d", s.UniqueCounterparties),
			fmt.Sprintf("Inbound BTC: %s", blankFallback(s.InboundValueBTC, "n/a")),
			fmt.Sprintf("Outbound BTC: %s", blankFallback(s.OutboundValueBTC, "n/a")),
		},
		TopCounterparties: renderBitcoinCounterparties(cc.TopCounterparties),
	}
}

func buildReportContextFromERC20Case(cc *datasets.ERC20CuratedLayer1Case) *reportContext {
	if cc == nil {
		return nil
	}

	s := cc.ERC20Summary
	return &reportContext{
		Mode:        "dataset",
		DatasetType: "ERC-20 Layer 1 dataset",
		CaseID:      cc.CaseID,
		CaseTitle:   cc.Title,
		Narrative:   firstNonEmpty(cc.CurationNotes.Narrative, cc.Description),
		Interpretation: fmt.Sprintf(
			"ERC-20 Layer 1 case showing a %s token-flow pattern across %d token contracts and %d counterparties.",
			blankFallback(s.DominantDirection, "mixed"),
			s.UniqueTokenContracts,
			s.UniqueCounterparties,
		),
		ChainContext: []string{
			fmt.Sprintf("Dominant direction: %s", blankFallback(s.DominantDirection, "unknown")),
			fmt.Sprintf("Unique token contracts: %d", s.UniqueTokenContracts),
			fmt.Sprintf("Unique counterparties: %d", s.UniqueCounterparties),
			fmt.Sprintf("Repeated counterparties: %d", s.RepeatedCounterparties),
			fmt.Sprintf("Dominant token: %s", blankFallback(s.DominantTokenSymbol, "unknown")),
			fmt.Sprintf("Dominant token transfer share: %.2f%%", s.DominantTokenTransferShare),
		},
		TopCounterparties: renderERC20Counterparties(cc.TopCounterparties),
	}
}

func renderReport(profile *model.WalletProfile, ctx *reportContext) string {
	if profile == nil {
		return "Crypto Profiler Analyst Report\nNo profile available.\n"
	}

	lines := []string{
		"Crypto Profiler Analyst Report",
		"",
		fmt.Sprintf("Address: %s", blankFallback(profile.Address, "unknown")),
		fmt.Sprintf("Network: %s", blankFallback(profile.Network, "unknown")),
	}

	if ctx != nil && ctx.Mode != "" {
		lines = append(lines, fmt.Sprintf("Mode: %s", strings.Title(ctx.Mode)))
	}
	if ctx != nil && ctx.DatasetType != "" {
		lines = append(lines, fmt.Sprintf("Dataset Context: %s", ctx.DatasetType))
	}
	if ctx != nil && ctx.CaseTitle != "" {
		caseLine := fmt.Sprintf("Case: %s", ctx.CaseTitle)
		if ctx.CaseID != "" {
			caseLine += fmt.Sprintf(" (%s)", ctx.CaseID)
		}
		lines = append(lines, caseLine)
	}

	lines = append(lines,
		fmt.Sprintf("Risk Score: %.2f", profile.RiskScore),
		fmt.Sprintf("Risk Grade: %s", blankFallback(profile.RiskGrade, "ungraded")),
		fmt.Sprintf("Review Recommended: %s", yesNo(profile.ReviewRecommended)),
	)

	if reasons := topRiskReasons(profile.RiskReasons, 5); len(reasons) > 0 {
		lines = append(lines, "", "Top Reasons:")
		for idx, reason := range reasons {
			lines = append(lines, fmt.Sprintf(
				"%d. [%s] %s (offset %s%.1f)",
				idx+1,
				blankFallback(reason.Category, "UNKNOWN"),
				reason.Description,
				offsetPrefix(reason.Offset),
				math.Abs(reason.Offset),
			))
		}
	}

	if ctx != nil && len(ctx.TopCounterparties) > 0 {
		lines = append(lines, "", "Top Counterparties:")
		for idx, cp := range limitCounterparties(ctx.TopCounterparties, 5) {
			entry := fmt.Sprintf("%d. %s", idx+1, cp.Address)
			if cp.Label != "" {
				entry += " [" + cp.Label + "]"
			}
			if cp.Detail != "" {
				entry += " - " + cp.Detail
			}
			lines = append(lines, entry)
		}
	}

	narrative := ""
	if ctx != nil {
		narrative = firstNonEmpty(ctx.Narrative, ctx.Interpretation)
	}
	if narrative == "" {
		narrative = profile.ValidationDetails
	}
	if narrative != "" {
		lines = append(lines, "", "Interpretation:", narrative)
	}

	if ctx != nil && len(ctx.ChainContext) > 0 {
		lines = append(lines, "", "Layer 1 Context:")
		for _, item := range ctx.ChainContext {
			lines = append(lines, "- "+item)
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func topRiskReasons(reasons []model.RiskReason, limit int) []model.RiskReason {
	if len(reasons) == 0 {
		return nil
	}

	sorted := append([]model.RiskReason(nil), reasons...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := math.Abs(sorted[i].Offset)
		right := math.Abs(sorted[j].Offset)
		if left != right {
			return left > right
		}
		return sorted[i].Description < sorted[j].Description
	})

	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func limitCounterparties(counterparties []reportCounterparty, limit int) []reportCounterparty {
	if limit > 0 && len(counterparties) > limit {
		return counterparties[:limit]
	}
	return counterparties
}

func renderEVMCounterparties(counterparties []datasets.CounterpartySummary) []reportCounterparty {
	out := make([]reportCounterparty, 0, len(counterparties))
	for _, cp := range counterparties {
		out = append(out, reportCounterparty{
			Address: cp.Address,
			Label:   cp.Label,
			Detail:  fmt.Sprintf("%d interactions (%d inbound / %d outbound)", cp.Interactions, cp.InboundCount, cp.OutboundCount),
		})
	}
	return out
}

func renderSolanaCounterparties(counterparties []datasets.SolanaStablecoinCounterparty) []reportCounterparty {
	out := make([]reportCounterparty, 0, len(counterparties))
	for _, cp := range counterparties {
		out = append(out, reportCounterparty{
			Address: cp.Address,
			Detail: fmt.Sprintf(
				"%d interactions (%d inbound / %d outbound / %d authority)",
				cp.Interactions,
				cp.InboundCount,
				cp.OutboundCount,
				cp.AuthorityCount,
			),
		})
	}
	return out
}

func renderBitcoinCounterparties(counterparties []datasets.BitcoinCounterparty) []reportCounterparty {
	out := make([]reportCounterparty, 0, len(counterparties))
	for _, cp := range counterparties {
		out = append(out, reportCounterparty{
			Address: cp.Address,
			Detail:  fmt.Sprintf("%d interactions (%d inbound / %d outbound)", cp.Interactions, cp.InboundCount, cp.OutboundCount),
		})
	}
	return out
}

func renderERC20Counterparties(counterparties []datasets.ERC20CounterpartySummary) []reportCounterparty {
	out := make([]reportCounterparty, 0, len(counterparties))
	for _, cp := range counterparties {
		out = append(out, reportCounterparty{
			Address: cp.Address,
			Label:   cp.Label,
			Detail:  fmt.Sprintf("%d interactions across %d tokens", cp.Interactions, cp.UniqueTokens),
		})
	}
	return out
}

func evmInterpretation(cc *datasets.CuratedCase) string {
	if cc == nil {
		return ""
	}

	parts := []string{
		fmt.Sprintf(
			"Ethereum curated Layer 1 case built from %d transfer rows and %d counterparties.",
			cc.SourceTransferCount,
			cc.Summary.UniqueCounterparties,
		),
	}
	if cc.TraceSummary != nil {
		parts = append(parts, "Trace enrichment adds internal call context beyond plain transfer rows.")
	}
	return strings.Join(parts, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func blankFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func offsetPrefix(offset float64) string {
	if offset < 0 {
		return "-"
	}
	return "+"
}
