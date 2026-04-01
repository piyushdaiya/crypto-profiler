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
	"encoding/json"
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
	MemberSummaries   []reportMemberSummary
}

type reportCounterparty struct {
	Address string
	Label   string
	Detail  string
}

type reportMemberSummary struct {
	Chain                 string
	Address               string
	TxCount               int
	InboundCount          int
	OutboundCount         int
	UniqueCounterparties  int
	DominantContractShare float64
	FailureRatePct        float64
	TopBridgeFamily       string
	TopProtocolFamily     string
	TopStablecoinFamily   string
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
		return "Crypto Profiler Analyst Report\n\nNo profile available."
	}

	lines := []string{
		"Crypto Profiler Analyst Report",
		"",
		fmt.Sprintf("Address: %s", blankFallback(profile.Address, "unknown")),
		fmt.Sprintf("Network: %s", blankFallback(profile.Network, "unknown")),
	}

	if ctx != nil {
		if strings.TrimSpace(ctx.Mode) != "" {
			lines = append(lines, fmt.Sprintf("Mode: %s", strings.Title(strings.ToLower(ctx.Mode))))
		}
		if strings.TrimSpace(ctx.DatasetType) != "" {
			lines = append(lines, fmt.Sprintf("Dataset Context: %s", ctx.DatasetType))
		}
		if strings.TrimSpace(ctx.CaseTitle) != "" || strings.TrimSpace(ctx.CaseID) != "" {
			lines = append(lines, fmt.Sprintf("Case: %s (%s)", blankFallback(ctx.CaseTitle, "Untitled Case"), blankFallback(ctx.CaseID, "unknown-case")))
		}
	}

	lines = append(lines,
		fmt.Sprintf("Risk Score: %.2f", profile.RiskScore),
		fmt.Sprintf("Risk Grade: %s", blankFallback(profile.RiskGrade, "UNKNOWN")),
		fmt.Sprintf("Review Recommended: %s", yesNo(profile.ReviewRecommended)),
	)

	if attributionLines := renderAttributionSummary(profile); len(attributionLines) > 0 {
		lines = append(lines, attributionLines...)
	}

	if topReasons := renderTopReasons(profile); len(topReasons) > 0 {
		lines = append(lines, topReasons...)
	}

	if insightLines := renderAttributionInsights(profile); len(insightLines) > 0 {
		lines = append(lines, insightLines...)
	}

	if topCounterpartyLines := renderTopCounterparties(ctx); len(topCounterpartyLines) > 0 {
		lines = append(lines, topCounterpartyLines...)
	}

	if ctx != nil && strings.TrimSpace(ctx.Interpretation) != "" {
		lines = append(lines, "", "Interpretation:", ctx.Interpretation)
	}

	if ctx != nil {
		if memberLines := renderMemberSummaries(ctx.MemberSummaries); len(memberLines) > 0 {
			lines = append(lines, memberLines...)
		}
	}

	if graphLines := renderGraphSummary(profile.GraphSummary); len(graphLines) > 0 {
		lines = append(lines, graphLines...)
	}

	if ctx != nil && len(ctx.ChainContext) > 0 {
		lines = append(lines, "", "Chain Context:")
		for _, item := range ctx.ChainContext {
			if strings.TrimSpace(item) == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("- %s", item))
		}
	}

	return strings.Join(lines, "\n")
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

