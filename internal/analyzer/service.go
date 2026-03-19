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

package analyzer

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
	"github.com/piyushdaiya/crypto-profiler/internal/watchlist"
)

func Investigate(profile *model.WalletProfile, txs []model.Transaction) {
	var fraudScore, repScore, lendScore float64
	reasons := make([]model.RiskReason, 0)
	hits := make([]RuleHit, 0)

	// ---------------------------------------------------------
	// 1. WATCHLIST / SANCTIONS CHECK
	// ---------------------------------------------------------
	engineResp, err := watchlist.CheckWatchlist(profile.Address)
	if err != nil {
		// Fail open: record warning, continue analysis.
		hits = append(hits, RuleHit{
			Code:          "watchlist_engine_unavailable",
			Category:      "SYSTEM",
			Description:   "Watchlist Engine unavailable - sanctions check skipped",
			Offset:        0.0,
			Source:        "watchlist_engine",
			EvidenceCount: 1,
		})
		profile.ValidationDetails += " | [Warning: Watchlist Engine Offline]"
	} else if engineResp.Sanctioned {
		// Critical short-circuit for sanctioned addresses.
		reasons = append(reasons, model.RiskReason{
			Code:           "direct_sanctions_match",
			Category:       "FRAUD",
			Description:    fmt.Sprintf("CRITICAL: %s sanctioned address (%s)", engineResp.Source, engineResp.Currency),
			Offset:         100.0,
			Source:         "watchlist_engine",
			RelatedEntity:  engineResp.Source,
			RelatedAddress: profile.Address,
			Severity:       model.LabelSeverityCritical,
			Confidence:     model.LabelConfidenceHigh,
			EvidenceCount:  1,
		})

		profile.RiskScore = 100.0
		profile.RiskGrade = "CRITICAL (Sanctioned)"
		profile.ReviewRecommended = true
		profile.RiskBreakdown = model.RiskCategory{
			Fraud:      100.0,
			Reputation: 100.0,
			Lending:    100.0,
		}
		profile.RiskReasons = reasons
		return
	}

	// ---------------------------------------------------------
	// 1B. DIRECT LABEL ON PROFILED ADDRESS
	// ---------------------------------------------------------
	if label, ok := LookupEntityLabel(profile.Address); ok {
		switch label.Category {
		case model.LabelCategoryMixer:
			addHit(
				&hits,
				"FRAUD",
				"profiled_address_high_risk_label",
				fmt.Sprintf("Profiled address labeled as high-risk entity: %s", label.Name),
				45.0,
				&label,
				1,
			)

		case model.LabelCategoryExploit, model.LabelCategoryScam:
			addHit(
				&hits,
				"FRAUD",
				"profiled_address_high_risk_label",
				fmt.Sprintf("Profiled address labeled as high-risk entity: %s", label.Name),
				45.0,
				&label,
				1,
			)

		case model.LabelCategoryExchange:
			addHit(
				&hits,
				"REPUTATION",
				"profiled_address_trusted_label",
				fmt.Sprintf("Profiled address labeled as known exchange: %s", label.Name),
				-10.0,
				&label,
				1,
			)

		case model.LabelCategoryTrusted, model.LabelCategoryProtocol:
			addHit(
				&hits,
				"REPUTATION",
				"profiled_address_trusted_label",
				fmt.Sprintf("Profiled address labeled as trusted protocol: %s", label.Name),
				-10.0,
				&label,
				1,
			)
		}
	}

	// ---------------------------------------------------------
	// 2. AGE / HISTORY SIGNALS
	// ---------------------------------------------------------
	if profile.FirstSeen != nil {
		hoursOld := time.Since(*profile.FirstSeen).Hours()

		switch {
		case hoursOld > 24*365:
			addHit(&hits, "REPUTATION", "established_history", "Established History (>1 Year)", -10.0, nil, 1)

		case hoursOld < 24:
			addHit(&hits, "FRAUD", "fresh_wallet", "Freshly Created Wallet (<24h)", 35.0, nil, 1)
		}
	}

	// ---------------------------------------------------------
	// 3. DIRECT ENTITY INTERACTION SIGNALS
	// ---------------------------------------------------------
	directMixerSeen := false
	directHighRiskSeen := false
	directTrustedSeen := false

	for _, tx := range txs {
		var otherParty string
		if strings.EqualFold(tx.From, profile.Address) {
			otherParty = strings.ToLower(strings.TrimSpace(tx.To))
		} else {
			otherParty = strings.ToLower(strings.TrimSpace(tx.From))
		}

		if otherParty == "" || strings.EqualFold(otherParty, profile.Address) {
			continue
		}

		label, ok := LookupEntityLabel(otherParty)
		if !ok {
			continue
		}

		switch label.Category {
		case model.LabelCategorySanctions:
			addHit(
				&hits,
				"FRAUD",
				"direct_sanctions_exposure",
				"Direct interaction with sanctioned entity",
				100.0,
				&label,
				1,
			)

		case model.LabelCategoryMixer:
			if !directMixerSeen {
				addHit(
					&hits,
					"FRAUD",
					"direct_mixer_interaction",
					fmt.Sprintf("Direct interaction with %s", label.Name),
					20.0,
					&label,
					1,
				)
				directMixerSeen = true
			}

		case model.LabelCategoryExploit, model.LabelCategoryScam:
			if !directHighRiskSeen {
				addHit(
					&hits,
					"FRAUD",
					"direct_high_risk_entity",
					fmt.Sprintf("Direct interaction with high-risk entity: %s", label.Name),
					45.0,
					&label,
					1,
				)
				directHighRiskSeen = true
			}

		case model.LabelCategoryExchange:
			addHit(
				&hits,
				"REPUTATION",
				"exchange_interaction",
				fmt.Sprintf("Interaction with known exchange: %s", label.Name),
				-5.0,
				&label,
				1,
			)

		case model.LabelCategoryTrusted, model.LabelCategoryProtocol:
			if !directTrustedSeen {
				addHit(
					&hits,
					"REPUTATION",
					"trusted_or_protocol_interaction",
					fmt.Sprintf("Interaction with known protocol/trusted entity: %s", label.Name),
					-5.0,
					&label,
					1,
				)
				directTrustedSeen = true
			}
		}
	}

	// ---------------------------------------------------------
	// 4. VELOCITY / ACTIVITY SIGNALS
	// ---------------------------------------------------------
	if profile.TxCount > 0 && profile.FirstSeen != nil {
		hoursActive := time.Since(*profile.FirstSeen).Hours()
		if hoursActive < 1 {
			hoursActive = 1
		}

		txPerHour := float64(profile.TxCount) / hoursActive
		if txPerHour > 20.0 {
			addHit(&hits, "FRAUD", "high_velocity_behavior", "High Velocity Behavior (Potential Bot)", 25.0, nil, 1)
		}
	}

	// ---------------------------------------------------------
	// 4B. REPEATED INTERACTION WITH FLAGGED COUNTERPARTIES
	// ---------------------------------------------------------
	applyRepeatedFlaggedCounterpartyHeuristic(profile, txs, &hits)

	// ---------------------------------------------------------
	// 4C. Concentration Heuristic
	// ---------------------------------------------------------
	applyServiceConcentrationHeuristic(profile, txs, &hits)

	// ---------------------------------------------------------
	// 5. NOISY INBOUND / DUSTING-LIKE OBSERVATION
	// ---------------------------------------------------------
	// This is intentionally low-severity and should not automatically
	// recommend review. It is meant to surface "observed" behavior
	// such as spammy inbound transfers, many counterparties, and
	// dusting-like patterns on public wallets.
	if _, hasProfileLabel := LookupEntityLabel(profile.Address); !hasProfileLabel {
		applyNoisyInboundHeuristics(profile, txs, &hits)
	}

	// ---------------------------------------------------------
	// 6. PLACEHOLDER FOR FUTURE PATTERNS
	// ---------------------------------------------------------
	// Future examples:
	// - rapid_passthrough_behavior
	// - peeling_chain_behavior
	// - hop_to_mixer_proximity
	// - dormant_wallet_reactivation
	// - suspicious_inbound_outbound_ratio

	// ---------------------------------------------------------
	// 7. APPLY COMBINATION RULES
	// ---------------------------------------------------------
	hits = append(hits, applyCombinationRules(hits)...)

	// ---------------------------------------------------------
	// 8. CONVERT HITS TO SCORES + REASONS
	// ---------------------------------------------------------
	for _, hit := range hits {
		switch hit.Category {
		case "FRAUD":
			fraudScore += hit.Offset
		case "REPUTATION":
			repScore += hit.Offset
		case "LENDING":
			lendScore += hit.Offset
		}

		appendReason(&reasons, hit)
	}

	// ---------------------------------------------------------
	// 9. FINALIZE SCORE
	// ---------------------------------------------------------
	fraudScore = clamp(fraudScore, 0, 100)
	repScore = clamp(repScore, 0, 100)
	lendScore = clamp(lendScore, 0, 100)

	combinedRisk := (fraudScore * 0.5) + (repScore * 0.3) + (lendScore * 0.2)
	combinedRisk = math.Round(combinedRisk*100) / 100

	profile.RiskScore = combinedRisk
	profile.RiskGrade = determineGrade(combinedRisk)
	profile.ReviewRecommended = shouldRecommendReview(combinedRisk, hits)
	profile.RiskBreakdown = model.RiskCategory{
		Fraud:      math.Round(fraudScore*100) / 100,
		Reputation: math.Round(repScore*100) / 100,
		Lending:    math.Round(lendScore*100) / 100,
	}
	profile.RiskReasons = reasons
}

