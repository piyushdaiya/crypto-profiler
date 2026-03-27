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
	"os"
	"path/filepath"
)

type BitcoinUTXOSummary struct {
	FirstSeen                  string `json:"first_seen,omitempty"`
	LastSeen                   string `json:"last_seen,omitempty"`
	InboundReceiptCount        int    `json:"inbound_receipt_count"`
	OutboundSpendCount         int    `json:"outbound_spend_count"`
	InboundValueSats           int64  `json:"inbound_value_sats"`
	OutboundValueSats          int64  `json:"outbound_value_sats"`
	InboundValueBTC            string `json:"inbound_value_btc,omitempty"`
	OutboundValueBTC           string `json:"outbound_value_btc,omitempty"`
	UniqueCounterparties       int    `json:"unique_counterparties"`
	CounterpartyResolutionMode string `json:"counterparty_resolution_mode,omitempty"`
	DominantRole               string `json:"dominant_role,omitempty"`
}

type BitcoinCounterparty struct {
	Address       string `json:"address"`
	Interactions  int    `json:"interactions"`
	InboundCount  int    `json:"inbound_count"`
	OutboundCount int    `json:"outbound_count"`
}

type BitcoinSampleEvent struct {
	Direction      string   `json:"direction,omitempty"`
	EventTime      string   `json:"event_time,omitempty"`
	TxHash         string   `json:"tx_hash,omitempty"`
	LinkedTxHash   string   `json:"linked_tx_hash,omitempty"`
	ValueSats      int64    `json:"value_sats"`
	ValueBTC       string   `json:"value_btc,omitempty"`
	ValueUSD       string   `json:"value_usd,omitempty"`
	Address        string   `json:"address,omitempty"`
	ScriptType     string   `json:"script_type,omitempty"`
	Counterparties []string `json:"counterparties,omitempty"`
}

type BitcoinCurationNotes struct {
	Narrative          string   `json:"narrative,omitempty"`
	BitcoinLayer       string   `json:"bitcoin_layer,omitempty"`
	IntendedTypologies []string `json:"intended_typologies,omitempty"`
	Limitations        []string `json:"limitations,omitempty"`
}

type BitcoinCuratedLayer1Case struct {
	CaseID            string                `json:"case_id"`
	Title             string                `json:"title"`
	Description       string                `json:"description,omitempty"`
	RiskPosture       string                `json:"risk_posture,omitempty"`
	Label             string                `json:"label,omitempty"`
	Address           string                `json:"address"`
	Chain             string                `json:"chain"`
	GeneratedAt       string                `json:"generated_at,omitempty"`
	SourceDatasetType string                `json:"source_dataset_type,omitempty"`
	SourceRowCount    int                   `json:"source_row_count"`
	UTXOSummary       BitcoinUTXOSummary    `json:"utxo_summary"`
	TopCounterparties []BitcoinCounterparty `json:"top_counterparties,omitempty"`
	SampleEvents      []BitcoinSampleEvent  `json:"sample_events,omitempty"`
	CurationNotes     BitcoinCurationNotes  `json:"curation_notes,omitempty"`
}

func LoadBitcoinCuratedLayer1Case(path string) (*BitcoinCuratedLayer1Case, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var cc BitcoinCuratedLayer1Case
	if err := json.Unmarshal(raw, &cc); err != nil {
		return nil, err
	}
	if err := requireCuratedIdentity("Bitcoin Layer 1", cc.CaseID, cc.Address, cc.Chain, "BITCOIN"); err != nil {
		return nil, err
	}

	return &cc, nil
}