func renderMemberSummaries(members []reportMemberSummary) []string {
	if len(members) == 0 {
		return nil
	}

	sorted := make([]reportMemberSummary, len(members))
	copy(sorted, members)

	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].TxCount == sorted[j].TxCount {
			if sorted[i].Chain == sorted[j].Chain {
				return sorted[i].Address < sorted[j].Address
			}
			return sorted[i].Chain < sorted[j].Chain
		}
		return sorted[i].TxCount > sorted[j].TxCount
	})

	lines := []string{"", "Cross-Chain Members:"}

	limit := len(sorted)
	if limit > 5 {
		limit = 5
	}

	for i := 0; i < limit; i++ {
		m := sorted[i]

		base := fmt.Sprintf(
			"%d. [%s] %s - %d txs (%d inbound / %d outbound, %d counterparties, %.2f%% dominant contract, %.2f%% failure rate)",
			i+1,
			blankFallback(m.Chain, "UNKNOWN"),
			blankFallback(m.Address, "unknown"),
			m.TxCount,
			m.InboundCount,
			m.OutboundCount,
			m.UniqueCounterparties,
			m.DominantContractShare,
			m.FailureRatePct,
		)

		families := make([]string, 0, 3)
		if strings.TrimSpace(m.TopBridgeFamily) != "" {
			families = append(families, fmt.Sprintf("bridge=%s", m.TopBridgeFamily))
		}
		if strings.TrimSpace(m.TopProtocolFamily) != "" {
			families = append(families, fmt.Sprintf("protocol=%s", m.TopProtocolFamily))
		}
		if strings.TrimSpace(m.TopStablecoinFamily) != "" {
			families = append(families, fmt.Sprintf("stablecoin=%s", m.TopStablecoinFamily))
		}

		if len(families) > 0 {
			base = fmt.Sprintf("%s [%s]", base, strings.Join(families, ", "))
		}

		lines = append(lines, base)
	}

	return lines
}

func buildReportContextFromCrosschainL2Case(cc *crosschainL2Case) *reportContext {
	if cc == nil {
		return nil
	}

	s := cc.CrosschainSummary

	memberSummaries := make([]reportMemberSummary, 0, len(cc.Members))
	for _, m := range cc.Members {
		memberSummaries = append(memberSummaries, reportMemberSummary{
			Chain:                 m.Chain,
			Address:               m.Address,
			TxCount:               m.TxCount,
			InboundCount:          m.InboundCount,
			OutboundCount:         m.OutboundCount,
			UniqueCounterparties:  m.UniqueCounterparties,
			DominantContractShare: m.DominantContractShare,
			FailureRatePct:        m.FailureRatePct,
			TopBridgeFamily:       m.TopBridgeFamily,
			TopProtocolFamily:     m.TopProtocolFamily,
			TopStablecoinFamily:   m.TopStablecoinFamily,
		})
	}

	chainContext := []string{
		fmt.Sprintf("Chains included: %s", strings.Join(cc.ChainsIncluded, ", ")),
		fmt.Sprintf("Members: %d", cc.MemberCount),
		fmt.Sprintf("Unique addresses: %d", s.AddressCount),
		fmt.Sprintf("Total transactions: %d", s.TotalTxCount),
	}
	if s.MaxDominantContractShare > 0 {
		chainContext = append(chainContext, fmt.Sprintf("Max dominant contract share: %.2f%%", s.MaxDominantContractShare))
	}
	if s.MaxUniqueCounterparties > 0 {
		chainContext = append(chainContext, fmt.Sprintf("Max unique counterparties: %d", s.MaxUniqueCounterparties))
	}
	if s.BridgeOrStablecoinMemberCount > 0 {
		chainContext = append(chainContext, fmt.Sprintf("Bridge/stablecoin members: %d", s.BridgeOrStablecoinMemberCount))
	}

	return &reportContext{
		Mode:            "dataset",
		DatasetType:     "Cross-chain L2 dataset",
		CaseID:          cc.CaseID,
		CaseTitle:       cc.Title,
		Narrative:       cc.CurationNotes.Narrative,
		Interpretation:  cc.CurationNotes.Narrative,
		MemberSummaries: memberSummaries,
		ChainContext:    chainContext,
	}
}

func renderTopReasons(profile *model.WalletProfile) []string {
	if profile == nil || len(profile.RiskReasons) == 0 {
		return nil
	}

	reasons := topRiskReasons(profile.RiskReasons, 5)
	if len(reasons) == 0 {
		return nil
	}

	lines := []string{"", "Top Reasons:"}
	for i, r := range reasons {
		lines = append(lines,
			fmt.Sprintf(
				"%d. [%s] %s (offset %+0.1f)",
				i+1,
				blankFallback(string(r.Category), "UNKNOWN"),
				blankFallback(r.Description, r.Code),
				r.Offset,
			),
		)
	}

	return lines
}

