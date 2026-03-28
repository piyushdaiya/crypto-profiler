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
	"math"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func ApplyTier1Attribution(profile *model.WalletProfile) {
	ApplyAttributionContext(profile)
}

func ApplyAttributionContext(profile *model.WalletProfile) {
	if profile == nil {
		return
	}

	if profile.Attribution == nil {
		if resolved, ok := ResolveAddress(profile.Address, profile.Network); ok {
			profile.Attribution = resolved
		}
	}
	if profile.Attribution == nil {
		return
	}
	if hasReasonCode(profile.RiskReasons, "direct_sanctions_match") {
		return
	}

	reason, breakdownCategory, ok := modifierForResolved(profile.Attribution)
	applyReasonModifier(profile, reason, breakdownCategory, ok)

	reason, breakdownCategory, ok = corroborationModifier(profile.Attribution)
	applyReasonModifier(profile, reason, breakdownCategory, ok)

	reason, breakdownCategory, ok = conflictModifier(profile.Attribution)
	applyReasonModifier(profile, reason, breakdownCategory, ok)

	if profile.RiskScore >= 5 || profile.Attribution.Escalating {
		profile.ReviewRecommended = true
	}

	detail := fmt.Sprintf(
		"Attribution: %s [%s/%s] via %s (confidence %.2f)",
		firstNonEmpty(profile.Attribution.Label, profile.Attribution.Actor),
		profile.Attribution.Category,
		profile.Attribution.RiskClass,
		profile.Attribution.SourceName,
		profile.Attribution.Confidence,
	)
	if strings.TrimSpace(profile.ValidationDetails) == "" {
		profile.ValidationDetails = detail
	} else if !strings.Contains(profile.ValidationDetails, detail) {
		profile.ValidationDetails += " | " + detail
	}
}

func modifierForResolved(resolved *model.ResolvedAttribution) (model.RiskReason, string, bool) {
	if resolved == nil {
		return model.RiskReason{}, "", false
	}

	name := firstNonEmpty(resolved.Label, resolved.Actor, resolved.Address)
	base := model.RiskReason{
		Source:         reasonSource(resolved),
		RelatedEntity:  firstNonEmpty(resolved.Actor, resolved.Label),
		RelatedAddress: resolved.Address,
		Severity:       severityForResolved(resolved),
		Confidence:     confidenceBucket(resolved.Confidence),
		EvidenceCount:  len(resolved.SupportingSources),
	}

	if resolved.SecondaryOnly {
		switch resolved.RiskClass {
		case model.AttributionRiskClassSanctioned, model.AttributionRiskClassIllicitService, model.AttributionRiskClassExploit, model.AttributionRiskClassScam:
			base.Code = "secondary_profile_risky_attribution"
			base.Category = "FRAUD"
			base.Description = fmt.Sprintf("Secondary corroborating source suggests risky attribution for profiled address: %s", name)
			base.Offset = 4.0
			return base, "FRAUD", true
		case model.AttributionRiskClassExchange, model.AttributionRiskClassTrustedService:
			base.Code = "secondary_profile_contextual_attribution"
			base.Category = "REPUTATION"
			base.Description = fmt.Sprintf("Secondary corroborating source suggests contextual service attribution: %s", name)
			base.Offset = -2.0
			return base, "REPUTATION", true
		case model.AttributionRiskClassMiningPool, model.AttributionRiskClassTreasury:
			base.Code = "secondary_profile_contextual_attribution"
			base.Category = "REPUTATION"
			base.Description = fmt.Sprintf("Secondary corroborating source suggests contextual infrastructure attribution: %s", name)
			base.Offset = -2.0
			return base, "REPUTATION", true
		default:
			return model.RiskReason{}, "", false
		}
	}

	switch resolved.RiskClass {
	case model.AttributionRiskClassSanctioned:
		base.Code = "tier1_profile_sanctioned_attribution"
		base.Category = "FRAUD"
		base.Description = fmt.Sprintf("Tier 1 attribution resolved profiled address to sanctioned actor: %s", name)
		base.Offset = 60.0
		return base, "FRAUD", true
	case model.AttributionRiskClassIllicitService, model.AttributionRiskClassExploit, model.AttributionRiskClassScam:
		base.Code = "tier1_profile_risky_attribution"
		base.Category = "FRAUD"
		base.Description = fmt.Sprintf("Tier 1 attribution resolved profiled address to risky actor: %s", name)
		base.Offset = 45.0
		return base, "FRAUD", true
	case model.AttributionRiskClassExchange, model.AttributionRiskClassTrustedService:
		base.Code = "tier1_profile_contextual_attribution"
		base.Category = "REPUTATION"
		base.Description = fmt.Sprintf("Tier 1 attribution resolved profiled address to contextual service: %s", name)
		base.Offset = -10.0
		return base, "REPUTATION", true
	case model.AttributionRiskClassMiningPool, model.AttributionRiskClassTreasury:
		base.Code = "tier1_profile_contextual_attribution"
		base.Category = "REPUTATION"
		base.Description = fmt.Sprintf("Tier 1 attribution resolved profiled address to contextual infrastructure: %s", name)
		base.Offset = -8.0
		return base, "REPUTATION", true
	default:
		return model.RiskReason{}, "", false
	}
}

