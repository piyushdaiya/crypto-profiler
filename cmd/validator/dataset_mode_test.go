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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func TestRun_DatasetModeRoutesCuratedCasesAcrossChains(t *testing.T) {
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", repoPath("data", "labels", "bootstrap_entities.json"))
	setEnvForTest(t, "GRAPHSENSE_LABELS_PATH", repoPath("data", "labels", "tier1_graphsense_entities.json"))
	setEnvForTest(t, "BITCOIN_MINING_POOLS_PATH", repoPath("data", "labels", "tier1_bitcoin_mining_pools.json"))
	setEnvForTest(t, "WALLET_EXPLORER_LABELS_PATH", repoPath("data", "labels", "tier2_wallet_explorer_entities.json"))
	setEnvForTest(t, "CORROBORATING_LABELS_PATH", repoPath("data", "labels", "tier2_corroborating_entities.json"))

	tests := []struct {
		name                string
		path                string
		wantNetwork         string
		wantDetailSubstring string
		wantReasonCode      string
		wantAttribution     model.LabelCategory
		wantInsightType     model.AttributionInsightType
	}{
		{
			name:                "ethereum trace-enriched curated path",
			path:                repoPath("data", "cases", "curated-enriched", "tornado-router-high-risk.json"),
			wantNetwork:         "EVM",
			wantDetailSubstring: "Trace Summary",
			wantReasonCode:      "dataset_trace_activity_observed",
			wantAttribution:     model.LabelCategoryMixer,
		},
		{
			name:                "solana curated stablecoin path",
			path:                repoPath("data", "cases", "curated-solana", "solana-stablecoin-authority-operator.json"),
			wantNetwork:         "SOLANA",
			wantDetailSubstring: "Loaded curated Solana stablecoin-flow case",
			wantReasonCode:      "solana_authority_heavy_stablecoin_operator",
		},
		{
			name:                "bitcoin curated layer1 path",
			path:                repoPath("data", "cases", "curated-bitcoin", "bitcoin-legacy-mixed-flow-broad-value.json"),
			wantNetwork:         "BITCOIN",
			wantDetailSubstring: "Loaded curated Bitcoin Layer 1 case",
			wantReasonCode:      "bitcoin_legacy_mixed_flow_broad_value",
		},
		{
			name:                "erc20 curated layer1 path",
			path:                repoPath("data", "cases", "curated-erc20", "erc20-uniswap-v2-router-trusted-token-hub.json"),
			wantNetwork:         "EVM",
			wantDetailSubstring: "Loaded curated ERC-20 Layer 1 case",
			wantReasonCode:      "erc20_trusted_protocol_token_hub",
			wantAttribution:     model.LabelCategoryProtocol,
			wantInsightType:     model.AttributionInsightClusterGrouping,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, stderr, code := runDatasetModeForTest(t, tt.path)
			if code != 0 {
				t.Fatalf("expected exit code 0, got %d; stderr=%s", code, stderr)
			}

			if profile.Network != tt.wantNetwork {
				t.Fatalf("expected network %q, got %q", tt.wantNetwork, profile.Network)
			}

			if profile.RiskScore <= 0 {
				t.Fatalf("expected positive risk score, got %v", profile.RiskScore)
			}

			if !strings.Contains(profile.ValidationDetails, tt.wantDetailSubstring) {
				t.Fatalf("expected validation details to contain %q, got %q", tt.wantDetailSubstring, profile.ValidationDetails)
			}

			if !hasRiskReasonCode(profile.RiskReasons, tt.wantReasonCode) {
				t.Fatalf("expected risk reason %q, got %+v", tt.wantReasonCode, profile.RiskReasons)
			}
			if tt.wantAttribution != "" {
				if profile.Attribution == nil {
					t.Fatalf("expected resolved attribution on profile")
				}
				if profile.Attribution.Category != tt.wantAttribution {
					t.Fatalf("expected attribution category %q, got %+v", tt.wantAttribution, profile.Attribution)
				}
			}
			if tt.wantInsightType != "" && !hasInsightType(profile.AttributionInsights, tt.wantInsightType) {
				t.Fatalf("expected attribution insight %q, got %+v", tt.wantInsightType, profile.AttributionInsights)
			}
		})
	}
}