func applyRepeatedFlaggedCounterpartyHeuristic(profile *model.WalletProfile, txs []model.Transaction, hits *[]RuleHit) {
	if profile == nil || len(txs) == 0 {
		return
	}

	address := strings.ToLower(strings.TrimSpace(profile.Address))
	if address == "" {
		return
	}

	type interactionSummary struct {
		Label        model.EntityLabel
		Count        int
		UniqueTxHash map[string]struct{}
	}

	seen := map[string]*interactionSummary{}

	for _, tx := range txs {
		from := strings.ToLower(strings.TrimSpace(tx.From))
		to := strings.ToLower(strings.TrimSpace(tx.To))

		var counterparty string
		switch {
		case from == address && to != "" && to != address:
			counterparty = to
		case to == address && from != "" && from != address:
			counterparty = from
		default:
			continue
		}

		label, ok := LookupEntityLabel(counterparty)
		if !ok || !isFlaggedCounterpartyCategory(label.Category) {
			continue
		}

		entry, exists := seen[counterparty]
		if !exists {
			entry = &interactionSummary{
				Label:        label,
				UniqueTxHash: map[string]struct{}{},
			}
			seen[counterparty] = entry
		}

		txHash := strings.TrimSpace(tx.Hash)
		if txHash == "" {
			txHash = fmt.Sprintf("%s|%s|%d", tx.From, tx.To, tx.TimeStamp)
		}
		if _, already := entry.UniqueTxHash[txHash]; already {
			continue
		}

		entry.UniqueTxHash[txHash] = struct{}{}
		entry.Count++
	}

	for _, entry := range seen {
		if entry.Count < 3 {
			continue
		}

		offset := repeatedFlaggedInteractionOffset(entry.Label.Category, entry.Count)

		addHit(
			hits,
			"FRAUD",
			"repeated_flagged_counterparty_interaction",
			fmt.Sprintf("Repeated interaction with flagged counterparty: %s (%d interactions)", entry.Label.Name, entry.Count),
			offset,
			&entry.Label,
			entry.Count,
		)
	}
}

