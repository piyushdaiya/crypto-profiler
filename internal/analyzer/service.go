package analyzer

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
	"github.com/piyushdaiya/crypto-profiler/internal/watchlist"
)

// ---------------------------------------------------------
// CORE: Investigator Logic
// ---------------------------------------------------------

// Known heuristic threats (fallback/supplementary to OFAC)
var knownThreats = map[string]string{
	"0xd90e2f925da726b50c4ed8d0fb90ad053324f31b": "Tornado Cash Router",
}

// Investigate analyzes risk using both Heuristics and the Remote Watchlist Engine
func Investigate(profile *model.WalletProfile, txs []model.Transaction) {
	var fraudScore, repScore, lendScore float64
	var reasons []model.RiskReason

	// Helper to track risk
	addRisk := func(category, desc string, offset float64) {
		reasons = append(reasons, model.RiskReason{
			Category:    category,
			Description: desc,
			Offset:      offset,
		})
		switch category {
		case "FRAUD":
			fraudScore += offset
		case "REPUTATION":
			repScore += offset
		case "LENDING":
			lendScore += offset
		}
	}

	// ---------------------------------------------------------
	// 1. CALL REMOTE WATCHLIST ENGINE
	// ---------------------------------------------------------
	engineResp, err := watchlist.CheckWatchlist(profile.Address)

	if err != nil {
		// FAIL OPEN: If profiler is down, warn but don't crash
		addRisk("SYSTEM", "⚠️ Watchlist Engine Unavailable - Sanctions Check Skipped", 0.0)
		profile.ValidationDetails += " | [Warning: Sanctions DB Offline]"
	} else if engineResp.Sanctioned {
		// CRITICAL HIT
		addRisk("FRAUD", fmt.Sprintf("CRITICAL: %s Sanctioned Address (%s)", engineResp.Source, engineResp.Currency), 100.0)
		addRisk("REPUTATION", "Government Blacklisted Entity", 100.0)
		addRisk("LENDING", "Prohibited: Federal Sanctions", 100.0)

		// Force Max Score Immediately
		profile.RiskScore = 100.0
		profile.RiskGrade = "CRITICAL (Sanctioned)"
		profile.RiskBreakdown = model.RiskCategory{100, 100, 100}
		profile.RiskReasons = reasons
		return // Stop processing
	}

	// ---------------------------------------------------------
	// 2. HEURISTICS (Age, Velocity, Mixers)
	// ---------------------------------------------------------

	// Age Check
	if profile.FirstSeen != nil {
		hoursOld := time.Since(*profile.FirstSeen).Hours()
		if hoursOld > 24*365 {
			addRisk("REPUTATION", "Established History (>1 Year)", -10.0)
		} else if hoursOld < 24 {
			addRisk("FRAUD", "Freshly Created Wallet (<24h)", 35.0)
		}
	}

	// Interactions Check
	directThreat := false
	for _, tx := range txs {
		otherParty := ""
		if strings.EqualFold(tx.From, profile.Address) {
			otherParty = strings.ToLower(tx.To)
		} else {
			otherParty = strings.ToLower(tx.From)
		}

		if label, isThreat := knownThreats[otherParty]; isThreat {
			if !directThreat {
				addRisk("FRAUD", fmt.Sprintf("Direct Interaction with %s", label), 55.0)
				directThreat = true
			}
		}
	}

	// Velocity Check
	if profile.TxCount > 0 && profile.FirstSeen != nil {
		hoursActive := time.Since(*profile.FirstSeen).Hours()
		if hoursActive < 1 {
			hoursActive = 1
		}

		txPerHour := float64(profile.TxCount) / hoursActive
		if txPerHour > 20.0 {
			addRisk("FRAUD", "High Velocity Behavior (Potential Bot)", 25.0)
		}
	}

	// ---------------------------------------------------------
	// 3. FINALIZE SCORE
	// ---------------------------------------------------------

	// Normalize
	fraudScore = clamp(fraudScore, 0, 100)
	repScore = clamp(repScore, 0, 100)
	lendScore = clamp(lendScore, 0, 100)

	combinedRisk := (fraudScore * 0.5) + (repScore * 0.3) + (lendScore * 0.2)

	grade := "UNKNOWN"
	if combinedRisk < 10 {
		grade = "EXCELLENT (Safe)"
	} else if combinedRisk < 35 {
		grade = "LOW (Neutral)"
	} else if combinedRisk < 60 {
		grade = "WARNING (Elevated)"
	} else {
		grade = "FAILING (High Risk)"
	}

	profile.RiskScore = math.Round(combinedRisk*100) / 100
	profile.RiskGrade = grade
	profile.RiskBreakdown = model.RiskCategory{
		Fraud:      math.Round(fraudScore*100) / 100,
		Reputation: math.Round(repScore*100) / 100,
		Lending:    math.Round(lendScore*100) / 100,
	}
	profile.RiskReasons = reasons
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