func corroborationModifier(resolved *model.ResolvedAttribution) (model.RiskReason, string, bool) {
	if resolved == nil || resolved.SecondaryOnly {
		return model.RiskReason{}, "", false
	}

	secondarySupport := countSecondarySources(resolved.CorroboratingSources)
	if secondarySupport == 0 {
		return model.RiskReason{}, "", false
	}

	name := firstNonEmpty(resolved.Label, resolved.Actor, resolved.Address)
	base := model.RiskReason{
		Source:         "secondary_corroboration",
		RelatedEntity:  firstNonEmpty(resolved.Actor, resolved.Label),
		RelatedAddress: resolved.Address,
		Severity:       severityForResolved(resolved),
		Confidence:     confidenceBucket(resolved.Confidence),
		EvidenceCount:  secondarySupport,
	}

	if resolved.Escalating {
		base.Code = "secondary_corroborated_risky_attribution"
		base.Category = "FRAUD"
		base.Description = fmt.Sprintf("Secondary corroborating source supports risky attribution: %s", name)
		base.Offset = 3.0
		return base, "FRAUD", true
	}

	if resolved.Contextual {
		base.Code = "secondary_corroborated_contextual_attribution"
		base.Category = "REPUTATION"
		base.Description = fmt.Sprintf("Secondary corroborating source supports contextual attribution: %s", name)
		base.Offset = -2.0
		return base, "REPUTATION", true
	}

	return model.RiskReason{}, "", false
}

func conflictModifier(resolved *model.ResolvedAttribution) (model.RiskReason, string, bool) {
	if resolved == nil || len(resolved.ConflictingSources) == 0 {
		return model.RiskReason{}, "", false
	}

	return model.RiskReason{
		Code:           "attribution_source_conflict_observed",
		Source:         "attribution_conflict",
		Category:       "REPUTATION",
		Description:    "Conflicting attribution sources observed; stronger source precedence retained",
		Offset:         0.0,
		Severity:       model.LabelSeverityMedium,
		Confidence:     confidenceBucket(resolved.Confidence),
		EvidenceCount:  len(resolved.ConflictingSources),
		RelatedEntity:  firstNonEmpty(resolved.Actor, resolved.Label),
		RelatedAddress: resolved.Address,
	}, "REPUTATION", true
}

func applyReasonModifier(profile *model.WalletProfile, reason model.RiskReason, breakdownCategory string, ok bool) {
	if profile == nil || !ok || hasReasonCode(profile.RiskReasons, reason.Code) {
		return
	}

	profile.RiskReasons = append(profile.RiskReasons, reason)
	switch breakdownCategory {
	case "FRAUD":
		profile.RiskBreakdown.Fraud = clamp(profile.RiskBreakdown.Fraud + reason.Offset)
	case "REPUTATION":
		profile.RiskBreakdown.Reputation = clamp(profile.RiskBreakdown.Reputation + reason.Offset)
	case "LENDING":
		profile.RiskBreakdown.Lending = clamp(profile.RiskBreakdown.Lending + reason.Offset)
	}

	profile.RiskScore = combinedRisk(profile.RiskBreakdown)
	profile.RiskGrade = determineGrade(profile.RiskScore)
}

func countSecondarySources(sources []model.ResolvedAttributionSource) int {
	count := 0
	for _, source := range sources {
		if source.Tier == model.AttributionSourceTierSecondary {
			count++
		}
	}
	return count
}

func reasonSource(resolved *model.ResolvedAttribution) string {
	if resolved == nil {
		return "attribution"
	}
	if resolved.SecondaryOnly {
		return "secondary_attribution"
	}
	return "tier1_attribution"
}

func combinedRisk(breakdown model.RiskCategory) float64 {
	combined := (breakdown.Fraud * 0.5) + (breakdown.Reputation * 0.3) + (breakdown.Lending * 0.2)
	return math.Round(combined*100) / 100
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

func clamp(val float64) float64 {
	if val < 0 {
		return 0
	}
	if val > 100 {
		return 100
	}
	return math.Round(val*100) / 100
}

func hasReasonCode(reasons []model.RiskReason, code string) bool {
	for _, reason := range reasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}
