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
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *datasetPath != "" {
		return runDatasetMode(*datasetPath, out, errOut)
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(errOut, "Usage: ./validator <wallet-address>")
		fmt.Fprintln(errOut, "   or: ./validator --dataset <curated-case.json>")
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

	return writeProfile(result, out, errOut)
}

func runDatasetMode(path string, out io.Writer, errOut io.Writer) int {
	fmt.Fprintf(errOut, "🔍 Analyzing curated dataset %s...\n", path)

	cc, err := datasets.LoadCuratedCase(path)
	if err != nil {
		fmt.Fprintf(errOut, "Error loading dataset: %v\n", err)
		return 1
	}

	profile := datasets.BuildWalletProfileFromCuratedCase(cc)
	txs := datasets.BuildTransactionsFromCuratedCase(cc)

	analyzer.Investigate(profile, txs)
	applyCuratedTraceContext(profile, cc)

	return writeProfile(profile, out, errOut)
}

func writeProfile(result *model.WalletProfile, out io.Writer, errOut io.Writer) int {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(errOut, "Error encoding JSON: %v\n", err)
		return 1
	}

	return 0
}
