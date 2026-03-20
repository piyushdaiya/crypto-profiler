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

type SolanaStablecoinSummary struct {
	FirstSeen                 string `json:"first_seen,omitempty"`
	LastSeen                  string `json:"last_seen,omitempty"`
	SourceTransferCount       int    `json:"source_transfer_count"`
	DestinationTransferCount  int    `json:"destination_transfer_count"`
	AuthorityTransferCount    int    `json:"authority_transfer_count"`
	SourceValueRaw            string `json:"source_value_raw,omitempty"`
	DestinationValueRaw       string `json:"destination_value_raw,omitempty"`
	AuthorityValueRaw         string `json:"authority_value_raw,omitempty"`
	UniqueCounterparties      int    `json:"unique_counterparties"`
	SourceCounterparties      int    `json:"source_counterparties"`
	DestinationCounterparties int    `json:"destination_counterparties"`
	AuthorityCounterparties   int    `json:"authority_counterparties"`
	DominantRole              string `json:"dominant_role,omitempty"`
	DominantMint              string `json:"dominant_mint,omitempty"`
}

type SolanaMintCount struct {
	Mint  string `json:"mint"`
	Count int    `json:"count"`
}

type SolanaTransferTypeCount struct {
	TransferType string `json:"transfer_type"`
	Count        int    `json:"count"`
}

type SolanaStablecoinCounterparty struct {
	Address           string            `json:"address"`
	Interactions      int               `json:"interactions"`
	InboundCount      int               `json:"inbound_count"`
	OutboundCount     int               `json:"outbound_count"`
	AuthorityCount    int               `json:"authority_count"`
	TotalValueRaw     string            `json:"total_value_raw,omitempty"`
	InboundValueRaw   string            `json:"inbound_value_raw,omitempty"`
	OutboundValueRaw  string            `json:"outbound_value_raw,omitempty"`
	AuthorityValueRaw string            `json:"authority_value_raw,omitempty"`
	TopMints          []SolanaMintCount `json:"top_mints,omitempty"`
}

type SolanaAuthorityPair struct {
	Source        string `json:"source"`
	Destination   string `json:"destination"`
	Interactions  int    `json:"interactions"`
	TotalValueRaw string `json:"total_value_raw,omitempty"`
}

type SolanaSampleTransfer struct {
	BlockTimestamp string `json:"block_timestamp,omitempty"`
	TxSignature    string `json:"tx_signature,omitempty"`
	Source         string `json:"source,omitempty"`
	Destination    string `json:"destination,omitempty"`
	Authority      string `json:"authority,omitempty"`
	TokenAmountRaw string `json:"token_amount_raw,omitempty"`
	TokenAmountUI  string `json:"token_amount_ui,omitempty"`
	Decimals       int    `json:"decimals"`
	Mint           string `json:"mint,omitempty"`
	TransferType   string `json:"transfer_type,omitempty"`
}

type SolanaCurationNotes struct {
	Narrative          string   `json:"narrative,omitempty"`
	SolanaLayer        string   `json:"solana_layer,omitempty"`
	IntendedTypologies []string `json:"intended_typologies,omitempty"`
	Limitations        []string `json:"limitations,omitempty"`
}

type SolanaCuratedStablecoinCase struct {
	CaseID                string                         `json:"case_id"`
	Title                 string                         `json:"title"`
	Description           string                         `json:"description,omitempty"`
	RiskPosture           string                         `json:"risk_posture,omitempty"`
	Label                 string                         `json:"label,omitempty"`
	Address               string                         `json:"address"`
	Chain                 string                         `json:"chain"`
	GeneratedAt           string                         `json:"generated_at,omitempty"`
	SourceDatasetType     string                         `json:"source_dataset_type,omitempty"`
	SourceRowCount        int                            `json:"source_row_count"`
	StablecoinSummary     SolanaStablecoinSummary        `json:"stablecoin_summary"`
	MintBreakdown         []SolanaMintCount              `json:"mint_breakdown,omitempty"`
	TransferTypeBreakdown []SolanaTransferTypeCount      `json:"transfer_type_breakdown,omitempty"`
	TopCounterparties     []SolanaStablecoinCounterparty `json:"top_counterparties,omitempty"`
	TopAuthorityPairs     []SolanaAuthorityPair          `json:"top_authority_pairs,omitempty"`
	SampleTransfers       []SolanaSampleTransfer         `json:"sample_transfers,omitempty"`
	CurationNotes         SolanaCurationNotes            `json:"curation_notes,omitempty"`
}

func LoadSolanaCuratedStablecoinCase(path string) (*SolanaCuratedStablecoinCase, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var cc SolanaCuratedStablecoinCase
	if err := json.Unmarshal(raw, &cc); err != nil {
		return nil, err
	}

	return &cc, nil
}
