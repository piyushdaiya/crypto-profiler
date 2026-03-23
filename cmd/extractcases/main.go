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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
)

func main() {
	var (
		ethInput      = flag.String("eth", "", "Path to Ethereum TSV/TSV.GZ file or directory")
		erc20Input    = flag.String("erc20", "", "Path to ERC-20 TSV/TSV.GZ file or directory")
		labelsPath    = flag.String("labels", "data/labels/legacy_labels.json", "Path to labels JSON")
		addressesPath = flag.String("addresses", "data/candidates/evm_addresses.example.txt", "Path to target address list")
		outDir        = flag.String("out", "data/cases/extracted", "Output directory")
	)
	flag.Parse()

	addresses, err := datasets.LoadAddressList(*addressesPath)
	if err != nil {
		fail("load addresses", err)
	}

	labels := map[string]string{}
	if *labelsPath != "" {
		labels, err = datasets.LoadLegacyLabels(*labelsPath)
		if err != nil {
			fail("load labels", err)
		}
	}

	ethFiles, err := collectFiles(*ethInput, "blockchair_ethereum_transactions_")
	if err != nil {
		fail("collect ethereum files", err)
	}

	erc20Files, err := collectFiles(*erc20Input, "blockchair_erc-20_transactions_")
	if err != nil {
		fail("collect erc20 files", err)
	}

	results, err := datasets.ExtractEVMCaseDatasets(ethFiles, erc20Files, addresses, labels)
	if err != nil {
		fail("extract datasets", err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail("create output directory", err)
	}

	for _, addr := range addresses {
		ds, ok := results[addr]
		if !ok {
			continue
		}

		filename := filepath.Join(*outDir, sanitizeFilename(addr)+".json")
		raw, err := json.MarshalIndent(ds, "", "  ")
		if err != nil {
			fail("marshal dataset", err)
		}

		if err := os.WriteFile(filename, raw, 0o644); err != nil {
			fail("write dataset", err)
		}

		fmt.Printf("wrote %s (%d transfers)\n", filename, len(ds.Transfers))
	}
}

func collectFiles(input string, prefix string) ([]string, error) {
	if input == "" {
		return nil, nil
	}

	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []string{input}, nil
	}

	entries, err := os.ReadDir(input)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if !(strings.HasSuffix(name, ".tsv") || strings.HasSuffix(name, ".tsv.gz")) {
			continue
		}

		out = append(out, filepath.Join(input, name))
	}

	sort.Strings(out)
	return out, nil
}

func sanitizeFilename(addr string) string {
	addr = datasets.NormalizeHexAddress(addr)
	if len(addr) > 10 {
		return addr[2:]
	}
	return addr
}

func fail(step string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", step, err)
	os.Exit(1)
}
