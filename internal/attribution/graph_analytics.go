package attribution

import (
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
	Actor          string
	Category       string
	RiskClass      string
	Contextual     bool
	RiskEscalating bool
	Confidence     float64

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
	if len(edges) == 0 {
		return nil
	}

	return BuildGraphSummary(profiled, edges, resolve)
}

func BuildGraphSummary(
	profiled string,
	edges []FlowEdge,
	resolve ResolveAttributionFunc,
) *model.GraphSummary {
	if profiled == "" || len(edges) == 0 || resolve == nil {
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
				RiskEscalating:  attrRiskEscalating(attr),
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

		if attrRiskEscalating(attr) {
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
			RiskEscalating:   acc.RiskEscalating,
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
			// For now, treat authority-linked flow as profiled -> counterparty
			edges = append(edges, FlowEdge{
				Source:      profiled,
				Destination: cp,
			})
		}
	}

	return edges
}

func buildGraphMotifs(stats map[string]*actorAccumulator) []model.GraphMotif {
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

	motifs := make([]model.GraphMotif, 0)

	for _, inAcc := range inboundActors {
		for _, outAcc := range outboundActors {
			if inAcc.Actor == outAcc.Actor && inAcc.RiskEscalating {
				motifs = append(motifs, model.GraphMotif{
					Code:       "risky_actor_u_turn",
					Summary:    "Sampled flows show inbound and outbound interaction with the same risky actor.",
					FromActor:  inAcc.Actor,
					ToActor:    outAcc.Actor,
					Count:      minInt(inAcc.InboundCount, outAcc.OutboundCount),
					Confidence: roundFloat(math.Min(inAcc.Confidence, outAcc.Confidence), 2),
				})
				continue
			}

			if inAcc.Contextual && outAcc.RiskEscalating && inAcc.Actor != outAcc.Actor {
				motifs = append(motifs, model.GraphMotif{
					Code:         "contextual_to_risky_pass_through",
					Summary:      "Sampled flows suggest contextual-to-risky pass-through behavior.",
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

	if len(motifs) > 3 {
		motifs = motifs[:3]
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
			if attr == nil || !attrRiskEscalating(attr) {
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

func attrRiskEscalating(attr *model.ResolvedAttribution) bool {
	if attr == nil {
		return false
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
