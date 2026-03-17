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
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
)

func main() {
	var (
		inDir      = flag.String("in", "data/cases/extracted", "Directory containing extracted address datasets")
		outDir     = flag.String("out", "data/cases/curated", "Directory for curated case files")
		manifest   = flag.String("manifest", "data/cases/manifest.json", "Case manifest JSON")
		topN       = flag.Int("top", 20, "Top counterparties to include")
		sampleSize = flag.Int("sample", 50, "Number of sample transfers to include")
	)
	flag.Parse()

	caseManifest, err := datasets.LoadCaseManifest(*manifest)
	if err != nil {
		fail("load case manifest", err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail("create output dir", err)
	}

	for address, meta := range caseManifest {
		inputFile := filepath.Join(*inDir, extractedFilename(address))
		ds, err := datasets.LoadAddressDataset(inputFile)
		if err != nil {
			fail(fmt.Sprintf("load extracted dataset for %s", address), err)
		}

		curated := datasets.CurateDataset(ds, meta, *topN, *sampleSize)
		outFile := filepath.Join(*outDir, outputFilename(meta, address))

		raw, err := json.MarshalIndent(curated, "", "  ")
		if err != nil {
			fail("marshal curated case", err)
		}

		if err := os.WriteFile(outFile, raw, 0o644); err != nil {
			fail("write curated case", err)
		}

		fmt.Printf("wrote %s (sample=%d, counterparties=%d, source_transfers=%d)\n",
			outFile,
			len(curated.SampleTransfers),
			len(curated.TopCounterparties),
			curated.SourceTransferCount,
		)
	}
}

func extractedFilename(address string) string {
	addr := datasets.NormalizeHexAddress(address)
	if strings.HasPrefix(addr, "0x") {
		return addr[2:] + ".json"
	}
	return addr + ".json"
}

func outputFilename(meta datasets.CaseMetadata, address string) string {
	if meta.CaseID != "" {
		return meta.CaseID + ".json"
	}
	addr := datasets.NormalizeHexAddress(address)
	return strings.TrimPrefix(addr, "0x") + ".json"
}

func fail(step string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", step, err)
	os.Exit(1)
}
