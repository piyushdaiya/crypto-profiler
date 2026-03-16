package analyzer

import (
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type RuleHit struct {
	Code           string
	Category       string
	Description    string
	Offset         float64
	Source         string
	RelatedEntity  string
	RelatedAddress string
	Severity       model.LabelSeverity
	Confidence     model.LabelConfidence
	EvidenceCount  int
}

type CombinationRule struct {
	Code        string
	RequiresAll []string
	Forbid      []string
	Category    string
	Description string
	Offset      float64
}

var combinationRules = []CombinationRule{
	{
		Code:        "combo_mixer_plus_fresh_wallet",
		RequiresAll: []string{"direct_mixer_interaction", "fresh_wallet"},
		Category:    "FRAUD",
		Description: "Fresh wallet directly interacting with mixer-related infrastructure",
		Offset:      20.0,
	},
	{
		Code:        "combo_mixer_plus_high_velocity",
		RequiresAll: []string{"direct_mixer_interaction", "high_velocity_behavior"},
		Category:    "FRAUD",
		Description: "Mixer exposure combined with high-velocity activity",
		Offset:      20.0,
	},
	{
		Code:        "combo_contextual_mitigation_established_wallet",
		RequiresAll: []string{"direct_mixer_interaction", "established_history"},
		Forbid:      []string{"fresh_wallet", "high_velocity_behavior", "rapid_passthrough_behavior"},
		Category:    "FRAUD",
		Description: "Contextual mitigation: established wallet with mixer exposure but no additional fraud signals",
		Offset:      -15.0,
	},
}

func applyCombinationRules(hits []RuleHit) []RuleHit {
	hitIndex := make(map[string]bool, len(hits))
	for _, hit := range hits {
		hitIndex[hit.Code] = true
	}

	var derived []RuleHit
	for _, rule := range combinationRules {
		if !hasAll(hitIndex, rule.RequiresAll) {
			continue
		}
		if hasAny(hitIndex, rule.Forbid) {
			continue
		}

		derived = append(derived, RuleHit{
			Code:        rule.Code,
			Category:    rule.Category,
			Description: rule.Description,
			Offset:      rule.Offset,
			Source:      "combination_rule",
		})
	}

	return derived
}

func hasAll(index map[string]bool, required []string) bool {
	for _, code := range required {
		if !index[code] {
			return false
		}
	}
	return true
}

func hasAny(index map[string]bool, forbidden []string) bool {
	for _, code := range forbidden {
		if index[code] {
			return true
		}
	}
	return false
}

func addHit(hits *[]RuleHit, category, code, desc string, offset float64, label *model.EntityLabel, evidenceCount int) {
	hit := RuleHit{
		Code:          code,
		Category:      category,
		Description:   desc,
		Offset:        offset,
		EvidenceCount: evidenceCount,
	}

	if label != nil {
		hit.Source = label.Source
		hit.RelatedEntity = label.Name
		hit.RelatedAddress = label.Address
		hit.Severity = label.Severity
		hit.Confidence = label.Confidence
	}

	*hits = append(*hits, hit)
}

func appendReason(reasons *[]model.RiskReason, hit RuleHit) {
	*reasons = append(*reasons, model.RiskReason{
		Code:           hit.Code,
		Category:       hit.Category,
		Description:    hit.Description,
		Offset:         hit.Offset,
		Source:         hit.Source,
		RelatedEntity:  hit.RelatedEntity,
		RelatedAddress: hit.RelatedAddress,
		Severity:       hit.Severity,
		Confidence:     hit.Confidence,
		EvidenceCount:  hit.EvidenceCount,
	})
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func determineGrade(score float64) string {
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

func shouldRecommendReview(score float64, hits []RuleHit) bool {
	if score >= 5 {
		return true
	}

	reviewTriggerCodes := map[string]bool{
		"direct_sanctions_exposure":                      true,
		"direct_mixer_interaction":                       true,
		"direct_high_risk_entity":                        true,
		"high_velocity_behavior":                         true,
		"fresh_wallet":                                   true,
		"combo_mixer_plus_fresh_wallet":                  true,
		"combo_mixer_plus_high_velocity":                 true,
		"rapid_passthrough_behavior":                     true,
		"hop_to_mixer_proximity":                         true,
		"watchlist_engine_unavailable":                   false,
		"combo_contextual_mitigation_established_wallet": false,
	}

	for _, hit := range hits {
		if reviewTriggerCodes[hit.Code] {
			return true
		}
	}

	return false
}