func TestRun_DatasetModeRejectsMalformedJSON(t *testing.T) {
	path := writeValidatorDatasetFile(t, "{not-json")

	_, stderr, code := runDatasetModeForTest(t, path)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, stderr)
	}

	if !strings.Contains(stderr, "Error probing dataset") {
		t.Fatalf("expected probing error, got %q", stderr)
	}
}

func TestRun_DatasetModeRejectsMissingRequiredFields(t *testing.T) {
	path := writeValidatorDatasetFile(t, `{
	  "case_id": "",
	  "address": "",
	  "chain": "SOLANA",
	  "stablecoin_summary": {}
	}`)

	_, stderr, code := runDatasetModeForTest(t, path)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%s", code, stderr)
	}

	if !strings.Contains(stderr, "Error loading Solana curated dataset") {
		t.Fatalf("expected Solana loader error, got %q", stderr)
	}
}

func TestApplySolanaCuratedStablecoinContext_ThresholdsAndReasons(t *testing.T) {
	t.Run("broad mixed surface uses stronger rule", func(t *testing.T) {
		curated := &datasets.SolanaCuratedStablecoinCase{
			CaseID:         "solana-test",
			Address:        "solana-test-address",
			Chain:          "SOLANA",
			RiskPosture:    "TEST",
			SourceRowCount: 200000,
			StablecoinSummary: datasets.SolanaStablecoinSummary{
				DominantRole:           "authority",
				DominantMint:           "mint-a",
				AuthorityTransferCount: 15000,
				UniqueCounterparties:   6000,
			},
			MintBreakdown: []datasets.SolanaMintCount{
				{Mint: "mint-a", Count: 12000},
				{Mint: "mint-b", Count: 3000},
			},
			TopCounterparties: []datasets.SolanaStablecoinCounterparty{
				{Address: "cp-1", Interactions: 12000},
			},
			CurationNotes: datasets.SolanaCurationNotes{
				Narrative: "Synthetic Solana authority-heavy broad surface case.",
			},
		}

		profile := buildWalletProfileFromSolanaCuratedStablecoinCase(curated)
		applySolanaCuratedStablecoinContext(profile, curated)

		assertHasRiskReasonCode(t, profile.RiskReasons, "solana_authority_heavy_stablecoin_operator")
		assertHasRiskReasonCode(t, profile.RiskReasons, "solana_broad_mixed_stablecoin_surface")
		assertHasRiskReasonCode(t, profile.RiskReasons, "solana_mixed_stablecoin_activity")
		assertHasRiskReasonCode(t, profile.RiskReasons, "solana_repeated_large_counterparty_interaction")

		if hasRiskReasonCode(profile.RiskReasons, "solana_broad_stablecoin_counterparty_surface") {
			t.Fatalf("did not expect generic broad surface reason when stronger mixed-surface rule applies")
		}
	})

	t.Run("below-threshold case stays minimal", func(t *testing.T) {
		curated := &datasets.SolanaCuratedStablecoinCase{
			CaseID:         "solana-below-threshold",
			Address:        "solana-threshold-address",
			Chain:          "SOLANA",
			RiskPosture:    "TEST",
			SourceRowCount: 10,
			StablecoinSummary: datasets.SolanaStablecoinSummary{
				DominantRole:         "source",
				SourceTransferCount:  99999,
				SourceCounterparties: 499,
				UniqueCounterparties: 999,
			},
			MintBreakdown: []datasets.SolanaMintCount{
				{Mint: "mint-a", Count: 99999},
			},
		}

		profile := buildWalletProfileFromSolanaCuratedStablecoinCase(curated)
		applySolanaCuratedStablecoinContext(profile, curated)

		if len(profile.RiskReasons) != 0 {
			t.Fatalf("expected no risk reasons below thresholds, got %+v", profile.RiskReasons)
		}
		if profile.RiskScore != 0 {
			t.Fatalf("expected zero risk score below thresholds, got %v", profile.RiskScore)
		}
		if profile.ReviewRecommended {
			t.Fatalf("expected review_recommended=false below thresholds")
		}
	})
}

