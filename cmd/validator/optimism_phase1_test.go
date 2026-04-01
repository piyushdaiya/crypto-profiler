package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeOptimismPhase1Case(t *testing.T, dir string, caseID string) string {
	t.Helper()

	payload := map[string]any{
		"chain":        "OPTIMISM",
		"case_id":      caseID,
		"title":        "Optimism Test Case",
		"description":  "Optimism Phase 1 dataset-mode test case.",
		"risk_posture": "TEST_OPTIMISM_PHASE1",
		"address":      "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789",
		"window_start": "2025-03-16T00:00:00Z",
		"window_end":   "2025-06-17T00:00:00Z",
		"layer2_summary": map[string]any{
			"first_seen":                       "2025-03-16T00:00:00Z",
			"last_seen":                        "2025-06-16T23:59:59Z",
			"tx_count":                         764546,
			"inbound_count":                    764546,
			"outbound_count":                   0,
			"unique_counterparties":            281,
			"unique_to_addresses":              1,
			"unique_from_addresses":            281,
			"unique_function_selectors":        1,
			"dominant_direction":               "inbound",
			"dominant_counterparty_share":      3.50,
			"dominant_contract_share":          100.00,
			"dominant_function_selector_share": 100.00,
		},
		"top_counterparties": []map[string]any{
			{"key": "0x596680f2ea1bdb041570c74fb5fc8c0c0a9fad80", "count": 26750},
		},
		"top_to_addresses": []map[string]any{
			{"key": "0xroutercontract000000000000000000000000000001", "count": 764546},
		},
		"top_from_addresses": []map[string]any{
			{"key": "0x596680f2ea1bdb041570c74fb5fc8c0c0a9fad80", "count": 26750},
		},
		"top_function_selectors": []map[string]any{
			{"key": "0x12345678", "count": 764546},
		},
		"curation_notes": map[string]any{
			"narrative":       "This wallet is dominated by one contract destination and one function selector across a very large transaction volume.",
			"selection_basis": "Selected for tx-only Phase 1 Optimism routing and report validation.",
		},
	}

	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal optimism case: %v", err)
	}

	path := filepath.Join(dir, caseID+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write optimism case: %v", err)
	}

	return path
}

func TestLoadDatasetMode_OptimismPhase1Routing(t *testing.T) {
	dir := t.TempDir()
	path := writeOptimismPhase1Case(t, dir, "optimism-repeated-contract-router-like")

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

	if profile.Network != "OPTIMISM" {
		t.Fatalf("expected OPTIMISM network, got %q", profile.Network)
	}
	if profile.Address != "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789" {
		t.Fatalf("unexpected address: %q", profile.Address)
	}
	if !profile.IsValid {
		t.Fatal("expected valid profile")
	}
	if profile.RiskScore <= 0 {
		t.Fatalf("expected positive risk score, got %.2f", profile.RiskScore)
	}
	if len(profile.RiskReasons) == 0 {
		t.Fatal("expected optimism risk reasons")
	}
	if ctx.DatasetType != "Optimism Layer 2 dataset" {
		t.Fatalf("unexpected dataset type: %q", ctx.DatasetType)
	}
	if ctx.CaseID != "optimism-repeated-contract-router-like" {
		t.Fatalf("unexpected case id: %q", ctx.CaseID)
	}
}

func TestRenderReport_UsesChainContextForOptimism(t *testing.T) {
	dir := t.TempDir()
	path := writeOptimismPhase1Case(t, dir, "optimism-repeated-contract-router-like")

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
	if !strings.Contains(out, "Optimism Layer 2 dataset") {
		t.Fatalf("expected report to mention Optimism dataset context, got:\n%s", out)
	}
	if !strings.Contains(out, "Unique function selectors: 1") {
		t.Fatalf("expected report to include chain context details, got:\n%s", out)
	}
}
