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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
)

func main() {
	inDir := flag.String("in", "", "Input curated cases directory")
	traceDir := flag.String("trace", "", "Extracted trace summaries directory")
	outDir := flag.String("out", "", "Output directory for enriched curated cases")
	flag.Parse()

	if strings.TrimSpace(*inDir) == "" || strings.TrimSpace(*traceDir) == "" || strings.TrimSpace(*outDir) == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/enrichcases --in ./data/cases/curated --trace ./data/cases/extracted-traces --out ./data/cases/curated-enriched")
		os.Exit(1)
	}

	entries, err := filepath.Glob(filepath.Join(*inDir, "*.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: list curated files: %v\n", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "error: no curated case json files found in %s\n", *inDir)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: create output dir: %v\n", err)
		os.Exit(1)
	}

	for _, path := range entries {
		cc, err := datasets.LoadCuratedCase(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: load curated case %s: %v\n", path, err)
			os.Exit(1)
		}

		traceMeta, err := datasets.LoadExtractedTraceDatasetByAddress(*traceDir, cc.Address)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: load trace summary for %s: %v\n", cc.Address, err)
			os.Exit(1)
		}

		if traceMeta != nil {
			cc.TraceSummary = &traceMeta.Summary
			cc.TraceTopCounterparties = traceMeta.TopCounterparties
			cc.TraceSourceCount = traceMeta.SourceTraceCount
			cc.TraceRawFile = traceMeta.RawTraceFile
		}

		outPath := filepath.Join(*outDir, filepath.Base(path))
		if err := datasets.WriteCuratedCase(outPath, cc); err != nil {
			fmt.Fprintf(os.Stderr, "error: write enriched case %s: %v\n", outPath, err)
			os.Exit(1)
		}

		if traceMeta != nil {
			fmt.Printf("wrote %s (trace_source_count=%d, trace_counterparties=%d)\n", outPath, traceMeta.SourceTraceCount, len(traceMeta.TopCounterparties))
		} else {
			fmt.Printf("wrote %s (no trace summary)\n", outPath)
		}
	}
}