func TestApplyBitcoinCuratedLayer1Context_UsesSpecificLegacyRule(t *testing.T) {
	curated := &datasets.BitcoinCuratedLayer1Case{
		CaseID:         "bitcoin-legacy-test",
		Address:        "1legacybtcaddress111111111111111111111",
		Chain:          "BITCOIN",
		RiskPosture:    "TEST",
		SourceRowCount: 300000,
		UTXOSummary: datasets.BitcoinUTXOSummary{
			DominantRole:         "outbound",
			InboundReceiptCount:  125000,
			OutboundSpendCount:   140000,
			UniqueCounterparties: 65000,
		},
		TopCounterparties: []datasets.BitcoinCounterparty{
			{Address: "bc1qtopcounterparty", Interactions: 125000},
		},
		CurationNotes: datasets.BitcoinCurationNotes{
			Narrative: "Synthetic legacy mixed-flow regression case.",
		},
	}

	profile := buildWalletProfileFromBitcoinCuratedLayer1Case(curated)
	applyBitcoinCuratedLayer1Context(profile, curated)

	assertHasRiskReasonCode(t, profile.RiskReasons, "bitcoin_legacy_mixed_flow_broad_value")
	assertHasRiskReasonCode(t, profile.RiskReasons, "bitcoin_extremely_broad_counterparty_surface")
	assertHasRiskReasonCode(t, profile.RiskReasons, "bitcoin_extreme_repeated_counterparty_interaction")

	if hasRiskReasonCode(profile.RiskReasons, "bitcoin_spend_heavy_operational_hub") {
		t.Fatalf("did not expect generic spend-heavy rule when legacy mixed-flow rule applies")
	}
}

