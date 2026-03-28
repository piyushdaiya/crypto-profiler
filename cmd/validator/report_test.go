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

package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/address"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func TestRenderReport_IncludesAnalystFacingSections(t *testing.T) {
	firstSeen := time.Date(2025, 3, 16, 0, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2025, 6, 17, 0, 0, 0, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:           "0xtest",
		Network:           "EVM",
		IsValid:           true,
		ValidationDetails: "Loaded curated ERC-20 Layer 1 case",
		TxCount:           42,
		FirstSeen:         &firstSeen,
		LastSeen:          &lastSeen,
		RiskScore:         14.2,
		RiskGrade:         "LOW (Reviewable)",
		ReviewRecommended: true,
		RiskReasons: []model.RiskReason{
			{
				Code:        "erc20_noisy_token_inbound_surface",
				Category:    "FRAUD",
				Description: "Inbound-heavy ERC-20 surface observed (1200 counterparties across 40 tokens)",
				Offset:      14.0,
			},
		},
		Attribution: &model.ResolvedAttribution{
			Address:    "0xtest",
			Network:    "EVM",
			Label:      "Sample Exchange Wallet",
			Actor:      "Sample Exchange",
			Category:   model.LabelCategoryExchange,
			RiskClass:  model.AttributionRiskClassExchange,
			Confidence: 0.93,
			Contextual: true,
			Escalating: false,
			SourceName: "graphsense_structured_labels",
			SourceTier: model.AttributionSourceTierPrimaryStructured,
			SourceType: model.AttributionSourceTypeGraphSense,
		},
	}

	context := &reportContext{
		Mode:           "dataset",
		DatasetType:    "ERC-20 Layer 1 dataset",
		CaseID:         "erc20-noisy-inbound",
		CaseTitle:      "ERC-20 Noisy Inbound Broad Token Surface",
		Narrative:      "Representative inbound-heavy token surface with broad counterparties and repeated operational traffic.",
		Interpretation: "Synthetic interpretation for report rendering tests.",
		ChainContext: []string{
			"Dominant direction: inbound",
			"Unique token contracts: 40",
		},
		TopCounterparties: []reportCounterparty{
			{
				Address: "0x1111111111111111111111111111111111111111",
				Detail:  "120 interactions across 8 tokens",
			},
		},
	}

	report := renderReport(profile, context)

	for _, want := range []string{
		"Crypto Profiler Analyst Report",
		"Address: 0xtest",
		"Dataset Context: ERC-20 Layer 1 dataset",
		"Case: ERC-20 Noisy Inbound Broad Token Surface (erc20-noisy-inbound)",
		"Risk Score: 14.20",
		"Attribution:",
		"Resolved Label: Sample Exchange Wallet",
		"Actor: Sample Exchange",
		"Disposition: contextual / benign",
		"Top Reasons:",
		"Top Counterparties:",
		"Interpretation:",
		"Layer 1 Context:",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got:\n%s", want, report)
		}
	}
}

func TestRun_ReportModeUsesHumanReadableOutputForLiveProfile(t *testing.T) {
	firstSeen := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	profile := &model.WalletProfile{
		Address:           "0xtest",
		Network:           "EVM",
		IsValid:           true,
		ValidationDetails: "stubbed profile",
		IsActive:          true,
		Balance:           "1.0000 ETH",
		TxCount:           3,
		FirstSeen:         &firstSeen,
		LastSeen:          &lastSeen,
		RiskScore:         12.5,
		RiskGrade:         "LOW (Reviewable)",
		ReviewRecommended: true,
		RiskReasons: []model.RiskReason{
			{
				Code:        "stub_reason",
				Category:    "FRAUD",
				Description: "stub reason",
				Offset:      12.5,
			},
		},
	}

	strategies := []address.ChainStrategy{
		&stubStrategy{
			name:    "STUB",
			match:   true,
			profile: profile,
		},
	}

	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run([]string{"--report", "0xtest"}, &out, &errOut, strategies)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, errOut.String())
	}

	report := out.String()
	if strings.HasPrefix(strings.TrimSpace(report), "{") {
		t.Fatalf("expected human-readable report output, got JSON:\n%s", report)
	}
	if !strings.Contains(report, "Crypto Profiler Analyst Report") {
		t.Fatalf("expected report header, got:\n%s", report)
	}
	if !strings.Contains(report, "Balance: 1.0000 ETH") {
		t.Fatalf("expected live report context to include balance, got:\n%s", report)
	}
}

