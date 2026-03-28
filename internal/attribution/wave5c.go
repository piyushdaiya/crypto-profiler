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

package attribution

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type InteractionCounterparty struct {
	Address       string
	Label         string
	Interactions  int
	InboundCount  int
	OutboundCount int
}

type FlowObservation struct {
	Timestamp         time.Time
	Direction         string
	Counterparty      string
	CounterpartyLabel string
}

type Wave5CInput struct {
	Network        string
	Counterparties []InteractionCounterparty
	Flows          []FlowObservation
}

type actorAggregation struct {
	Key           string
	Actor         string
	Category      model.LabelCategory
	RiskClass     model.AttributionRiskClass
	Contextual    bool
	Escalating    bool
	Confidence    float64
	Interactions  int
	InboundCount  int
	OutboundCount int
	Addresses     map[string]struct{}
	Resolved      *model.ResolvedAttribution
}

type flowResolution struct {
	Flow      FlowObservation
	Resolved  *model.ResolvedAttribution
	ActorKey  string
	ActorName string
}

func ApplyWave5CContext(profile *model.WalletProfile, input Wave5CInput) {
	if profile == nil {
		return
	}

	network := strings.ToUpper(strings.TrimSpace(firstNonEmpty(input.Network, profile.Network)))
	if network == "" {
		return
	}

	aggregations, totalInteractions := resolveActorAggregations(profile, network, input.Counterparties)
	applyActorAwareScoring(profile, aggregations, totalInteractions)
	applyFlowAwareInsights(profile, network, input.Flows)
}

func resolveActorAggregations(profile *model.WalletProfile, network string, counterparties []InteractionCounterparty) ([]*actorAggregation, int) {
	totalInteractions := 0
	actors := map[string]*actorAggregation{}

	for _, cp := range counterparties {
		interactions := cp.Interactions
		if interactions <= 0 {
			interactions = cp.InboundCount + cp.OutboundCount
		}
		if interactions <= 0 {
			continue
		}
		totalInteractions += interactions

		resolved, ok := ResolveAddress(cp.Address, network)
		if !ok || !supportsActorInsight(resolved) {
			continue
		}

		key := actorKey(resolved)
		if key == "" {
			continue
		}

		agg := actors[key]
		if agg == nil {
			agg = &actorAggregation{
				Key:        key,
				Actor:      firstNonEmpty(resolved.Actor, resolved.Label),
				Category:   resolved.Category,
				RiskClass:  resolved.RiskClass,
				Contextual: resolved.Contextual,
				Escalating: resolved.Escalating,
				Addresses:  map[string]struct{}{},
				Resolved:   resolved,
			}
			actors[key] = agg
		}

		agg.Interactions += interactions
		agg.InboundCount += cp.InboundCount
		agg.OutboundCount += cp.OutboundCount
		agg.Addresses[normalizeAddress(cp.Address)] = struct{}{}
		if resolved.Confidence > agg.Confidence {
			agg.Confidence = resolved.Confidence
			agg.Resolved = resolved
			agg.Category = resolved.Category
			agg.RiskClass = resolved.RiskClass
			agg.Contextual = resolved.Contextual
			agg.Escalating = resolved.Escalating
		}
	}

	out := make([]*actorAggregation, 0, len(actors))
	for _, agg := range actors {
		if profile.Attribution != nil && sameActor(profile.Attribution, agg.Resolved) {
			agg.Addresses[normalizeAddress(profile.Address)] = struct{}{}
		}
		out = append(out, agg)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Escalating != out[j].Escalating {
			return out[i].Escalating
		}
		if out[i].Interactions != out[j].Interactions {
			return out[i].Interactions > out[j].Interactions
		}
		return out[i].Actor < out[j].Actor
	})

	return out, totalInteractions
}

