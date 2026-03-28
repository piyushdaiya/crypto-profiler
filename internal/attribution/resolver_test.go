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
	"os"
	"path/filepath"
	"testing"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func TestResolveAddress_LocalOverrideWinsOverPrimaryStructured(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0xabc0000000000000000000000000000000000000",
	    "network": "EVM",
	    "label": "High Risk Mixer",
	    "actor": "Risk Actor",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.99,
	    "contextual": false,
	    "risk_escalating": true
	  }
	]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{
	  "0xabc0000000000000000000000000000000000000": {
	    "address": "0xabc0000000000000000000000000000000000000",
	    "name": "Demo Trusted Override",
	    "category": "TRUSTED",
	    "severity": "LOW",
	    "confidence": "HIGH",
	    "source": "bootstrap_entities",
	    "trusted": true
	  }
	}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[]`))
	ResetDefaultResolverForTesting()

	resolved, ok := ResolveAddress("0xabc0000000000000000000000000000000000000", "EVM")
	if !ok {
		t.Fatalf("expected attribution resolution to succeed")
	}

	if resolved.SourceTier != model.AttributionSourceTierLocalOverride {
		t.Fatalf("expected local override tier, got %q", resolved.SourceTier)
	}
	if resolved.Category != model.LabelCategoryTrusted {
		t.Fatalf("expected trusted override category, got %q", resolved.Category)
	}
	if !resolved.Contextual {
		t.Fatalf("expected resolved attribution to be contextual")
	}
}

func TestResolveAddress_MiningPoolContextResolves(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[
	  {
	    "address": "bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx",
	    "network": "BITCOIN",
	    "label": "Foundry USA Pool Payout",
	    "actor": "Foundry USA Pool",
	    "category": "MINING_POOL",
	    "risk_class": "MINING_POOL",
	    "confidence": 0.96,
	    "contextual": true,
	    "risk_escalating": false
	  }
	]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[]`))
	ResetDefaultResolverForTesting()

	resolved, ok := ResolveAddress("bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx", "BITCOIN")
	if !ok {
		t.Fatalf("expected mining-pool attribution to resolve")
	}
	if resolved.Category != model.LabelCategoryMiningPool {
		t.Fatalf("expected mining-pool category, got %q", resolved.Category)
	}
	if !resolved.Contextual || resolved.Escalating {
		t.Fatalf("expected contextual mining-pool resolution, got contextual=%v escalating=%v", resolved.Contextual, resolved.Escalating)
	}
}

func TestApplyTier1Attribution_RiskyEscalation(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
	    "network": "EVM",
	    "label": "Tornado Cash Router",
	    "actor": "Tornado Cash",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.99,
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
		Address:       "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
		Network:       "EVM",
		IsValid:       true,
		RiskBreakdown: model.RiskCategory{},
	}

	ApplyTier1Attribution(profile)

	if profile.RiskScore <= 0 {
		t.Fatalf("expected positive score after risky attribution, got %v", profile.RiskScore)
	}
	if !hasReasonCode(profile.RiskReasons, "tier1_profile_risky_attribution") {
		t.Fatalf("expected risky attribution reason, got %+v", profile.RiskReasons)
	}
	if profile.Attribution == nil || profile.Attribution.Actor != "Tornado Cash" {
		t.Fatalf("expected resolved actor Tornado Cash, got %+v", profile.Attribution)
	}
}

func TestApplyTier1Attribution_ContextualSuppression(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
	    "network": "EVM",
	    "label": "Uniswap V2 Router",
	    "actor": "Uniswap",
	    "category": "PROTOCOL",
	    "risk_class": "TRUSTED_SERVICE",
	    "confidence": 0.98,
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
		Address: "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
		Network: "EVM",
		IsValid: true,
		RiskBreakdown: model.RiskCategory{
			Fraud:      12,
			Reputation: 15,
		},
		RiskScore: 10.5,
	}

	ApplyTier1Attribution(profile)

	if profile.RiskScore >= 10.5 {
		t.Fatalf("expected contextual attribution to reduce score, got %v", profile.RiskScore)
	}
	if !hasReasonCode(profile.RiskReasons, "tier1_profile_contextual_attribution") {
		t.Fatalf("expected contextual attribution reason, got %+v", profile.RiskReasons)
	}
	if profile.Attribution == nil || !profile.Attribution.Contextual {
		t.Fatalf("expected contextual resolved attribution, got %+v", profile.Attribution)
	}
}