func TestRun_DatasetModeReportIncludesChainSpecificContext(t *testing.T) {
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", repoPath("data", "labels", "bootstrap_entities.json"))
	setEnvForTest(t, "GRAPHSENSE_LABELS_PATH", repoPath("data", "labels", "tier1_graphsense_entities.json"))
	setEnvForTest(t, "BITCOIN_MINING_POOLS_PATH", repoPath("data", "labels", "tier1_bitcoin_mining_pools.json"))

	tests := []struct {
		name                string
		args                []string
		wantDetailSubstring string
		wantCaseSubstring   string
		wantAttribution     string
	}{
		{
			name:                "ethereum curated report",
			args:                []string{"--report", "--dataset", repoPath("data", "cases", "curated-enriched", "tornado-router-high-risk.json")},
			wantDetailSubstring: "Trace activity:",
			wantCaseSubstring:   "Case: High-Risk Mixer Infrastructure (tornado-router-high-risk)",
			wantAttribution:     "Resolved Label: Tornado Cash Router",
		},
		{
			name:                "solana curated report",
			args:                []string{"--report", "--dataset", repoPath("data", "cases", "curated-solana", "solana-stablecoin-authority-operator.json")},
			wantDetailSubstring: "Dominant role: authority",
			wantCaseSubstring:   "Case: Solana Stablecoin Authority-Heavy Operator (solana-stablecoin-authority-operator)",
		},
		{
			name:                "bitcoin curated report",
			args:                []string{"--report", "--dataset", repoPath("data", "cases", "curated-bitcoin", "bitcoin-broad-spend-heavy-operational-hub.json")},
			wantDetailSubstring: "Inbound receipts:",
			wantCaseSubstring:   "Case: Bitcoin Broad Spend-Heavy Operational Hub (bitcoin-broad-spend-heavy-operational-hub)",
		},
		{
			name:                "erc20 curated report",
			args:                []string{"--report", "--dataset", repoPath("data", "cases", "curated-erc20", "erc20-uniswap-v2-router-trusted-token-hub.json")},
			wantDetailSubstring: "Unique token contracts:",
			wantCaseSubstring:   "Case: ERC-20 Trusted Protocol Token Hub (erc20-uniswap-v2-router-trusted-token-hub)",
			wantAttribution:     "Resolved Label: Uniswap V2 Router",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			var errOut bytes.Buffer

			code := run(tt.args, &out, &errOut, nil)
			if code != 0 {
				t.Fatalf("expected exit code 0, got %d; stderr=%s", code, errOut.String())
			}

			report := out.String()
			if strings.HasPrefix(strings.TrimSpace(report), "{") {
				t.Fatalf("expected human-readable report output, got JSON:\n%s", report)
			}
			if !strings.Contains(report, tt.wantCaseSubstring) {
				t.Fatalf("expected report to contain case context %q, got:\n%s", tt.wantCaseSubstring, report)
			}
			if !strings.Contains(report, tt.wantDetailSubstring) {
				t.Fatalf("expected report to contain chain-specific detail %q, got:\n%s", tt.wantDetailSubstring, report)
			}
			if tt.wantAttribution != "" && !strings.Contains(report, tt.wantAttribution) {
				t.Fatalf("expected report to contain attribution detail %q, got:\n%s", tt.wantAttribution, report)
			}
		})
	}
}