func applyActorAwareScoring(profile *model.WalletProfile, aggregations []*actorAggregation, totalInteractions int) {
	if profile == nil || len(aggregations) == 0 || totalInteractions <= 0 {
		return
	}

	var strongestRisky *actorAggregation
	var strongestContextual *actorAggregation

	for _, agg := range aggregations {
		if agg == nil || agg.Resolved == nil {
			continue
		}

		share := float64(agg.Interactions) / float64(totalInteractions)
		addressCount := len(agg.Addresses)
		actorName := firstNonEmpty(agg.Actor, agg.Resolved.Label, agg.Resolved.Address)

		if addressCount > 1 {
			appendInsight(profile, model.AttributionInsight{
				Code:          "cluster_grouping",
				Type:          model.AttributionInsightClusterGrouping,
				Summary:       fmt.Sprintf("Cluster-aware grouping links %d attributed addresses to actor %s in the sampled Layer 1 activity.", addressCount, actorName),
				Actor:         actorName,
				Category:      agg.Category,
				Confidence:    agg.Confidence,
				EvidenceCount: addressCount,
			})
		}

		exposureType := model.AttributionInsightDirectExposure
		exposureSummary := fmt.Sprintf(
			"Direct exposure to attributed %s actor %s across %d interactions.",
			exposureDescriptor(agg),
			actorName,
			agg.Interactions,
		)
		appendInsight(profile, model.AttributionInsight{
			Code:          "direct_actor_exposure",
			Type:          exposureType,
			Summary:       exposureSummary,
			Actor:         actorName,
			Category:      agg.Category,
			HopDepth:      1,
			Confidence:    agg.Confidence,
			EvidenceCount: agg.Interactions,
		})

		if agg.Escalating && supportsActorScoreRefinement(agg.Resolved) {
			if strongestRisky == nil {
				strongestRisky = agg
			}
			if agg.Interactions >= 8 {
				offset := 4.0
				if addressCount > 1 {
					offset = 6.0
				}
				appendInsight(profile, model.AttributionInsight{
					Code:          "actor_repeated_risky_interaction",
					Type:          model.AttributionInsightActorRepeated,
					Summary:       fmt.Sprintf("Actor-aware repeated interaction links %d touches to risky actor %s across %d address(es).", agg.Interactions, actorName, addressCount),
					Actor:         actorName,
					Category:      agg.Category,
					Confidence:    agg.Confidence,
					EvidenceCount: agg.Interactions,
				})
				applyReasonModifier(profile, model.RiskReason{
					Code:           "actor_repeated_risky_interaction",
					Source:         "wave5c_actor_refinement",
					Category:       "FRAUD",
					Description:    fmt.Sprintf("Repeated interaction observed with attributed risky actor %s across %d address(es) (%d interactions)", actorName, addressCount, agg.Interactions),
					Offset:         offset,
					RelatedEntity:  actorName,
					Severity:       severityForResolved(agg.Resolved),
					Confidence:     confidenceBucket(agg.Confidence),
					EvidenceCount:  agg.Interactions,
					RelatedAddress: agg.Resolved.Address,
				}, "FRAUD", true)
			}
			if share >= 0.20 || (addressCount > 1 && share >= 0.12) {
				appendInsight(profile, model.AttributionInsight{
					Code:          "actor_risky_concentration",
					Type:          model.AttributionInsightActorConcentration,
					Summary:       fmt.Sprintf("Actor-level concentration is centered on risky actor %s (%.0f%% of attributed interactions).", actorName, share*100),
					Actor:         actorName,
					Category:      agg.Category,
					Confidence:    agg.Confidence,
					EvidenceCount: agg.Interactions,
				})
				applyReasonModifier(profile, model.RiskReason{
					Code:           "actor_risky_concentration",
					Source:         "wave5c_actor_refinement",
					Category:       "FRAUD",
					Description:    fmt.Sprintf("Layer 1 activity shows actor-level concentration toward risky actor %s (%.0f%% of attributed interactions)", actorName, share*100),
					Offset:         5.0,
					RelatedEntity:  actorName,
					Severity:       severityForResolved(agg.Resolved),
					Confidence:     confidenceBucket(agg.Confidence),
					EvidenceCount:  agg.Interactions,
					RelatedAddress: agg.Resolved.Address,
				}, "FRAUD", true)
			}
		}

		if agg.Contextual {
			if strongestContextual == nil {
				strongestContextual = agg
			}
			if supportsActorScoreRefinement(agg.Resolved) && agg.Interactions >= 25 {
				appendInsight(profile, model.AttributionInsight{
					Code:          "actor_contextual_repeated_interaction",
					Type:          model.AttributionInsightActorRepeated,
					Summary:       fmt.Sprintf("Repeated interaction remains inside contextual actor %s (%d interactions).", actorName, agg.Interactions),
					Actor:         actorName,
					Category:      agg.Category,
					Confidence:    agg.Confidence,
					EvidenceCount: agg.Interactions,
				})
				applyReasonModifier(profile, model.RiskReason{
					Code:           "actor_contextual_repeated_interaction",
					Source:         "wave5c_actor_refinement",
					Category:       "REPUTATION",
					Description:    fmt.Sprintf("Repeated interaction remains concentrated on contextual actor %s (%d interactions)", actorName, agg.Interactions),
					Offset:         -1.5,
					RelatedEntity:  actorName,
					Severity:       severityForResolved(agg.Resolved),
					Confidence:     confidenceBucket(agg.Confidence),
					EvidenceCount:  agg.Interactions,
					RelatedAddress: agg.Resolved.Address,
				}, "REPUTATION", true)
			}
			if supportsActorScoreRefinement(agg.Resolved) && (share >= 0.10 || (addressCount > 1 && share >= 0.07)) {
				appendInsight(profile, model.AttributionInsight{
					Code:          "actor_contextual_concentration",
					Type:          model.AttributionInsightActorConcentration,
					Summary:       fmt.Sprintf("Actor-level concentration points to contextual actor %s (%.0f%% of attributed interactions).", actorName, share*100),
					Actor:         actorName,
					Category:      agg.Category,
					Confidence:    agg.Confidence,
					EvidenceCount: agg.Interactions,
				})
				applyReasonModifier(profile, model.RiskReason{
					Code:           "actor_contextual_concentration",
					Source:         "wave5c_actor_refinement",
					Category:       "REPUTATION",
					Description:    fmt.Sprintf("Actor-level concentration points to contextual service actor %s (%.0f%% of attributed interactions)", actorName, share*100),
					Offset:         -3.0,
					RelatedEntity:  actorName,
					Severity:       severityForResolved(agg.Resolved),
					Confidence:     confidenceBucket(agg.Confidence),
					EvidenceCount:  agg.Interactions,
					RelatedAddress: agg.Resolved.Address,
				}, "REPUTATION", true)
			}
		}
	}

	if strongestRisky != nil && strongestRisky.Interactions > 0 {
		profile.ReviewRecommended = true
	}
	if strongestContextual != nil && strongestContextual.Interactions > 0 && profile.RiskScore < 5 {
		profile.ReviewRecommended = profile.ReviewRecommended || profile.Attribution != nil && profile.Attribution.Escalating
	}
}

