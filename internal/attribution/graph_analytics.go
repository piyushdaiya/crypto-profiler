package attribution

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type FlowEdge struct {
	Source      string
	Destination string
}

type ResolveAttributionFunc func(address string) *model.ResolvedAttribution

type actorAccumulator struct {
	Actor           string
	Category        string
	RiskClass       string
	Contextual      bool
	Escalating      bool
	Confidence      float64
	InboundCount    int
	OutboundCount   int
	UniqueAddresses map[string]struct{}
}

func BuildGraphSummaryFromWave5CInput(
	profiled string,
	input Wave5CInput,
	resolve ResolveAttributionFunc,
) *model.GraphSummary {
	if strings.TrimSpace(profiled) == "" || resolve == nil {
		return nil
	}

	edges := buildFlowEdgesFromWave5CInput(profiled, input)
	if len(edges) > 0 {
		return BuildGraphSummary(profiled, edges, resolve)
	}

	// Fallback: synthesize simple edges from counterparties when sampled flows
	// are too sparse to give a useful ordered graph.
	fallbackEdges := make([]FlowEdge, 0, len(input.Counterparties))
	profiledNorm := normalizeAddr(profiled)

	for _, cp := range input.Counterparties {
		addr := normalizeAddr(cp.Address)
		if addr == "" || addr == profiledNorm {
			continue
		}

		switch {
		case cp.OutboundCount > cp.InboundCount:
			fallbackEdges = append(fallbackEdges, FlowEdge{
				Source:      profiledNorm,
				Destination: addr,
			})
		case cp.InboundCount > 0:
			fallbackEdges = append(fallbackEdges, FlowEdge{
				Source:      addr,
				Destination: profiledNorm,
			})
		default:
			fallbackEdges = append(fallbackEdges, FlowEdge{
				Source:      profiledNorm,
				Destination: addr,
			})
		}
	}

	if len(fallbackEdges) == 0 {
		return nil
	}

	return BuildGraphSummary(profiled, fallbackEdges, resolve)
}

func BuildGraphSummary(
	profiled string,
	edges []FlowEdge,
	resolve ResolveAttributionFunc,
) *model.GraphSummary {
	if strings.TrimSpace(profiled) == "" || len(edges) == 0 || resolve == nil {
		return nil
	}

	profiled = normalizeAddr(profiled)

	outAdj := make(map[string][]string)
	inAdj := make(map[string][]string)

	directCounterpartyAttr := make(map[string]*model.ResolvedAttribution)
	directActorStats := make(map[string]*actorAccumulator)
	directRiskyActors := make(map[string]struct{})
	directContextualActors := make(map[string]struct{})

	totalDirectInteractions := 0

	for _, e := range edges {
		src := normalizeAddr(e.Source)
		dst := normalizeAddr(e.Destination)

		if src == "" || dst == "" || src == dst {
			continue
		}

		outAdj[src] = append(outAdj[src], dst)
		inAdj[dst] = append(inAdj[dst], src)

		counterparty, direction, ok := extractCounterparty(profiled, src, dst)
		if !ok {
			continue
		}

		totalDirectInteractions++

		attr := resolve(counterparty)
		if attr == nil {
			continue
		}

		directCounterpartyAttr[counterparty] = attr

		actorKey := resolvedActorKey(attr)
		if actorKey == "" {
			continue
		}

		acc := directActorStats[actorKey]
		if acc == nil {
			acc = &actorAccumulator{
				Actor:           actorDisplayName(attr),
				Category:        string(attr.Category),
				RiskClass:       string(attr.RiskClass),
				Contextual:      attr.Contextual,
				Escalating:      attrEscalating(attr),
				Confidence:      attr.Confidence,
				UniqueAddresses: map[string]struct{}{},
			}
			directActorStats[actorKey] = acc
		}

		acc.UniqueAddresses[counterparty] = struct{}{}
		if direction == "inbound" {
			acc.InboundCount++
		} else {
			acc.OutboundCount++
		}

		if attrEscalating(attr) {
			directRiskyActors[actorKey] = struct{}{}
		}
		if attr.Contextual {
			directContextualActors[actorKey] = struct{}{}
		}
	}

	if totalDirectInteractions == 0 {
		return nil
	}

	summary := &model.GraphSummary{
		TotalInteractions:          totalDirectInteractions,
		DirectRiskyActorCount:      len(directRiskyActors),
		DirectContextualActorCount: len(directContextualActors),
	}

	if len(directActorStats) == 0 {
		return summary
	}

	topActors := make([]model.GraphActorSummary, 0, len(directActorStats))
	attributedInteractions := 0
	hhi := 0.0

	for _, acc := range directActorStats {
		count := acc.InboundCount + acc.OutboundCount
		attributedInteractions += count
	}

	for _, acc := range directActorStats {
		count := acc.InboundCount + acc.OutboundCount
		share := 0.0
		if attributedInteractions > 0 {
			share = float64(count) / float64(attributedInteractions)
			hhi += share * share
		}

		topActors = append(topActors, model.GraphActorSummary{
			Actor:            acc.Actor,
			Category:         acc.Category,
			RiskClass:        acc.RiskClass,
			Contextual:       acc.Contextual,
			RiskEscalating:   acc.Escalating,
			Confidence:       acc.Confidence,
			InteractionCount: count,
			InboundCount:     acc.InboundCount,
			OutboundCount:    acc.OutboundCount,
			UniqueAddresses:  len(acc.UniqueAddresses),
			Share:            roundFloat(share, 4),
		})
	}

	sort.Slice(topActors, func(i, j int) bool {
		if topActors[i].InteractionCount == topActors[j].InteractionCount {
			return topActors[i].Actor < topActors[j].Actor
		}
		return topActors[i].InteractionCount > topActors[j].InteractionCount
	})

	summary.AttributedInteractions = attributedInteractions
	summary.AttributedInteractionShare = roundRatio(attributedInteractions, totalDirectInteractions)
	summary.UniqueActors = len(topActors)
	summary.TopActors = topActors
	summary.ConcentrationHHI = roundFloat(hhi, 4)

	if len(topActors) > 0 {
		summary.TopActorShare = roundFloat(topActors[0].Share, 4)
	}

	summary.NearRiskyActorCount = countNearRiskyActors(profiled, directCounterpartyAttr, outAdj, inAdj, resolve)
	summary.Motifs = buildGraphMotifs(directActorStats)

	return summary
}