func TestApplyERC20CuratedLayer1Context_ThresholdsAndReasons(t *testing.T) {
	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", repoPath("data", "labels", "bootstrap_entities.json"))
	setEnvForTest(t, "GRAPHSENSE_LABELS_PATH", repoPath("data", "labels", "tier1_graphsense_entities.json"))
	setEnvForTest(t, "BITCOIN_MINING_POOLS_PATH", repoPath("data", "labels", "tier1_bitcoin_mining_pools.json"))
	setEnvForTest(t, "WALLET_EXPLORER_LABELS_PATH", repoPath("data", "labels", "tier2_wallet_explorer_entities.json"))
	setEnvForTest(t, "CORROBORATING_LABELS_PATH", repoPath("data", "labels", "tier2_corroborating_entities.json"))

	t.Run("exchange-like case avoids generic inbound rule", func(t *testing.T) {
		curated := &datasets.ERC20CuratedLayer1Case{
			CaseID:         "erc20-exchange-test",
			Address:        "0x28c6c06298d514db089934071355e5743bf21d60",
			Chain:          "EVM",
			Label:          "EXCHANGE-LIKE TOKEN SERVICE SURFACE",
			RiskPosture:    "TEST",
			SourceRowCount: 200000,
			ERC20Summary: datasets.ERC20Layer1Summary{
				InboundTransferCount:        15000,
				OutboundTransferCount:       9000,
				UniqueCounterparties:        3500,
				UniqueTokenContracts:        80,
				RepeatedCounterparties:      45,
				MaxCounterpartyInteractions: 180,
				DominantDirection:           "inbound",
				DominantTokenSymbol:         "USDC",
				DominantTokenTransferShare:  35.5,
			},
			TopCounterparties: []datasets.ERC20CounterpartySummary{
				{Address: "0xcp1", Interactions: 180},
			},
		}

		profile := buildWalletProfileFromERC20CuratedLayer1Case(curated)
		applyERC20CuratedLayer1Context(profile, curated)

		assertHasRiskReasonCode(t, profile.RiskReasons, "erc20_exchange_service_surface")
		assertHasRiskReasonCode(t, profile.RiskReasons, "erc20_broad_token_counterparty_surface")
		assertHasRiskReasonCode(t, profile.RiskReasons, "erc20_mixed_token_activity")
		assertHasRiskReasonCode(t, profile.RiskReasons, "erc20_repeated_counterparty_activity")

		if hasRiskReasonCode(profile.RiskReasons, "erc20_noisy_token_inbound_surface") {
			t.Fatalf("did not expect generic noisy inbound rule for exchange-like case")
		}
	})

	t.Run("unlabeled inbound-heavy case gets noisy inbound rule", func(t *testing.T) {
		curated := &datasets.ERC20CuratedLayer1Case{
			CaseID:         "erc20-noisy-inbound-test",
			Address:        "0x1111111111111111111111111111111111111111",
			Chain:          "EVM",
			Label:          "UNLABELED TOKEN SURFACE",
			RiskPosture:    "TEST",
			SourceRowCount: 9000,
			ERC20Summary: datasets.ERC20Layer1Summary{
				InboundTransferCount:        7000,
				OutboundTransferCount:       500,
				UniqueCounterparties:        1200,
				UniqueTokenContracts:        40,
				RepeatedCounterparties:      22,
				MaxCounterpartyInteractions: 120,
				DominantDirection:           "inbound",
				DominantTokenSymbol:         "USDT",
				DominantTokenTransferShare:  82.0,
			},
			TopCounterparties: []datasets.ERC20CounterpartySummary{
				{Address: "0xcp2", Interactions: 120},
			},
		}

		profile := buildWalletProfileFromERC20CuratedLayer1Case(curated)
		applyERC20CuratedLayer1Context(profile, curated)

		assertHasRiskReasonCode(t, profile.RiskReasons, "erc20_noisy_token_inbound_surface")
		assertHasRiskReasonCode(t, profile.RiskReasons, "erc20_broad_token_counterparty_surface")
		assertHasRiskReasonCode(t, profile.RiskReasons, "erc20_repeated_counterparty_activity")
		assertHasRiskReasonCode(t, profile.RiskReasons, "erc20_single_token_operational_concentration")
	})
}

func runDatasetModeForTest(t *testing.T, datasetPath string) (*model.WalletProfile, string, int) {
	t.Helper()

	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run([]string{"--dataset", datasetPath}, &out, &errOut, nil)

	var profile model.WalletProfile
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &profile); err != nil {
			t.Fatalf("failed to decode stdout json: %v\nstdout=%s", err, out.String())
		}
	}

	return &profile, errOut.String(), code
}

func hasInsightType(insights []model.AttributionInsight, want model.AttributionInsightType) bool {
	for _, insight := range insights {
		if insight.Type == want {
			return true
		}
	}
	return false
}

func repoPath(parts ...string) string {
	segments := append([]string{"..", ".."}, parts...)
	return filepath.Join(segments...)
}

func writeValidatorDatasetFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "dataset.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write dataset file: %v", err)
	}

	return path
}

func hasRiskReasonCode(reasons []model.RiskReason, code string) bool {
	for _, reason := range reasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}

func assertHasRiskReasonCode(t *testing.T, reasons []model.RiskReason, code string) {
	t.Helper()

	if !hasRiskReasonCode(reasons, code) {
		t.Fatalf("expected risk reason %q, got %+v", code, reasons)
	}
}