func applyNoisyInboundHeuristics(profile *model.WalletProfile, txs []model.Transaction, hits *[]RuleHit) {
	if profile == nil || len(txs) == 0 {
		return
	}

	address := strings.ToLower(strings.TrimSpace(profile.Address))
	if address == "" {
		return
	}

	inboundCount := 0
	outboundCount := 0
	zeroValueInboundCount := 0
	uniqueInboundSenders := map[string]struct{}{}

	for _, tx := range txs {
		from := strings.ToLower(strings.TrimSpace(tx.From))
		to := strings.ToLower(strings.TrimSpace(tx.To))

		switch {
		case to == address && from != "" && from != address:
			inboundCount++
			uniqueInboundSenders[from] = struct{}{}
			if isZeroLikeValue(tx.Value) {
				zeroValueInboundCount++
			}

		case from == address && to != "" && to != address:
			outboundCount++
		}
	}

	totalDirectional := inboundCount + outboundCount
	if totalDirectional == 0 {
		return
	}

	inboundRatio := float64(inboundCount) / float64(totalDirectional)
	uniqueFanIn := len(uniqueInboundSenders)

	// Low-severity "observed" signal:
	// mostly inbound activity with many unique counterparties.
	if inboundCount >= 20 && inboundRatio >= 0.90 && uniqueFanIn >= 15 {
		addHit(
			hits,
			"FRAUD",
			"noisy_inbound_activity",
			"High-volume mostly-inbound activity with many unique senders",
			2.0,
			nil,
			uniqueFanIn,
		)
	}

	// Additional mild signal for very high fan-in.
	if uniqueFanIn >= 25 {
		addHit(
			hits,
			"FRAUD",
			"high_counterparty_fan_in",
			"High counterparty fan-in observed across sampled transfers",
			2.0,
			nil,
			uniqueFanIn,
		)
	}

	// Dusting-like observation: many inbound zero-value transfers.
	if zeroValueInboundCount >= 10 && inboundRatio >= 0.80 {
		addHit(
			hits,
			"FRAUD",
			"zero_value_inbound_pattern",
			"Frequent zero-value inbound transfers suggest dusting/spam-like activity",
			1.0,
			nil,
			zeroValueInboundCount,
		)
	}
}