func ApplyGraphSummaryContext(profile *model.WalletProfile) {
	if profile == nil || profile.GraphSummary == nil {
		return
	}

	summary := profile.GraphSummary
	if summary.AttributedInteractions < 5 || summary.AttributedInteractionShare < 0.10 {
		return
	}

	if len(summary.TopActors) > 0 {
		top := summary.TopActors[0]

		if top.RiskEscalating && top.Share >= 0.60 && top.InteractionCount >= 10 {
			appendGraphReason(
				profile,
				"graph_risky_actor_concentration",
				"FRAUD",
				fmt.Sprintf("Graph summary is concentrated on risky actor %s (%.2f%% of attributed interactions)", top.Actor, top.Share*100),
				2.0,
				top.InteractionCount,
			)
		}

		if top.Contextual && top.Share >= 0.80 && top.InteractionCount >= 10 && top.UniqueAddresses >= 2 {
			appendGraphReason(
				profile,
				"graph_contextual_actor_concentration",
				"REPUTATION",
				fmt.Sprintf("Graph summary is heavily concentrated on contextual actor %s (%.2f%% of attributed interactions)", top.Actor, top.Share*100),
				-1.0,
				top.InteractionCount,
			)
		}
	}

	if summary.NearRiskyActorCount > 0 {
		appendGraphReason(
			profile,
			"graph_near_risky_actor_exposure",
			"FRAUD",
			fmt.Sprintf("Graph summary observed near exposure to %d risky actor(s) within 2 hops of sampled flows", summary.NearRiskyActorCount),
			2.0,
			summary.NearRiskyActorCount,
		)
	}

	for _, motif := range summary.Motifs {
		switch motif.Code {
		case "contextual_to_risky_pass_through":
			appendGraphReason(
				profile,
				"graph_contextual_to_risky_pass_through",
				"FRAUD",
				motif.Summary,
				2.0,
				motif.Count,
			)
		case "risky_actor_u_turn":
			appendGraphReason(
				profile,
				"graph_risky_actor_u_turn",
				"FRAUD",
				motif.Summary,
				2.0,
				motif.Count,
			)
		case "risky_actor_fan_out":
			appendGraphReason(
				profile,
				"graph_risky_actor_fan_out",
				"FRAUD",
				motif.Summary,
				1.5,
				motif.Count,
			)
		case "contextual_fan_in_hub":
			appendGraphReason(
				profile,
				"graph_contextual_fan_in_hub",
				"REPUTATION",
				motif.Summary,
				-1.0,
				motif.Count,
			)
		case "contextual_fan_out_hub":
			appendGraphReason(
				profile,
				"graph_contextual_fan_out_hub",
				"REPUTATION",
				motif.Summary,
				-0.5,
				motif.Count,
			)
		}
	}

	profile.RiskScore = roundFloat(clampMin(profile.RiskScore, 0), 2)
	profile.RiskGrade = graphAdjustedRiskGrade(profile.RiskScore)
	if profile.RiskScore >= 5 {
		profile.ReviewRecommended = true
	}
}

