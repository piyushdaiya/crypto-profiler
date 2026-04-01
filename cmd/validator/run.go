package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/address"
	"github.com/piyushdaiya/crypto-profiler/internal/analyzer"
	"github.com/piyushdaiya/crypto-profiler/internal/attribution"
	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type datasetProbe struct {
	Chain      string `json:"chain"`
	CaseFamily string `json:"case_family"`
}

func defaultStrategies() []address.ChainStrategy {
	return []address.ChainStrategy{
		&address.EVMStrategy{},
		&address.BitcoinStrategy{},
		&address.SolanaStrategy{},
	}
}

func run(args []string, out io.Writer, errOut io.Writer, strategies []address.ChainStrategy) int {
	fs := flag.NewFlagSet("validator", flag.ContinueOnError)
	fs.SetOutput(errOut)

	datasetPath := fs.String("dataset", "", "Path to curated dataset JSON")
	reportMode := fs.Bool("report", false, "Render a human-readable analyst report instead of JSON")
	fs.Usage = func() {
		fmt.Fprintln(errOut, "Usage:")
		fmt.Fprintln(errOut, "  ./validator [--report] <wallet-address>")
		fmt.Fprintln(errOut, "  ./validator --dataset <curated-case.json> [--report]")
		fmt.Fprintln(errOut, "")
		fmt.Fprintln(errOut, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(errOut, "")
		fmt.Fprintln(errOut, "Examples:")
		fmt.Fprintln(errOut, "  go run ./cmd/validator 0xd90e2f925da726b50c4ed8d0fb90ad053324f31b")
		fmt.Fprintln(errOut, "  go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json")
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if *datasetPath != "" {
		return runDatasetMode(*datasetPath, *reportMode, out, errOut)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return 1
	}

	walletAddress := strings.TrimSpace(fs.Arg(0))

	etherscanKey := os.Getenv("ETHERSCAN_API_KEY")
	coinstatsKey := os.Getenv("COINSTATS_API_KEY")

	var result *model.WalletProfile

	for _, strategy := range strategies {
		if !strategy.IsValidSyntax(walletAddress) {
			continue
		}

		configParam := ""
		switch strategy.Name() {
		case "EVM (Etherscan)":
			configParam = etherscanKey
		case "SOLANA":
			configParam = coinstatsKey
		case "BITCOIN":
			configParam = ""
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

		fmt.Fprintf(errOut, "🔍 Analyzing %s on %s...\n", walletAddress, strategy.Name())

		res, err := strategy.FetchState(ctx, walletAddress, configParam)
		cancel()

		if err != nil {
			fmt.Fprintf(errOut, "⚠️ Error validating: %v\n", err)
		}

		if res != nil && res.RiskScore == 0 && len(res.RiskReasons) == 0 {
			analyzer.Investigate(res, nil)
		}

		result = res
		break
	}

	if result == nil {
		result = &model.WalletProfile{
			Address:           walletAddress,
			Network:           "UNKNOWN",
			IsValid:           false,
			ValidationDetails: "Invalid format or no matching chain strategy",
		}
	}

	attribution.ApplyTier1Attribution(result)
	liveWave5CInput := attribution.Wave5CInput{Network: result.Network}
	attribution.ApplyWave5CContext(result, liveWave5CInput)
	applyGraphSummary(result, liveWave5CInput)

	return writeOutput(result, buildLiveReportContext(result), *reportMode, out, errOut)
}

func runDatasetMode(path string, reportMode bool, out io.Writer, errOut io.Writer) int {
	fmt.Fprintf(errOut, "🔍 Analyzing curated dataset %s...\n", path)

	profile, reportContext, err := loadDatasetMode(path)
	if err != nil {
		fmt.Fprintf(errOut, "%v\n", err)
		return 1
	}

	return writeOutput(profile, reportContext, reportMode, out, errOut)
}

func loadDatasetMode(path string) (*model.WalletProfile, *reportContext, error) {
	// #nosec G304 -- validator dataset mode intentionally reads an explicit local curated-case path provided by the operator.
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, nil, fmt.Errorf("Error loading dataset: %v", err)
	}

	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, nil, fmt.Errorf("Error probing dataset: %v", err)
	}

	var meta datasetProbe
	_ = json.Unmarshal(raw, &meta)

	_, hasStablecoinSummary := probe["stablecoin_summary"]
	_, hasUTXOSummary := probe["utxo_summary"]
	_, hasERC20Summary := probe["erc20_summary"]

	if strings.EqualFold(meta.Chain, "SOLANA") && hasStablecoinSummary {
		cc, err := datasets.LoadSolanaCuratedStablecoinCase(path)
		if err != nil {
			return nil, nil, fmt.Errorf("Error loading Solana curated dataset: %v", err)
		}

		profile := buildWalletProfileFromSolanaCuratedStablecoinCase(cc)
		applySolanaCuratedStablecoinContext(profile, cc)
		attribution.ApplyTier1Attribution(profile)

		input := buildWave5CInputFromSolanaCase(cc)
		attribution.ApplyWave5CContext(profile, input)
		applyGraphSummary(profile, input)

		return profile, buildReportContextFromSolanaCase(cc), nil
	}

	if strings.EqualFold(meta.Chain, "BITCOIN") && hasUTXOSummary {
		cc, err := datasets.LoadBitcoinCuratedLayer1Case(path)
		if err != nil {
			return nil, nil, fmt.Errorf("Error loading Bitcoin curated dataset: %v", err)
		}

		profile := buildWalletProfileFromBitcoinCuratedLayer1Case(cc)
		applyBitcoinCuratedLayer1Context(profile, cc)
		attribution.ApplyTier1Attribution(profile)

		input := buildWave5CInputFromBitcoinCase(cc)
		attribution.ApplyWave5CContext(profile, input)
		applyGraphSummary(profile, input)

		return profile, buildReportContextFromBitcoinCase(cc), nil
	}

	if strings.EqualFold(meta.Chain, "EVM") && hasERC20Summary {
		cc, err := datasets.LoadERC20CuratedLayer1Case(path)
		if err != nil {
			return nil, nil, fmt.Errorf("Error loading ERC-20 curated dataset: %v", err)
		}

		profile := buildWalletProfileFromERC20CuratedLayer1Case(cc)
		applyERC20CuratedLayer1Context(profile, cc)
		attribution.ApplyTier1Attribution(profile)

		input := buildWave5CInputFromERC20Case(cc)
		attribution.ApplyWave5CContext(profile, input)
		applyGraphSummary(profile, input)

		return profile, buildReportContextFromERC20Case(cc), nil
	}

	// Cross-chain must be checked before chain-specific L2 loaders.
	if strings.EqualFold(meta.CaseFamily, "crosschain_l2") {
		var cc crosschainL2Case
		if err := json.Unmarshal(raw, &cc); err != nil {
			return nil, nil, fmt.Errorf("Error loading cross-chain L2 curated dataset: %v", err)
		}

		profile := buildWalletProfileFromCrosschainL2Case(&cc)
		applyCrosschainL2Context(profile, &cc)
		attribution.ApplyTier1Attribution(profile)

		return profile, buildReportContextFromCrosschainL2Case(&cc), nil
	}

	if strings.EqualFold(meta.Chain, "ARBITRUM") {
		var cc arbitrumCuratedLayer2Case
		if err := json.Unmarshal(raw, &cc); err != nil {
			return nil, nil, fmt.Errorf("Error loading Arbitrum curated dataset: %v", err)
		}

		profile := buildWalletProfileFromArbitrumCuratedLayer2Case(&cc)
		applyArbitrumCuratedLayer2Context(profile, &cc)
		attribution.ApplyTier1Attribution(profile)

		return profile, buildReportContextFromArbitrumCase(&cc), nil
	}

	if strings.EqualFold(meta.Chain, "OPTIMISM") {
		var cc optimismCuratedLayer2Case
		if err := json.Unmarshal(raw, &cc); err != nil {
			return nil, nil, fmt.Errorf("Error loading Optimism curated dataset: %v", err)
		}

		profile := buildWalletProfileFromOptimismCuratedLayer2Case(&cc)
		applyOptimismCuratedLayer2Context(profile, &cc)
		attribution.ApplyTier1Attribution(profile)

		return profile, buildReportContextFromOptimismCase(&cc), nil
	}

	if strings.EqualFold(meta.Chain, "POLYGON") {
		var cc polygonCuratedLayer2Case
		if err := json.Unmarshal(raw, &cc); err != nil {
			return nil, nil, fmt.Errorf("Error loading Polygon curated dataset: %v", err)
		}

		profile := buildWalletProfileFromPolygonCuratedLayer2Case(&cc)
		applyPolygonCuratedLayer2Context(profile, &cc)
		attribution.ApplyTier1Attribution(profile)

		return profile, buildReportContextFromPolygonCase(&cc), nil
	}

	cc, err := datasets.LoadCuratedCase(path)
	if err != nil {
		return nil, nil, fmt.Errorf("Error loading dataset: %v", err)
	}

	profile := datasets.BuildWalletProfileFromCuratedCase(cc)
	txs := datasets.BuildTransactionsFromCuratedCase(cc)

	analyzer.Investigate(profile, txs)
	applyCuratedTraceContext(profile, cc)
	attribution.ApplyTier1Attribution(profile)

	input := buildWave5CInputFromCuratedCase(cc)
	attribution.ApplyWave5CContext(profile, input)
	applyGraphSummary(profile, input)

	return profile, buildReportContextFromCuratedCase(cc), nil
}

func writeOutput(result *model.WalletProfile, context *reportContext, reportMode bool, out io.Writer, errOut io.Writer) int {
	if reportMode {
		if _, err := io.WriteString(out, renderReport(result, context)); err != nil {
			fmt.Fprintf(errOut, "Error rendering report: %v\n", err)
			return 1
		}
		return 0
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(errOut, "Error encoding JSON: %v\n", err)
		return 1
	}

	return 0
}

func applyGraphSummary(profile *model.WalletProfile, input attribution.Wave5CInput) {
	if profile == nil {
		return
	}

	profile.GraphSummary = attribution.BuildGraphSummaryFromWave5CInput(
		profile.Address,
		input,
		func(address string) *model.ResolvedAttribution {
			resolved, ok := attribution.ResolveAddress(address, profile.Network)
			if !ok {
				return nil
			}
			return resolved
		},
	)

	attribution.ApplyGraphSummaryContext(profile)
}
