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

type ERC20Layer1Summary struct {
	FirstSeen                   string  `json:"first_seen,omitempty"`
	LastSeen                    string  `json:"last_seen,omitempty"`
	InboundTransferCount        int     `json:"inbound_transfer_count"`
	OutboundTransferCount       int     `json:"outbound_transfer_count"`
	SelfTransferCount           int     `json:"self_transfer_count,omitempty"`
	InboundCounterparties       int     `json:"inbound_counterparties"`
	OutboundCounterparties      int     `json:"outbound_counterparties"`
	UniqueCounterparties        int     `json:"unique_counterparties"`
	UniqueTokenContracts        int     `json:"unique_token_contracts"`
	RepeatedCounterparties      int     `json:"repeated_counterparties"`
	MaxCounterpartyInteractions int     `json:"max_counterparty_interactions"`
	DominantDirection           string  `json:"dominant_direction,omitempty"`
	DominantTokenAddress        string  `json:"dominant_token_address,omitempty"`
	DominantTokenSymbol         string  `json:"dominant_token_symbol,omitempty"`
	DominantTokenTransferShare  float64 `json:"dominant_token_transfer_share_pct,omitempty"`
}

type ERC20TokenBreakdown struct {
	TokenAddress         string `json:"token_address"`
	TokenName            string `json:"token_name,omitempty"`
	TokenSymbol          string `json:"token_symbol,omitempty"`
	TokenDecimals        int    `json:"token_decimals"`
	TransferCount        int    `json:"transfer_count"`
	InboundCount         int    `json:"inbound_count"`
	OutboundCount        int    `json:"outbound_count"`
	InboundValueRaw      string `json:"inbound_value_raw,omitempty"`
	OutboundValueRaw     string `json:"outbound_value_raw,omitempty"`
	UniqueCounterparties int    `json:"unique_counterparties"`
}

type ERC20TopToken struct {
	TokenAddress string `json:"token_address,omitempty"`
	Symbol       string `json:"symbol,omitempty"`
	Count        int    `json:"count"`
}

type ERC20CounterpartySummary struct {
	Address       string          `json:"address"`
	Label         string          `json:"label,omitempty"`
	Interactions  int             `json:"interactions"`
	InboundCount  int             `json:"inbound_count"`
	OutboundCount int             `json:"outbound_count"`
	UniqueTokens  int             `json:"unique_tokens"`
	TopTokens     []ERC20TopToken `json:"top_tokens,omitempty"`
}

type ERC20SampleTransfer struct {
	BlockTimestamp string `json:"block_timestamp,omitempty"`
	TxHash         string `json:"tx_hash,omitempty"`
	Sender         string `json:"sender,omitempty"`
	Recipient      string `json:"recipient,omitempty"`
	Direction      string `json:"direction,omitempty"`
	Counterparty   string `json:"counterparty,omitempty"`
	TokenAddress   string `json:"token_address,omitempty"`
	TokenName      string `json:"token_name,omitempty"`
	TokenSymbol    string `json:"token_symbol,omitempty"`
	TokenDecimals  int    `json:"token_decimals"`
	ValueRaw       string `json:"value_raw,omitempty"`
	ValueDisplay   string `json:"value_display,omitempty"`
	LabelSender    string `json:"label_sender,omitempty"`
	LabelRecipient string `json:"label_recipient,omitempty"`
}

type ERC20CurationNotes struct {
	Narrative          string   `json:"narrative,omitempty"`
	ERC20Layer         string   `json:"erc20_layer,omitempty"`
	IntendedTypologies []string `json:"intended_typologies,omitempty"`
	Limitations        []string `json:"limitations,omitempty"`
}

type ERC20CuratedLayer1Case struct {
	CaseID            string                     `json:"case_id"`
	Title             string                     `json:"title"`
	Description       string                     `json:"description,omitempty"`
	RiskPosture       string                     `json:"risk_posture,omitempty"`
	Label             string                     `json:"label,omitempty"`
	Address           string                     `json:"address"`
	Chain             string                     `json:"chain"`
	GeneratedAt       string                     `json:"generated_at,omitempty"`
	SourceDatasetType string                     `json:"source_dataset_type,omitempty"`
	SourceRowCount    int                        `json:"source_row_count"`
	RawSubsetFile     string                     `json:"raw_subset_file,omitempty"`
	ERC20Summary      ERC20Layer1Summary         `json:"erc20_summary"`
	TokenBreakdown    []ERC20TokenBreakdown      `json:"token_breakdown,omitempty"`
	TopCounterparties []ERC20CounterpartySummary `json:"top_counterparties,omitempty"`
	SampleTransfers   []ERC20SampleTransfer      `json:"sample_transfers,omitempty"`
	CurationNotes     ERC20CurationNotes         `json:"curation_notes,omitempty"`
}

func LoadERC20CuratedLayer1Case(path string) (*ERC20CuratedLayer1Case, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var cc ERC20CuratedLayer1Case
	if err := json.Unmarshal(raw, &cc); err != nil {
		return nil, err
	}
	if err := requireCuratedIdentity("ERC-20 Layer 1", cc.CaseID, cc.Address, cc.Chain, "EVM"); err != nil {
		return nil, err
	}

	return &cc, nil
}