func buildFlowEdgesFromWave5CInput(profiled string, input Wave5CInput) []FlowEdge {
	profiled = normalizeAddr(profiled)
	if profiled == "" || len(input.Flows) == 0 {
		return nil
	}

	edges := make([]FlowEdge, 0, len(input.Flows))
	for _, flow := range input.Flows {
		cp := normalizeAddr(flow.Counterparty)
		if cp == "" || cp == profiled {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(flow.Direction)) {
		case "inbound":
			edges = append(edges, FlowEdge{
				Source:      cp,
				Destination: profiled,
			})
		case "outbound":
			edges = append(edges, FlowEdge{
				Source:      profiled,
				Destination: cp,
			})
		case "authority":
			edges = append(edges, FlowEdge{
				Source:      profiled,
				Destination: cp,
			})
		}
	}

	return edges
}

func buildGraphMotifs(stats map[string]*actorAccumulator) []model.GraphMotif {
	motifs := make([]model.GraphMotif, 0)

	for _, acc := range stats {
		addressCount := len(acc.UniqueAddresses)
		total := acc.InboundCount + acc.OutboundCount

		if acc.Contextual && acc.InboundCount >= 5 && addressCount >= 2 {
			motifs = append(motifs, model.GraphMotif{
				Code:       "contextual_fan_in_hub",
				Summary:    fmt.Sprintf("Sampled graph shows contextual fan-in toward actor %s across %d addresses.", acc.Actor, addressCount),
				ToActor:    acc.Actor,
				ToCategory: acc.Category,
				Count:      acc.InboundCount,
				Confidence: roundFloat(acc.Confidence, 2),
			})
		}

		if acc.Contextual && acc.OutboundCount >= 5 && addressCount >= 2 {
			motifs = append(motifs, model.GraphMotif{
				Code:         "contextual_fan_out_hub",
				Summary:      fmt.Sprintf("Sampled graph shows contextual fan-out from actor %s across %d addresses.", acc.Actor, addressCount),
				FromActor:    acc.Actor,
				FromCategory: acc.Category,
				Count:        acc.OutboundCount,
				Confidence:   roundFloat(acc.Confidence, 2),
			})
		}

		if acc.Escalating && acc.OutboundCount >= 5 && addressCount >= 2 {
			motifs = append(motifs, model.GraphMotif{
				Code:         "risky_actor_fan_out",
				Summary:      fmt.Sprintf("Sampled graph shows risky fan-out from actor %s across %d addresses.", acc.Actor, addressCount),
				FromActor:    acc.Actor,
				FromCategory: acc.Category,
				Count:        total,
				Confidence:   roundFloat(acc.Confidence, 2),
			})
		}
	}

	inboundActors := make([]*actorAccumulator, 0)
	outboundActors := make([]*actorAccumulator, 0)
	for _, acc := range stats {
		if acc.InboundCount > 0 {
			inboundActors = append(inboundActors, acc)
		}
		if acc.OutboundCount > 0 {
			outboundActors = append(outboundActors, acc)
		}
	}

	for _, inAcc := range inboundActors {
		for _, outAcc := range outboundActors {
			if inAcc.Actor == outAcc.Actor && inAcc.Escalating && minInt(inAcc.InboundCount, outAcc.OutboundCount) >= 3 {
				motifs = append(motifs, model.GraphMotif{
					Code:       "risky_actor_u_turn",
					Summary:    fmt.Sprintf("Sampled flows show inbound and outbound interaction with risky actor %s (possible U-turn style pattern).", inAcc.Actor),
					FromActor:  inAcc.Actor,
					ToActor:    outAcc.Actor,
					Count:      minInt(inAcc.InboundCount, outAcc.OutboundCount),
					Confidence: roundFloat(math.Min(inAcc.Confidence, outAcc.Confidence), 2),
				})
				continue
			}

			if inAcc.Contextual && outAcc.Escalating && inAcc.Actor != outAcc.Actor && minInt(inAcc.InboundCount, outAcc.OutboundCount) >= 3 {
				motifs = append(motifs, model.GraphMotif{
					Code:         "contextual_to_risky_pass_through",
					Summary:      fmt.Sprintf("Sampled flows suggest contextual-to-risky pass-through from %s to %s.", inAcc.Actor, outAcc.Actor),
					FromActor:    inAcc.Actor,
					ToActor:      outAcc.Actor,
					FromCategory: inAcc.Category,
					ToCategory:   outAcc.Category,
					Count:        minInt(inAcc.InboundCount, outAcc.OutboundCount),
					Confidence:   roundFloat(math.Min(inAcc.Confidence, outAcc.Confidence), 2),
				})
			}
		}
	}

	sort.Slice(motifs, func(i, j int) bool {
		if motifs[i].Count == motifs[j].Count {
			return motifs[i].Code < motifs[j].Code
		}
		return motifs[i].Count > motifs[j].Count
	})

	if len(motifs) > 5 {
		motifs = motifs[:5]
	}

	return motifs
}

