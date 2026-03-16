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
			otherParty = strings.ToLower(tx.To)
		} else {
			otherParty = strings.ToLower(tx.From)
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
	// 5. PLACEHOLDER FOR FUTURE PATTERNS
	// ---------------------------------------------------------
	// Future examples:
	// - rapid_passthrough_behavior
	// - peeling_chain_behavior
	// - hop_to_mixer_proximity
	// - dormant_wallet_reactivation
	// - suspicious_inbound_outbound_ratio

	// ---------------------------------------------------------
	// 6. APPLY COMBINATION RULES
	// ---------------------------------------------------------
	hits = append(hits, applyCombinationRules(hits)...)

	// ---------------------------------------------------------
	// 7. CONVERT HITS TO SCORES + REASONS
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
	// 8. FINALIZE SCORE
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
