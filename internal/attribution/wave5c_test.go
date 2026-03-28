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
	"testing"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func TestApplyWave5CContext_ActorAwareRiskyAggregation(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0x1000000000000000000000000000000000000001",
	    "network": "EVM",
	    "label": "Bad Mixer Deposit Wallet A",
	    "actor": "Bad Mixer",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.96,
	    "contextual": false,
	    "risk_escalating": true
	  },
	  {
	    "address": "0x1000000000000000000000000000000000000002",
	    "network": "EVM",
	    "label": "Bad Mixer Deposit Wallet B",
	    "actor": "Bad Mixer",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.94,
	    "contextual": false,
	    "risk_escalating": true
	  }
	]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[]`))
	ResetDefaultResolverForTesting()

	profile := &model.WalletProfile{
		Address:       "0xprofile0000000000000000000000000000000001",
		Network:       "EVM",
		RiskBreakdown: model.RiskCategory{},
	}

	ApplyWave5CContext(profile, Wave5CInput{
		Network: "EVM",
		Counterparties: []InteractionCounterparty{
			{Address: "0x1000000000000000000000000000000000000001", Interactions: 12, InboundCount: 12},
			{Address: "0x1000000000000000000000000000000000000002", Interactions: 9, InboundCount: 9},
		},
	})

	if !hasReasonCode(profile.RiskReasons, "actor_repeated_risky_interaction") {
		t.Fatalf("expected actor-aware risky repeated-interaction reason, got %+v", profile.RiskReasons)
	}
	if !hasReasonCode(profile.RiskReasons, "actor_risky_concentration") {
		t.Fatalf("expected actor-aware risky concentration reason, got %+v", profile.RiskReasons)
	}
	if len(profile.AttributionInsights) == 0 || profile.AttributionInsights[0].Type != model.AttributionInsightClusterGrouping {
		t.Fatalf("expected cluster-aware insight, got %+v", profile.AttributionInsights)
	}
}

func TestApplyWave5CContext_ContextualClusterGroupingAndConcentration(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0x2000000000000000000000000000000000000001",
	    "network": "EVM",
	    "label": "Protocol Router",
	    "actor": "Sample Protocol",
	    "category": "PROTOCOL",
	    "risk_class": "TRUSTED_SERVICE",
	    "confidence": 0.97,
	    "contextual": true,
	    "risk_escalating": false
	  },
	  {
	    "address": "0x2000000000000000000000000000000000000002",
	    "network": "EVM",
	    "label": "Protocol Pool",
	    "actor": "Sample Protocol",
	    "category": "PROTOCOL",
	    "risk_class": "TRUSTED_SERVICE",
	    "confidence": 0.93,
	    "contextual": true,
	    "risk_escalating": false
	  }
	]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[]`))
	ResetDefaultResolverForTesting()

	profile := &model.WalletProfile{
		Address: "0x2000000000000000000000000000000000000001",
		Network: "EVM",
		Attribution: &model.ResolvedAttribution{
			Address:        "0x2000000000000000000000000000000000000001",
			Network:        "EVM",
			Label:          "Protocol Router",
			Actor:          "Sample Protocol",
			Category:       model.LabelCategoryProtocol,
			RiskClass:      model.AttributionRiskClassTrustedService,
			Confidence:     0.97,
			Contextual:     true,
			SourceTier:     model.AttributionSourceTierPrimaryStructured,
			SourceType:     model.AttributionSourceTypeGraphSense,
			SourceName:     "graphsense_structured_labels",
			BaseConfidence: 0.97,
		},
		RiskBreakdown: model.RiskCategory{Reputation: 8},
		RiskScore:     2.4,
	}

	ApplyWave5CContext(profile, Wave5CInput{
		Network: "EVM",
		Counterparties: []InteractionCounterparty{
			{Address: "0x2000000000000000000000000000000000000002", Interactions: 120, InboundCount: 40, OutboundCount: 80},
		},
	})

	if !hasReasonCode(profile.RiskReasons, "actor_contextual_concentration") {
		t.Fatalf("expected contextual actor concentration reason, got %+v", profile.RiskReasons)
	}
	if !hasReasonCode(profile.RiskReasons, "actor_contextual_repeated_interaction") {
		t.Fatalf("expected contextual repeated-interaction reason, got %+v", profile.RiskReasons)
	}
	if profile.RiskScore >= 2.4 {
		t.Fatalf("expected contextual actor refinement to suppress score, got %v", profile.RiskScore)
	}
}

