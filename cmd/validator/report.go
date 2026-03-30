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

	if profile.Attribution != nil {
		lines = append(lines, "", "Attribution:")
		lines = append(lines, fmt.Sprintf("Resolved Label: %s", blankFallback(profile.Attribution.Label, "unknown")))
		if profile.Attribution.Actor != "" {
			lines = append(lines, fmt.Sprintf("Actor: %s", profile.Attribution.Actor))
		}
		lines = append(lines,
			fmt.Sprintf("Category: %s", blankFallback(string(profile.Attribution.Category), "UNKNOWN")),
			fmt.Sprintf("Risk Class: %s", blankFallback(string(profile.Attribution.RiskClass), "UNKNOWN")),
			fmt.Sprintf("Confidence: %.2f", profile.Attribution.Confidence),
			fmt.Sprintf("Primary Source: %s (%s / %s)", profile.Attribution.SourceName, profile.Attribution.SourceTier, profile.Attribution.SourceType),
			fmt.Sprintf("Disposition: %s", attributionDisposition(profile.Attribution)),
		)
		if profile.Attribution.BaseConfidence > 0 && math.Abs(profile.Attribution.Confidence-profile.Attribution.BaseConfidence) >= 0.01 {
			lines = append(lines, fmt.Sprintf("Confidence Basis: base %.2f, resolved %.2f", profile.Attribution.BaseConfidence, profile.Attribution.Confidence))
		}
		if len(profile.Attribution.CorroboratingSources) > 0 {
			lines = append(lines, fmt.Sprintf("Corroborating Sources: %s", joinAttributionSources(profile.Attribution.CorroboratingSources)))
		} else if len(profile.Attribution.SupportingSources) > 1 {
			lines = append(lines, fmt.Sprintf("Supporting Sources: %s", joinAttributionSources(profile.Attribution.SupportingSources[1:])))
		}
		if len(profile.Attribution.ConflictingSources) > 0 {
			lines = append(lines, fmt.Sprintf("Conflicting Sources: %s", joinAttributionSources(profile.Attribution.ConflictingSources)))
		}
	}

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

	if insights := topAttributionInsights(profile.AttributionInsights, 4); len(insights) > 0 {
		lines = append(lines, "", "Actor / Exposure Findings:")
		for idx, insight := range insights {
			lines = append(lines, fmt.Sprintf("%d. %s", idx+1, renderAttributionInsight(insight)))
		}
	}
	lines = append(lines, renderGraphSummary(profile.GraphSummary)...)
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
		lines = append(lines, "", "Chain Context:")
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

func topAttributionInsights(insights []model.AttributionInsight, limit int) []model.AttributionInsight {
	if len(insights) == 0 {
		return nil
	}

	sorted := append([]model.AttributionInsight(nil), insights...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if insightPriority(sorted[i]) != insightPriority(sorted[j]) {
			return insightPriority(sorted[i]) < insightPriority(sorted[j])
		}
		if sorted[i].Confidence != sorted[j].Confidence {
			return sorted[i].Confidence > sorted[j].Confidence
		}
		if sorted[i].EvidenceCount != sorted[j].EvidenceCount {
			return sorted[i].EvidenceCount > sorted[j].EvidenceCount
		}
		return sorted[i].Summary < sorted[j].Summary
	})

	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
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

func attributionDisposition(resolved *model.ResolvedAttribution) string {
	if resolved == nil {
		return "unknown"
	}
	if resolved.Escalating {
		return "risk-escalating"
	}
	if resolved.Contextual {
		return "contextual / benign"
	}
	return "informational"
}

func joinAttributionSources(sources []model.ResolvedAttributionSource) string {
	parts := make([]string, 0, len(sources))
	for _, source := range sources {
		description := blankFallback(source.Label, source.Actor)
		if description == "" || description == "unknown" {
			parts = append(parts, fmt.Sprintf("%s [%s]", source.Name, source.Tier))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s [%s: %s / %s]", source.Name, source.Tier, description, blankFallback(string(source.Category), "UNKNOWN")))
	}
	return strings.Join(parts, ", ")
}

func renderAttributionInsight(insight model.AttributionInsight) string {
	suffix := make([]string, 0, 2)
	if insight.Confidence > 0 {
		suffix = append(suffix, fmt.Sprintf("confidence %.2f", insight.Confidence))
	}
	if insight.HopDepth > 0 {
		suffix = append(suffix, fmt.Sprintf("%d-hop", insight.HopDepth))
	}
	if len(suffix) == 0 {
		return insight.Summary
	}
	return fmt.Sprintf("%s (%s)", insight.Summary, strings.Join(suffix, ", "))
}

func insightPriority(insight model.AttributionInsight) int {
	switch insight.Type {
	case model.AttributionInsightPassThrough, model.AttributionInsightUTurn:
		return 0
	case model.AttributionInsightNearExposure:
		return 1
	case model.AttributionInsightDirectExposure:
		return 2
	case model.AttributionInsightActorConcentration, model.AttributionInsightActorRepeated:
		return 3
	case model.AttributionInsightClusterGrouping:
		return 4
	default:
		return 5
	}
}
func renderGraphSummary(summary *model.GraphSummary) []string {
	if summary == nil {
		return nil
	}
	if summary.AttributedInteractions < 5 || summary.AttributedInteractionShare < 0.10 {
		return nil
	}

	lines := []string{
		"",
		"Graph Summary:",
		fmt.Sprintf(
			"- Attributed graph coverage: %d / %d sampled interactions (%.2f%%)",
			summary.AttributedInteractions,
			summary.TotalInteractions,
			summary.AttributedInteractionShare*100,
		),
		fmt.Sprintf("- Unique actors: %d", summary.UniqueActors),
		fmt.Sprintf("- Direct risky actors: %d", summary.DirectRiskyActorCount),
		fmt.Sprintf("- Direct contextual actors: %d", summary.DirectContextualActorCount),
		fmt.Sprintf("- Near risky actors: %d", summary.NearRiskyActorCount),
	}

	if summary.AttributedInteractions >= 10 && summary.UniqueActors >= 2 {
		lines = append(lines,
			fmt.Sprintf("- Top actor share: %.2f%%", summary.TopActorShare*100),
			fmt.Sprintf("- Concentration (HHI): %.4f", summary.ConcentrationHHI),
		)
	}

	if len(summary.TopActors) > 0 {
		lines = append(lines, "", "Top Actors:")
		limit := len(summary.TopActors)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			a := summary.TopActors[i]
			lines = append(
				lines,
				fmt.Sprintf(
					"%d. %s - %d interactions (%.2f%% share, %d inbound / %d outbound, %d addresses)",
					i+1,
					a.Actor,
					a.InteractionCount,
					a.Share*100,
					a.InboundCount,
					a.OutboundCount,
					a.UniqueAddresses,
				),
			)
		}
	}

	if len(summary.Motifs) > 0 {
		lines = append(lines, "", "Graph Motifs:")
		for i, motif := range summary.Motifs {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, motif.Summary))
		}
	}

	return lines
}