func isFlaggedCounterpartyCategory(category model.LabelCategory) bool {
	switch category {
	case model.LabelCategoryMixer,
		model.LabelCategoryExploit,
		model.LabelCategoryScam,
		model.LabelCategorySanctions:
		return true
	default:
		return false
	}
}

func repeatedFlaggedInteractionOffset(category model.LabelCategory, count int) float64 {
	base := 8.0

	switch category {
	case model.LabelCategorySanctions:
		base = 15.0
	case model.LabelCategoryMixer:
		base = 10.0
	case model.LabelCategoryExploit, model.LabelCategoryScam:
		base = 12.0
	}

	switch {
	case count >= 10:
		return base + 8.0
	case count >= 5:
		return base + 4.0
	default:
		return base
	}
}

func isZeroLikeValue(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}

	allZero := true
	for _, ch := range s {
		if ch == '.' {
			continue
		}
		if ch != '0' {
			allZero = false
			break
		}
	}

	return allZero
}
func applyServiceConcentrationHeuristic(profile *model.WalletProfile, txs []model.Transaction, hits *[]RuleHit) {
	if profile == nil || len(txs) == 0 {
		return
	}

	address := strings.ToLower(strings.TrimSpace(profile.Address))
	if address == "" {
		return
	}

	type serviceInteractionSummary struct {
		Label model.EntityLabel
		Count int
	}

	totalDirectional := 0
	seen := map[string]*serviceInteractionSummary{}

	for _, tx := range txs {
		from := strings.ToLower(strings.TrimSpace(tx.From))
		to := strings.ToLower(strings.TrimSpace(tx.To))

		var counterparty string
		switch {
		case from == address && to != "" && to != address:
			counterparty = to
		case to == address && from != "" && from != address:
			counterparty = from
		default:
			continue
		}

		totalDirectional++

		label, ok := LookupEntityLabel(counterparty)
		if !ok || !isConcentrationServiceCategory(label.Category) {
			continue
		}

		entry, exists := seen[counterparty]
		if !exists {
			entry = &serviceInteractionSummary{Label: label}
			seen[counterparty] = entry
		}
		entry.Count++
	}

	if totalDirectional == 0 || len(seen) == 0 {
		return
	}

	var top *serviceInteractionSummary
	for _, entry := range seen {
		if top == nil || entry.Count > top.Count {
			top = entry
		}
	}
	if top == nil || top.Count < 4 {
		return
	}

	ratio := float64(top.Count) / float64(totalDirectional)
	percent := ratio * 100.0

	switch top.Label.Category {
	case model.LabelCategoryMixer, model.LabelCategoryExploit, model.LabelCategoryScam, model.LabelCategorySanctions:
		if ratio >= 0.40 {
			addHit(
				hits,
				"FRAUD",
				"high_risk_service_concentration",
				fmt.Sprintf("High interaction concentration to high-risk service: %s (%.1f%% of observed activity)", top.Label.Name, percent),
				18.0,
				&top.Label,
				top.Count,
			)
		}

	case model.LabelCategoryExchange:
		if ratio >= 0.60 {
			addHit(
				hits,
				"REPUTATION",
				"exchange_concentration",
				fmt.Sprintf("High interaction concentration to known exchange: %s (%.1f%% of observed activity)", top.Label.Name, percent),
				-4.0,
				&top.Label,
				top.Count,
			)
		}

	case model.LabelCategoryTrusted, model.LabelCategoryProtocol:
		if ratio >= 0.60 {
			addHit(
				hits,
				"REPUTATION",
				"single_service_concentration",
				fmt.Sprintf("High interaction concentration to known protocol/trusted service: %s (%.1f%% of observed activity)", top.Label.Name, percent),
				-4.0,
				&top.Label,
				top.Count,
			)
		}
	}
}

func isConcentrationServiceCategory(category model.LabelCategory) bool {
	switch category {
	case model.LabelCategoryMixer,
		model.LabelCategoryExploit,
		model.LabelCategoryScam,
		model.LabelCategorySanctions,
		model.LabelCategoryExchange,
		model.LabelCategoryTrusted,
		model.LabelCategoryProtocol:
		return true
	default:
		return false
	}
}
