package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCrosschainL2Case(t *testing.T, dir string, caseID string) string {
	t.Helper()

	payload := map[string]any{
		"schema_version": "0.3.0",
		"case_family":    "crosschain_l2",
		"case_id":        caseID,
		"title":          "Cross-chain L2 Test Case",
		"description":    "Cross-chain L2 dataset-mode test case.",
		"chains_included": []string{
			"OPTIMISM",
			"POLYGON",
			"ARBITRUM",
		},
		"member_count": 3,
		"members": []map[string]any{
			{
				"chain":                   "OPTIMISM",
				"address":                 "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789",
				"tx_count":                764546,
				"inbound_count":           764546,
				"outbound_count":          0,
				"unique_counterparties":   281,
				"dominant_contract_share": 100.0,
				"failure_rate_pct":        0.06,
				"unique_emitters":         7,
				"unique_topic0s":          12,
				"bridge_hit_count":        0,
				"protocol_hit_count":      1,
				"stablecoin_hit_count":    0,
				"service_hit_count":       1,
				"top_bridge_family":       "",
				"top_protocol_family":     "erc4337_entrypoint_v06",
				"top_stablecoin_family":   "",
			},
			{
				"chain":                   "POLYGON",
				"address":                 "0x2b92478666516ef5d997490981aa3e00e9c8da9a",
				"tx_count":                2504895,
				"inbound_count":           1243392,
				"outbound_count":          1261503,
				"unique_counterparties":   1244465,
				"dominant_contract_share": 49.64,
				"failure_rate_pct":        0.42,
				"unique_emitters":         1834,
				"unique_topic0s":          221,
				"bridge_hit_count":        1,
				"protocol_hit_count":      3,
				"stablecoin_hit_count":    2,
				"service_hit_count":       0,
				"top_bridge_family":       "polygon_pos_bridge",
				"top_protocol_family":     "official_usdc",
				"top_stablecoin_family":   "official_usdc",
			},
			{
				"chain":                   "ARBITRUM",
				"address":                 "0x3bdb03ad7363152dfbc185ee23ebc93f0cf93fd1",
				"tx_count":                997251,
				"inbound_count":           416642,
				"outbound_count":          580609,
				"unique_counterparties":   187218,
				"dominant_contract_share": 41.78,
				"failure_rate_pct":        0.31,
				"unique_emitters":         624,
				"unique_topic0s":          109,
				"bridge_hit_count":        1,
				"protocol_hit_count":      2,
				"stablecoin_hit_count":    1,
				"service_hit_count":       0,
				"top_bridge_family":       "arbitrum_token_bridge",
				"top_protocol_family":     "official_usdc",
				"top_stablecoin_family":   "official_usdc",
			},
		},
		"crosschain_summary": map[string]any{
			"address_count":                     3,
			"chain_count":                       3,
			"total_tx_count":                    4266692,
			"max_dominant_contract_share":       100.0,
			"max_unique_counterparties":         1244465,
			"bridge_or_stablecoin_member_count": 2,
		},
		"curation_notes": map[string]any{
			"narrative":       "Cross-chain case capturing repeated service-like and operational L2 patterns.",
			"selection_basis": "Selected for validator routing and report rendering tests.",
		},
	}

	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal cross-chain case: %v", err)
	}

	path := filepath.Join(dir, caseID+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write cross-chain case: %v", err)
	}

	return path
}

func TestLoadDatasetMode_CrosschainL2Routing(t *testing.T) {
	dir := t.TempDir()
	path := writeCrosschainL2Case(t, dir, "crosschain-l2-broad-operational-hub")

	profile, ctx, err := loadDatasetMode(path)
	if err != nil {
		t.Fatalf("loadDatasetMode returned error: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile")
	}
	if ctx == nil {
		t.Fatal("expected report context")
	}

	if profile.Network != "MULTI-CHAIN" {
		t.Fatalf("expected MULTI-CHAIN network, got %q", profile.Network)
	}
	if !profile.IsValid {
		t.Fatal("expected valid profile")
	}
	if profile.RiskScore <= 0 {
		t.Fatalf("expected positive risk score, got %.2f", profile.RiskScore)
	}
	if len(profile.RiskReasons) == 0 {
		t.Fatal("expected cross-chain risk reasons")
	}
	if ctx.DatasetType != "Cross-chain L2 dataset" {
		t.Fatalf("unexpected dataset type: %q", ctx.DatasetType)
	}
	if ctx.CaseID != "crosschain-l2-broad-operational-hub" {
		t.Fatalf("unexpected case id: %q", ctx.CaseID)
	}
	if !profile.ReviewRecommended {
		t.Fatal("expected cross-chain operational hub case to recommend review")
	}
}

func TestRenderReport_UsesChainContextForCrosschainL2(t *testing.T) {
	dir := t.TempDir()
	path := writeCrosschainL2Case(t, dir, "crosschain-l2-repeated-contract-service-pattern")

	profile, ctx, err := loadDatasetMode(path)
	if err != nil {
		t.Fatalf("loadDatasetMode returned error: %v", err)
	}
	if profile == nil || ctx == nil {
		t.Fatal("expected profile and context")
	}

	out := renderReport(profile, ctx)

	if !strings.Contains(out, "Chain Context:") {
		t.Fatalf("expected report to contain Chain Context, got:\n%s", out)
	}
	if strings.Contains(out, "Layer 1 Context:") {
		t.Fatalf("did not expect Layer 1 Context in report, got:\n%s", out)
	}
	if !strings.Contains(out, "Cross-chain L2 dataset") {
		t.Fatalf("expected report to mention cross-chain dataset context, got:\n%s", out)
	}
	if !strings.Contains(out, "Chains included: OPTIMISM, POLYGON, ARBITRUM") {
		t.Fatalf("expected report to include chain context details, got:\n%s", out)
	}
	if !strings.Contains(out, "Case: Cross-chain L2 Test Case (crosschain-l2-repeated-contract-service-pattern)") {
		t.Fatalf("expected report to include case title/id, got:\n%s", out)
	}
}

func TestLoadDatasetMode_CrosschainStablecoinBridgeCase(t *testing.T) {
	dir := t.TempDir()
	path := writeCrosschainL2Case(t, dir, "crosschain-l2-stablecoin-bridge-operational-surface")

	profile, _, err := loadDatasetMode(path)
	if err != nil {
		t.Fatalf("loadDatasetMode returned error: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile")
	}

	if profile.Network != "MULTI-CHAIN" {
		t.Fatalf("expected MULTI-CHAIN network, got %q", profile.Network)
	}
	if profile.RiskScore < 5 {
		t.Fatalf("expected stablecoin/bridge cross-chain case to be reviewable, got %.2f", profile.RiskScore)
	}
	if !profile.ReviewRecommended {
		t.Fatal("expected stablecoin/bridge cross-chain case to recommend review")
	}
}