func TestApplyWave5CContext_SecondaryOnlyRemainsInsightOnly(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[
	  {
	    "address": "bc1qsecondary0000000000000000000000000000000",
	    "network": "BITCOIN",
	    "label": "WalletExplorer-style Cluster",
	    "actor": "Weak Cluster",
	    "category": "EXCHANGE",
	    "risk_class": "EXCHANGE",
	    "confidence": 0.56,
	    "contextual": true,
	    "risk_escalating": false
	  }
	]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[]`))
	ResetDefaultResolverForTesting()

	profile := &model.WalletProfile{
		Address:       "bc1qprofile000000000000000000000000000000000",
		Network:       "BITCOIN",
		RiskBreakdown: model.RiskCategory{Reputation: 6},
		RiskScore:     1.8,
	}

	ApplyWave5CContext(profile, Wave5CInput{
		Network: "BITCOIN",
		Counterparties: []InteractionCounterparty{
			{Address: "bc1qsecondary0000000000000000000000000000000", Interactions: 500, OutboundCount: 500},
		},
	})

	if hasReasonCode(profile.RiskReasons, "actor_contextual_concentration") || hasReasonCode(profile.RiskReasons, "actor_repeated_risky_interaction") {
		t.Fatalf("expected secondary-only actor refinement to remain bounded, got %+v", profile.RiskReasons)
	}
	if len(profile.AttributionInsights) == 0 {
		t.Fatalf("expected insight-only output for secondary-only attribution")
	}
}

func TestApplyWave5CContext_PassThroughAndUTurnPatterns(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0x3000000000000000000000000000000000000001",
	    "network": "EVM",
	    "label": "Trusted Exchange",
	    "actor": "Trusted Exchange",
	    "category": "EXCHANGE",
	    "risk_class": "EXCHANGE",
	    "confidence": 0.95,
	    "contextual": true,
	    "risk_escalating": false
	  },
	  {
	    "address": "0x3000000000000000000000000000000000000002",
	    "network": "EVM",
	    "label": "Bad Mixer",
	    "actor": "Bad Mixer",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.98,
	    "contextual": false,
	    "risk_escalating": true
	  }
	]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[]`))
	ResetDefaultResolverForTesting()

	base := time.Date(2025, 3, 16, 0, 0, 0, 0, time.UTC)
	profile := &model.WalletProfile{
		Address:       "0xprofilepassthrough000000000000000000000001",
		Network:       "EVM",
		RiskBreakdown: model.RiskCategory{},
	}

	ApplyWave5CContext(profile, Wave5CInput{
		Network: "EVM",
		Flows: []FlowObservation{
			{Timestamp: base, Direction: "inbound", Counterparty: "0x3000000000000000000000000000000000000001"},
			{Timestamp: base.Add(5 * time.Minute), Direction: "outbound", Counterparty: "0x3000000000000000000000000000000000000002"},
			{Timestamp: base.Add(10 * time.Minute), Direction: "inbound", Counterparty: "0x3000000000000000000000000000000000000002"},
			{Timestamp: base.Add(15 * time.Minute), Direction: "outbound", Counterparty: "0x3000000000000000000000000000000000000002"},
		},
	})

	if !hasReasonCode(profile.RiskReasons, "actor_pass_through_risky_exposure") {
		t.Fatalf("expected pass-through risky exposure reason, got %+v", profile.RiskReasons)
	}
	if !hasReasonCode(profile.RiskReasons, "actor_u_turn_risky_service") {
		t.Fatalf("expected risky U-turn reason, got %+v", profile.RiskReasons)
	}

	var hasPassThrough, hasNearExposure, hasUTurn bool
	for _, insight := range profile.AttributionInsights {
		switch insight.Type {
		case model.AttributionInsightPassThrough:
			hasPassThrough = true
		case model.AttributionInsightNearExposure:
			hasNearExposure = true
		case model.AttributionInsightUTurn:
			hasUTurn = true
		}
	}
	if !hasPassThrough || !hasNearExposure || !hasUTurn {
		t.Fatalf("expected pass-through, near-exposure, and U-turn insights, got %+v", profile.AttributionInsights)
	}
}
