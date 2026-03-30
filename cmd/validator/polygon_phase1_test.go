package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePolygonPhase1Case(t *testing.T, dir string, caseID string) string {
	t.Helper()

	payload := map[string]any{
		"chain":        "POLYGON",
		"case_id":      caseID,
		"title":        "Polygon Test Case",
		"description":  "Polygon Phase 1 dataset-mode test case.",
		"risk_posture": "TEST_POLYGON_PHASE1",
		"address":      "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789",
		"window_start": "2025-03-16T00:00:00Z",
		"window_end":   "2025-06-17T00:00:00Z",
		"layer2_summary": map[string]any{
			"first_seen":                  "2025-03-16T00:00:00Z",
			"last_seen":                   "2025-06-16T23:59:59Z",
			"tx_count":                    7517514,
			"inbound_count":               7517514,
			"outbound_count":              0,
			"unique_counterparties":       963,
			"unique_to_addresses":         1,
			"unique_from_addresses":       963,
			"dominant_direction":          "inbound",
			"dominant_counterparty_share": 3.93,
			"dominant_contract_share":     100.00,
		},
		"top_counterparties": []map[string]any{
			{"key": "0x1111111111111111111111111111111111111111", "count": 120000},
		},
		"top_to_addresses": []map[string]any{
			{"key": "0x2222222222222222222222222222222222222222", "count": 7517514},
		},
		"top_from_addresses": []map[string]any{
			{"key": "0x1111111111111111111111111111111111111111", "count": 120000},
		},
		"curation_notes": map[string]any{
			"narrative":       "This wallet is dominated by one contract destination across a very large transaction volume.",
			"selection_basis": "Selected for tx-only Phase 1 Polygon routing and report validation.",
		},
	}

	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal polygon case: %v", err)
	}

	path := filepath.Join(dir, caseID+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write polygon case: %v", err)
	}

	return path
}

func TestLoadDatasetMode_PolygonPhase1Routing(t *testing.T) {
	dir := t.TempDir()
	path := writePolygonPhase1Case(t, dir, "polygon-repeated-contract-service-like")

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

	if profile.Network != "POLYGON" {
		t.Fatalf("expected POLYGON network, got %q", profile.Network)
	}
	if !profile.IsValid {
		t.Fatal("expected valid profile")
	}
	if profile.RiskScore <= 0 {
		t.Fatalf("expected positive risk score, got %.2f", profile.RiskScore)
	}
	if len(profile.RiskReasons) == 0 {
		t.Fatal("expected polygon risk reasons")
	}
	if ctx.DatasetType != "Polygon Layer 2 dataset" {
		t.Fatalf("unexpected dataset type: %q", ctx.DatasetType)
	}
	if ctx.CaseID != "polygon-repeated-contract-service-like" {
		t.Fatalf("unexpected case id: %q", ctx.CaseID)
	}
}

func TestRenderReport_UsesChainContextForPolygon(t *testing.T) {
	dir := t.TempDir()
	path := writePolygonPhase1Case(t, dir, "polygon-repeated-contract-service-like")

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
	if !strings.Contains(out, "Polygon Layer 2 dataset") {
		t.Fatalf("expected report to mention Polygon dataset context, got:\n%s", out)
	}
	if !strings.Contains(out, "Unique counterparties: 963") {
		t.Fatalf("expected report to include chain context details, got:\n%s", out)
	}
}