func countNearRiskyActors(
	profiled string,
	direct map[string]*model.ResolvedAttribution,
	outAdj map[string][]string,
	inAdj map[string][]string,
	resolve ResolveAttributionFunc,
) int {
	found := make(map[string]struct{})

	for counterparty := range direct {
		neighbors := append([]string{}, outAdj[counterparty]...)
		neighbors = append(neighbors, inAdj[counterparty]...)

		for _, next := range neighbors {
			if normalizeAddr(next) == profiled {
				continue
			}

			attr := resolve(next)
			if attr == nil || !attrEscalating(attr) {
				continue
			}

			key := resolvedActorKey(attr)
			if key != "" {
				found[key] = struct{}{}
			}
		}
	}

	return len(found)
}

func appendGraphReason(profile *model.WalletProfile, code, category, description string, offset float64, evidenceCount int) {
	if profile == nil {
		return
	}

	profile.RiskReasons = append(profile.RiskReasons, model.RiskReason{
		Code:          code,
		Category:      category,
		Description:   description,
		Offset:        offset,
		Source:        "graph_summary",
		EvidenceCount: evidenceCount,
	})

	switch category {
	case "FRAUD":
		profile.RiskBreakdown.Fraud = roundFloat(profile.RiskBreakdown.Fraud+offset, 2)
		profile.RiskScore = roundFloat(profile.RiskScore+(offset*0.5), 2)
	case "REPUTATION":
		profile.RiskBreakdown.Reputation = roundFloat(profile.RiskBreakdown.Reputation+offset, 2)
		profile.RiskScore = roundFloat(profile.RiskScore+(offset*0.3), 2)
	}
}

func graphAdjustedRiskGrade(score float64) string {
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

func extractCounterparty(profiled, src, dst string) (string, string, bool) {
	switch {
	case src == profiled && dst != profiled:
		return dst, "outbound", true
	case dst == profiled && src != profiled:
		return src, "inbound", true
	default:
		return "", "", false
	}
}

func resolvedActorKey(attr *model.ResolvedAttribution) string {
	if attr == nil {
		return ""
	}
	if strings.TrimSpace(attr.Actor) != "" {
		return strings.ToLower(strings.TrimSpace(attr.Actor))
	}
	if strings.TrimSpace(attr.Label) != "" {
		return strings.ToLower(strings.TrimSpace(attr.Label))
	}
	return ""
}

func actorDisplayName(attr *model.ResolvedAttribution) string {
	if attr == nil {
		return ""
	}
	if strings.TrimSpace(attr.Actor) != "" {
		return attr.Actor
	}
	return attr.Label
}

func attrEscalating(attr *model.ResolvedAttribution) bool {
	if attr == nil {
		return false
	}
	if attr.Escalating {
		return true
	}
	if attr.Contextual {
		return false
	}

	switch string(attr.RiskClass) {
	case "", "TRUSTED_SERVICE", "EXCHANGE", "MINING_POOL":
		return false
	default:
		return true
	}
}

func normalizeAddr(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func roundRatio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return roundFloat(float64(numerator)/float64(denominator), 4)
}

func roundFloat(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampMin(v, min float64) float64 {
	if v < min {
		return min
	}
	return v
}