func applyFlowAwareInsights(profile *model.WalletProfile, network string, flows []FlowObservation) {
	if profile == nil || len(flows) < 2 {
		return
	}

	resolvedFlows := make([]flowResolution, 0, len(flows))
	for _, flow := range flows {
		if flow.Timestamp.IsZero() {
			continue
		}
		direction := normalizeFlowDirection(flow.Direction)
		if direction == "" {
			continue
		}
		resolved, ok := ResolveAddress(flow.Counterparty, network)
		if !ok || !supportsActorInsight(resolved) {
			continue
		}

		resolvedFlows = append(resolvedFlows, flowResolution{
			Flow:      flow,
			Resolved:  resolved,
			ActorKey:  actorKey(resolved),
			ActorName: firstNonEmpty(resolved.Actor, resolved.Label),
		})
	}

	if len(resolvedFlows) < 2 {
		return
	}

	sort.SliceStable(resolvedFlows, func(i, j int) bool {
		return resolvedFlows[i].Flow.Timestamp.Before(resolvedFlows[j].Flow.Timestamp)
	})

	for i := 0; i < len(resolvedFlows)-1; i++ {
		left := resolvedFlows[i]
		right := resolvedFlows[i+1]
		if right.Flow.Timestamp.Sub(left.Flow.Timestamp) > 2*time.Hour {
			continue
		}

		if left.ActorKey != "" && left.ActorKey == right.ActorKey && isInboundLike(left.Flow.Direction) && isOutboundLike(right.Flow.Direction) {
			appendInsight(profile, model.AttributionInsight{
				Code:          "u_turn_actor_pattern",
				Type:          model.AttributionInsightUTurn,
				Summary:       fmt.Sprintf("Sampled Layer 1 activity shows a U-turn style pattern through actor %s within the observed window.", left.ActorName),
				Actor:         left.ActorName,
				Category:      left.Resolved.Category,
				HopDepth:      1,
				Confidence:    minFloat(left.Resolved.Confidence, right.Resolved.Confidence),
				EvidenceCount: 2,
			})
			if left.Resolved.Escalating && supportsActorScoreRefinement(left.Resolved) {
				applyReasonModifier(profile, model.RiskReason{
					Code:           "actor_u_turn_risky_service",
					Source:         "wave5c_flow_refinement",
					Category:       "FRAUD",
					Description:    fmt.Sprintf("Sampled Layer 1 activity suggests U-turn routing through risky actor %s", left.ActorName),
					Offset:         3.0,
					RelatedEntity:  left.ActorName,
					Severity:       severityForResolved(left.Resolved),
					Confidence:     confidenceBucket(minFloat(left.Resolved.Confidence, right.Resolved.Confidence)),
					EvidenceCount:  2,
					RelatedAddress: left.Resolved.Address,
				}, "FRAUD", true)
			}
			continue
		}

		if !isInboundLike(left.Flow.Direction) || !isOutboundLike(right.Flow.Direction) {
			continue
		}
		if left.ActorKey == "" || right.ActorKey == "" || left.ActorKey == right.ActorKey {
			continue
		}

		inboundActor := left.ActorName
		outboundActor := right.ActorName
		appendInsight(profile, model.AttributionInsight{
			Code:          "pass_through_actor_pattern",
			Type:          model.AttributionInsightPassThrough,
			Summary:       fmt.Sprintf("Sampled Layer 1 activity shows rapid pass-through from %s to %s.", inboundActor, outboundActor),
			Actor:         outboundActor,
			Category:      right.Resolved.Category,
			HopDepth:      1,
			Confidence:    minFloat(left.Resolved.Confidence, right.Resolved.Confidence),
			EvidenceCount: 2,
		})

		if right.Resolved.Escalating && supportsActorScoreRefinement(right.Resolved) {
			appendInsight(profile, model.AttributionInsight{
				Code:          "near_risky_actor_exposure",
				Type:          model.AttributionInsightNearExposure,
				Summary:       fmt.Sprintf("Near exposure to risky actor %s is visible through an intermediary pass-through path.", outboundActor),
				Actor:         outboundActor,
				Category:      right.Resolved.Category,
				HopDepth:      2,
				Confidence:    right.Resolved.Confidence,
				EvidenceCount: 2,
			})
			applyReasonModifier(profile, model.RiskReason{
				Code:           "actor_pass_through_risky_exposure",
				Source:         "wave5c_flow_refinement",
				Category:       "FRAUD",
				Description:    fmt.Sprintf("Pass-through style routing links sampled activity to risky actor %s", outboundActor),
				Offset:         4.0,
				RelatedEntity:  outboundActor,
				Severity:       severityForResolved(right.Resolved),
				Confidence:     confidenceBucket(right.Resolved.Confidence),
				EvidenceCount:  2,
				RelatedAddress: right.Resolved.Address,
			}, "FRAUD", true)
		}
	}
}

