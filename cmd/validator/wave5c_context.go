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
	"strings"
	"time"

	"github.com/piyushdaiya/crypto-profiler/internal/attribution"
	"github.com/piyushdaiya/crypto-profiler/internal/datasets"
)

func buildWave5CInputFromCuratedCase(cc *datasets.CuratedCase) attribution.Wave5CInput {
	if cc == nil {
		return attribution.Wave5CInput{}
	}

	counterparties := make([]attribution.InteractionCounterparty, 0, len(cc.TopCounterparties)+len(cc.TraceTopCounterparties))
	for _, cp := range cc.TopCounterparties {
		counterparties = append(counterparties, attribution.InteractionCounterparty{
			Address:       cp.Address,
			Label:         cp.Label,
			Interactions:  cp.Interactions,
			InboundCount:  cp.InboundCount,
			OutboundCount: cp.OutboundCount,
		})
	}
	for _, cp := range cc.TraceTopCounterparties {
		counterparties = append(counterparties, attribution.InteractionCounterparty{
			Address:       cp.Address,
			Interactions:  cp.Interactions,
			InboundCount:  cp.InboundCount,
			OutboundCount: cp.OutboundCount,
		})
	}

	flows := make([]attribution.FlowObservation, 0, len(cc.SampleTransfers))
	for _, tr := range cc.SampleTransfers {
		direction := ""
		counterparty := ""
		counterpartyLabel := ""
		switch {
		case strings.EqualFold(tr.To, cc.Address):
			direction = "inbound"
			counterparty = tr.From
			counterpartyLabel = tr.LabelFrom
		case strings.EqualFold(tr.From, cc.Address):
			direction = "outbound"
			counterparty = tr.To
			counterpartyLabel = tr.LabelTo
		default:
			continue
		}

		flows = append(flows, attribution.FlowObservation{
			Timestamp:         tr.Timestamp,
			Direction:         direction,
			Counterparty:      counterparty,
			CounterpartyLabel: counterpartyLabel,
		})
	}

	return attribution.Wave5CInput{
		Network:        cc.Chain,
		Counterparties: counterparties,
		Flows:          flows,
	}
}

func buildWave5CInputFromERC20Case(cc *datasets.ERC20CuratedLayer1Case) attribution.Wave5CInput {
	if cc == nil {
		return attribution.Wave5CInput{}
	}

	counterparties := make([]attribution.InteractionCounterparty, 0, len(cc.TopCounterparties))
	for _, cp := range cc.TopCounterparties {
		counterparties = append(counterparties, attribution.InteractionCounterparty{
			Address:       cp.Address,
			Label:         cp.Label,
			Interactions:  cp.Interactions,
			InboundCount:  cp.InboundCount,
			OutboundCount: cp.OutboundCount,
		})
	}

	flows := make([]attribution.FlowObservation, 0, len(cc.SampleTransfers))
	for _, tr := range cc.SampleTransfers {
		flows = append(flows, attribution.FlowObservation{
			Timestamp:         parseWave5CTime(tr.BlockTimestamp),
			Direction:         tr.Direction,
			Counterparty:      tr.Counterparty,
			CounterpartyLabel: firstNonEmpty(tr.LabelSender, tr.LabelRecipient),
		})
	}

	return attribution.Wave5CInput{
		Network:        cc.Chain,
		Counterparties: counterparties,
		Flows:          flows,
	}
}

func buildWave5CInputFromBitcoinCase(cc *datasets.BitcoinCuratedLayer1Case) attribution.Wave5CInput {
	if cc == nil {
		return attribution.Wave5CInput{}
	}

	counterparties := make([]attribution.InteractionCounterparty, 0, len(cc.TopCounterparties))
	for _, cp := range cc.TopCounterparties {
		counterparties = append(counterparties, attribution.InteractionCounterparty{
			Address:       cp.Address,
			Interactions:  cp.Interactions,
			InboundCount:  cp.InboundCount,
			OutboundCount: cp.OutboundCount,
		})
	}

	flows := make([]attribution.FlowObservation, 0, len(cc.SampleEvents))
	for _, event := range cc.SampleEvents {
		for _, cp := range event.Counterparties {
			flows = append(flows, attribution.FlowObservation{
				Timestamp:    parseWave5CTime(event.EventTime),
				Direction:    event.Direction,
				Counterparty: cp,
			})
		}
	}

	return attribution.Wave5CInput{
		Network:        cc.Chain,
		Counterparties: counterparties,
		Flows:          flows,
	}
}

func buildWave5CInputFromSolanaCase(cc *datasets.SolanaCuratedStablecoinCase) attribution.Wave5CInput {
	if cc == nil {
		return attribution.Wave5CInput{}
	}

	counterparties := make([]attribution.InteractionCounterparty, 0, len(cc.TopCounterparties))
	for _, cp := range cc.TopCounterparties {
		counterparties = append(counterparties, attribution.InteractionCounterparty{
			Address:       cp.Address,
			Interactions:  cp.Interactions,
			InboundCount:  cp.InboundCount,
			OutboundCount: cp.OutboundCount,
		})
	}

	flows := make([]attribution.FlowObservation, 0, len(cc.SampleTransfers))
	for _, tr := range cc.SampleTransfers {
		counterparty := firstNonEmpty(tr.Destination, tr.Source)
		if strings.EqualFold(counterparty, cc.Address) {
			counterparty = tr.Authority
		}
		if strings.EqualFold(counterparty, cc.Address) || strings.TrimSpace(counterparty) == "" {
			continue
		}
		direction := tr.TransferType
		if strings.EqualFold(tr.Source, cc.Address) {
			direction = "outbound"
		} else if strings.EqualFold(tr.Destination, cc.Address) {
			direction = "inbound"
		} else if strings.EqualFold(tr.Authority, cc.Address) {
			direction = "authority"
		}

		flows = append(flows, attribution.FlowObservation{
			Timestamp:    parseWave5CTime(tr.BlockTimestamp),
			Direction:    direction,
			Counterparty: counterparty,
		})
	}

	return attribution.Wave5CInput{
		Network:        cc.Chain,
		Counterparties: counterparties,
		Flows:          flows,
	}
}

func parseWave5CTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC()
		}
	}
	return time.Time{}
}