func renderAttributionSummary(profile *model.WalletProfile) []string {
	if profile == nil || profile.Attribution == nil {
		return nil
	}

	resolved := profile.Attribution
	lines := []string{"", "Attribution:"}

	resolvedLabel := blankFallback(resolved.Label, resolved.Actor)
	if strings.TrimSpace(resolvedLabel) != "" && resolvedLabel != "unknown" {
		lines = append(lines, fmt.Sprintf("Resolved Label: %s", resolvedLabel))
	}
	if actor := strings.TrimSpace(resolved.Actor); actor != "" {
		lines = append(lines, fmt.Sprintf("Actor: %s", actor))
	}
	if category := strings.TrimSpace(string(resolved.Category)); category != "" {
		lines = append(lines, fmt.Sprintf("Category: %s", category))
	}
	if riskClass := strings.TrimSpace(string(resolved.RiskClass)); riskClass != "" {
		lines = append(lines, fmt.Sprintf("Risk Class: %s", riskClass))
	}
	if resolved.Confidence > 0 {
		lines = append(lines, fmt.Sprintf("Confidence: %.2f", resolved.Confidence))
	}

	if sourceName := strings.TrimSpace(resolved.SourceName); sourceName != "" {
		lines = append(lines, fmt.Sprintf(
			"Primary Source: %s (%s / %s)",
			sourceName,
			blankFallback(string(resolved.SourceTier), "UNKNOWN"),
			blankFallback(string(resolved.SourceType), "UNKNOWN"),
		))
	}

	lines = append(lines, fmt.Sprintf("Disposition: %s", attributionDisposition(resolved)))

	if resolved.BaseConfidence > 0 || resolved.Confidence > 0 {
		lines = append(lines, fmt.Sprintf(
			"Confidence Basis: base %.2f, resolved %.2f",
			resolved.BaseConfidence,
			resolved.Confidence,
		))
	}

	if len(resolved.CorroboratingSources) > 0 {
		lines = append(lines, fmt.Sprintf(
			"Corroborating Sources: %s",
			joinAttributionSources(resolved.CorroboratingSources),
		))
	}

	if len(resolved.ConflictingSources) > 0 {
		lines = append(lines, fmt.Sprintf(
			"Conflicting Sources: %s",
			joinAttributionSources(resolved.ConflictingSources),
		))
	}

	return lines
}

func renderAttributionInsights(profile *model.WalletProfile) []string {
	if profile == nil || len(profile.AttributionInsights) == 0 {
		return nil
	}

	insights := topAttributionInsights(profile.AttributionInsights, 5)
	if len(insights) == 0 {
		return nil
	}

	lines := []string{"", "Actor / Exposure Findings:"}
	for i, insight := range insights {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, renderAttributionInsight(insight)))
	}

	return lines
}

func renderTopCounterparties(ctx *reportContext) []string {
	if ctx == nil || len(ctx.TopCounterparties) == 0 {
		return nil
	}

	counterparties := limitCounterparties(ctx.TopCounterparties, 5)
	if len(counterparties) == 0 {
		return nil
	}

	lines := []string{"", "Top Counterparties:"}
	for i, cp := range counterparties {
		address := blankFallback(cp.Address, "unknown")
		detail := blankFallback(cp.Detail, "observed")
		lines = append(lines, fmt.Sprintf("%d. %s - %s", i+1, address, detail))
	}

	return lines
}
func asInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case json.Number:
		i, _ := t.Int64()
		return i
	default:
		return 0
	}
}

func asFloat64(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func formatCrosschainSummaryValue(key string, value any) string {
	switch key {
	case "max_dominant_contract_share", "max_failure_rate_pct":
		return fmt.Sprintf("%.2f%%", asFloat64(value))
	case "address_count",
		"chain_count",
		"total_tx_count",
		"max_unique_counterparties",
		"bridge_or_stablecoin_member_count",
		"member_count",
		"unique_address_count":
		return fmt.Sprintf("%d", asInt64(value))
	default:
		switch value.(type) {
		case int, int64:
			return fmt.Sprintf("%d", asInt64(value))
		case float64, float32:
			return fmt.Sprintf("%.2f", asFloat64(value))
		default:
			return fmt.Sprintf("%v", value)
		}
	}
}