func appendInsight(profile *model.WalletProfile, insight model.AttributionInsight) {
	if profile == nil || strings.TrimSpace(insight.Code) == "" || strings.TrimSpace(insight.Summary) == "" {
		return
	}
	for _, existing := range profile.AttributionInsights {
		if existing.Code == insight.Code && existing.Actor == insight.Actor {
			return
		}
	}
	profile.AttributionInsights = append(profile.AttributionInsights, insight)
}

func actorKey(resolved *model.ResolvedAttribution) string {
	if resolved == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(firstNonEmpty(resolved.Actor, resolved.Label)))
}

func sameActor(left *model.ResolvedAttribution, right *model.ResolvedAttribution) bool {
	return actorKey(left) != "" && actorKey(left) == actorKey(right)
}

func supportsActorInsight(resolved *model.ResolvedAttribution) bool {
	if resolved == nil {
		return false
	}
	if resolved.SecondaryOnly {
		return resolved.Confidence >= 0.50
	}
	return resolved.Confidence >= 0.60
}

func supportsActorScoreRefinement(resolved *model.ResolvedAttribution) bool {
	if resolved == nil || resolved.SecondaryOnly {
		return false
	}
	if resolved.SourceTier == model.AttributionSourceTierLocalOverride {
		return true
	}
	return resolved.Confidence >= 0.80
}

func exposureDescriptor(agg *actorAggregation) string {
	if agg == nil {
		return "attributed"
	}
	if agg.Escalating {
		return "risky"
	}
	if agg.Contextual {
		return "contextual"
	}
	return "attributed"
}

func normalizeFlowDirection(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "inbound", "destination", "receive", "received":
		return "inbound"
	case "outbound", "source", "send", "sent":
		return "outbound"
	case "authority":
		return "authority"
	default:
		return ""
	}
}

func isInboundLike(direction string) bool {
	return normalizeFlowDirection(direction) == "inbound"
}

func isOutboundLike(direction string) bool {
	return normalizeFlowDirection(direction) == "outbound"
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}
