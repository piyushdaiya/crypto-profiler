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
	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

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

	var chain string
	if chainRaw, ok := probe["chain"]; ok {
		_ = json.Unmarshal(chainRaw, &chain)
	}

	_, hasStablecoinSummary := probe["stablecoin_summary"]
	_, hasUTXOSummary := probe["utxo_summary"]
	_, hasERC20Summary := probe["erc20_summary"]

	if strings.EqualFold(chain, "SOLANA") && hasStablecoinSummary {
		cc, err := datasets.LoadSolanaCuratedStablecoinCase(path)
		if err != nil {
			return nil, nil, fmt.Errorf("Error loading Solana curated dataset: %v", err)
		}

		profile := buildWalletProfileFromSolanaCuratedStablecoinCase(cc)
		applySolanaCuratedStablecoinContext(profile, cc)
		return profile, buildReportContextFromSolanaCase(cc), nil
	}

	if strings.EqualFold(chain, "BITCOIN") && hasUTXOSummary {
		cc, err := datasets.LoadBitcoinCuratedLayer1Case(path)
		if err != nil {
			return nil, nil, fmt.Errorf("Error loading Bitcoin curated dataset: %v", err)
		}

		profile := buildWalletProfileFromBitcoinCuratedLayer1Case(cc)
		applyBitcoinCuratedLayer1Context(profile, cc)
		return profile, buildReportContextFromBitcoinCase(cc), nil
	}

	if strings.EqualFold(chain, "EVM") && hasERC20Summary {
		cc, err := datasets.LoadERC20CuratedLayer1Case(path)
		if err != nil {
			return nil, nil, fmt.Errorf("Error loading ERC-20 curated dataset: %v", err)
		}

		profile := buildWalletProfileFromERC20CuratedLayer1Case(cc)
		applyERC20CuratedLayer1Context(profile, cc)
		return profile, buildReportContextFromERC20Case(cc), nil
	}

	cc, err := datasets.LoadCuratedCase(path)
	if err != nil {
		return nil, nil, fmt.Errorf("Error loading dataset: %v", err)
	}

	profile := datasets.BuildWalletProfileFromCuratedCase(cc)
	txs := datasets.BuildTransactionsFromCuratedCase(cc)

	analyzer.Investigate(profile, txs)
	applyCuratedTraceContext(profile, cc)

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
