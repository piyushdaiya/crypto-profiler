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
	"fmt"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func applyCuratedTraceContext(profile *model.WalletProfile, curated *datasets.CuratedCase) {
	if profile == nil || curated == nil || curated.TraceSummary == nil {
		return
	}

	ts := curated.TraceSummary

	detailParts := []string{
		fmt.Sprintf("Trace Summary"),
		fmt.Sprintf("Trace Count: %d", curated.TraceSourceCount),
		fmt.Sprintf("Inbound Traces: %d", ts.InboundTraceCount),
		fmt.Sprintf("Outbound Traces: %d", ts.OutboundTraceCount),
		fmt.Sprintf("Failed Traces: %d", ts.FailedTraceCount),
		fmt.Sprintf("Value-Bearing Traces: %d", ts.ValueTransferTraceCount),
		fmt.Sprintf("Unique Trace Counterparties: %d", ts.UniqueCounterparties),
		fmt.Sprintf("Max Trace Depth: %d", ts.MaxDepth),
	}

	if profile.ValidationDetails != "" {
		profile.ValidationDetails += " | "
	}
	profile.ValidationDetails += strings.Join(detailParts, " | ")

	appendReason := func(code, category, description string, evidenceCount int) {
		profile.RiskReasons = append(profile.RiskReasons, model.RiskReason{
			Code:          code,
			Category:      category,
			Description:   description,
			Offset:        0.0,
			Source:        "dataset_trace_summary",
			EvidenceCount: evidenceCount,
		})
	}

	if curated.TraceSourceCount > 0 {
		appendReason(
			"dataset_trace_activity_observed",
			"REPUTATION",
			fmt.Sprintf("Trace-aware dataset context loaded (%d internal traces observed)", curated.TraceSourceCount),
			curated.TraceSourceCount,
		)
	}

	if ts.FailedTraceCount > 0 {
		appendReason(
			"dataset_trace_failed_calls_observed",
			"REPUTATION",
			fmt.Sprintf("Observed %d failed internal calls in trace dataset", ts.FailedTraceCount),
			ts.FailedTraceCount,
		)
	}

	if ts.MaxDepth >= 8 {
		appendReason(
			"dataset_trace_deep_routing_observed",
			"REPUTATION",
			fmt.Sprintf("Observed deep internal call routing (max depth %d)", ts.MaxDepth),
			ts.MaxDepth,
		)
	}

	if ts.UniqueCounterparties >= 1000 {
		appendReason(
			"dataset_trace_broad_counterparty_surface",
			"REPUTATION",
			fmt.Sprintf("Trace dataset shows broad internal counterparty surface (%d counterparties)", ts.UniqueCounterparties),
			ts.UniqueCounterparties,
		)
	}
}