func TestResolveAddress_SecondaryCorroborationBoostsConfidence(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
	    "network": "EVM",
	    "label": "Tornado Cash Router",
	    "actor": "Tornado Cash",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.92,
	    "contextual": false,
	    "risk_escalating": true
	  }
	]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[
	  {
	    "address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
	    "network": "EVM",
	    "label": "Tornado Cash Router (Secondary)",
	    "actor": "Tornado Cash",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.60,
	    "contextual": false,
	    "risk_escalating": true
	  }
	]`))
	ResetDefaultResolverForTesting()

	resolved, ok := ResolveAddress("0xd90e2f925da726b50c4ed8d0fb90ad053324f31b", "EVM")
	if !ok {
		t.Fatalf("expected attribution resolution to succeed")
	}
	if len(resolved.CorroboratingSources) != 1 {
		t.Fatalf("expected 1 corroborating source, got %+v", resolved.CorroboratingSources)
	}
	if resolved.Confidence <= resolved.BaseConfidence {
		t.Fatalf("expected corroboration to raise confidence, got base=%v resolved=%v", resolved.BaseConfidence, resolved.Confidence)
	}
}

func TestResolveAddress_LoneSecondarySourceRemainsBounded(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[
	  {
	    "address": "bc1q8ys49pxp3c6um7enemwdkl4ud5fwwg2rpdegxu",
	    "network": "BITCOIN",
	    "label": "WalletExplorer-style Exchange Cluster",
	    "actor": "Sample Exchange Cluster",
	    "category": "EXCHANGE",
	    "risk_class": "EXCHANGE",
	    "confidence": 0.91,
	    "contextual": true,
	    "risk_escalating": false
	  }
	]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[]`))
	ResetDefaultResolverForTesting()

	resolved, ok := ResolveAddress("bc1q8ys49pxp3c6um7enemwdkl4ud5fwwg2rpdegxu", "BITCOIN")
	if !ok {
		t.Fatalf("expected secondary attribution resolution to succeed")
	}
	if !resolved.SecondaryOnly {
		t.Fatalf("expected secondary-only attribution, got %+v", resolved)
	}
	if resolved.Confidence > 0.65 {
		t.Fatalf("expected bounded secondary confidence, got %v", resolved.Confidence)
	}

	profile := &model.WalletProfile{
		Address: "bc1q8ys49pxp3c6um7enemwdkl4ud5fwwg2rpdegxu",
		Network: "BITCOIN",
		IsValid: true,
		RiskBreakdown: model.RiskCategory{
			Reputation: 8,
		},
		RiskScore: 2.4,
	}
	ApplyAttributionContext(profile)

	if !hasReasonCode(profile.RiskReasons, "secondary_profile_contextual_attribution") {
		t.Fatalf("expected bounded secondary contextual reason, got %+v", profile.RiskReasons)
	}
	if profile.RiskScore >= 2.4 {
		t.Fatalf("expected modest contextual reduction, got %v", profile.RiskScore)
	}
}

func TestResolveAddress_Tier1PrecedenceOverSecondaryConflict(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
	    "network": "EVM",
	    "label": "Uniswap V2 Router",
	    "actor": "Uniswap",
	    "category": "PROTOCOL",
	    "risk_class": "TRUSTED_SERVICE",
	    "confidence": 0.95,
	    "contextual": true,
	    "risk_escalating": false
	  }
	]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[
	  {
	    "address": "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
	    "network": "EVM",
	    "label": "Suspicious Mixer Router",
	    "actor": "Unknown Mixer",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.64,
	    "contextual": false,
	    "risk_escalating": true
	  }
	]`))
	ResetDefaultResolverForTesting()

	resolved, ok := ResolveAddress("0x7a250d5630b4cf539739df2c5dacb4c659f2488d", "EVM")
	if !ok {
		t.Fatalf("expected resolution to succeed")
	}
	if resolved.SourceTier != model.AttributionSourceTierPrimaryStructured {
		t.Fatalf("expected Tier 1 precedence, got %q", resolved.SourceTier)
	}
	if resolved.RiskClass != model.AttributionRiskClassTrustedService {
		t.Fatalf("expected trusted-service primary to win, got %q", resolved.RiskClass)
	}
	if len(resolved.ConflictingSources) != 1 {
		t.Fatalf("expected 1 conflicting source, got %+v", resolved.ConflictingSources)
	}
}

func TestApplyAttributionContext_ConflictAddsNoteOnlyReason(t *testing.T) {
	t.Setenv("GRAPHSENSE_LABELS_PATH", writeFile(t, "graphsense.json", `[
	  {
	    "address": "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
	    "network": "EVM",
	    "label": "Uniswap V2 Router",
	    "actor": "Uniswap",
	    "category": "PROTOCOL",
	    "risk_class": "TRUSTED_SERVICE",
	    "confidence": 0.95,
	    "contextual": true,
	    "risk_escalating": false
	  }
	]`))
	t.Setenv("BOOTSTRAP_LABELS_PATH", writeFile(t, "bootstrap.json", `{}`))
	t.Setenv("BITCOIN_MINING_POOLS_PATH", writeFile(t, "mining.json", `[]`))
	t.Setenv("WALLET_EXPLORER_LABELS_PATH", writeFile(t, "wallet-explorer.json", `[]`))
	t.Setenv("CORROBORATING_LABELS_PATH", writeFile(t, "corroborating.json", `[
	  {
	    "address": "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
	    "network": "EVM",
	    "label": "Secondary Mixer Claim",
	    "actor": "Unknown Mixer",
	    "category": "MIXER",
	    "risk_class": "ILLICIT_SERVICE",
	    "confidence": 0.60,
	    "contextual": false,
	    "risk_escalating": true
	  }
	]`))
	ResetDefaultResolverForTesting()

	profile := &model.WalletProfile{
		Address: "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
		Network: "EVM",
		IsValid: true,
		RiskBreakdown: model.RiskCategory{
			Fraud:      12,
			Reputation: 15,
		},
		RiskScore: 10.5,
	}
	ApplyAttributionContext(profile)

	if !hasReasonCode(profile.RiskReasons, "attribution_source_conflict_observed") {
		t.Fatalf("expected conflict note reason, got %+v", profile.RiskReasons)
	}
}

func writeFile(t *testing.T, name, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write fixture %s: %v", name, err)
	}
	return path
}
