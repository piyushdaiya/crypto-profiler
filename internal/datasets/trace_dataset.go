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

package datasets

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type TraceSummary struct {
	FirstSeen               string `json:"first_seen,omitempty"`
	LastSeen                string `json:"last_seen,omitempty"`
	InboundTraceCount       int    `json:"inbound_trace_count"`
	OutboundTraceCount      int    `json:"outbound_trace_count"`
	SelfTraceCount          int    `json:"self_trace_count"`
	FailedTraceCount        int    `json:"failed_trace_count"`
	ValueTransferTraceCount int    `json:"value_transfer_trace_count"`
	UniqueCounterparties    int    `json:"unique_counterparties"`
	MaxDepth                int    `json:"max_depth"`
}

type TraceCounterpartySummary struct {
	Address            string `json:"address"`
	Interactions       int    `json:"interactions"`
	InboundCount       int    `json:"inbound_count"`
	OutboundCount      int    `json:"outbound_count"`
	ValueTransferCount int    `json:"value_transfer_count"`
	FailedCount        int    `json:"failed_count"`
}

type ExtractedTraceDataset struct {
	Address           string                     `json:"address"`
	Chain             string                     `json:"chain"`
	GeneratedAt       string                     `json:"generated_at,omitempty"`
	Summary           TraceSummary               `json:"summary"`
	TopCounterparties []TraceCounterpartySummary `json:"top_counterparties,omitempty"`
	SourceTraceCount  int                        `json:"source_trace_count"`
	RawTraceFile      string                     `json:"raw_trace_file,omitempty"`
}

func LoadExtractedTraceDataset(path string) (*ExtractedTraceDataset, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var ds ExtractedTraceDataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, err
	}
	return &ds, nil
}

func LoadExtractedTraceDatasetByAddress(dir, address string) (*ExtractedTraceDataset, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, nil
	}

	name := normalizeTraceDatasetFilename(address)
	if name == "" {
		return nil, errors.New("invalid address for trace dataset lookup")
	}

	path := filepath.Join(dir, name+".json")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return LoadExtractedTraceDataset(path)
}

func normalizeTraceDatasetFilename(address string) string {
	s := strings.ToLower(strings.TrimSpace(address))
	s = strings.TrimPrefix(s, "0x")
	return s
}
