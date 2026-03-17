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

import "time"

type Transfer struct {
	Chain        string    `json:"chain"`
	AssetType    string    `json:"asset_type"` // NATIVE, ERC20
	TxHash       string    `json:"tx_hash"`
	BlockID      int64     `json:"block_id"`
	Timestamp    time.Time `json:"timestamp"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	Amount       string    `json:"amount"`
	AssetAddress string    `json:"asset_address,omitempty"`
	AssetName    string    `json:"asset_name,omitempty"`
	AssetSymbol  string    `json:"asset_symbol,omitempty"`
	LabelFrom    string    `json:"label_from,omitempty"`
	LabelTo      string    `json:"label_to,omitempty"`
}

type Summary struct {
	FirstSeen            *time.Time `json:"first_seen,omitempty"`
	LastSeen             *time.Time `json:"last_seen,omitempty"`
	InboundCount         int        `json:"inbound_count"`
	OutboundCount        int        `json:"outbound_count"`
	UniqueCounterparties int        `json:"unique_counterparties"`
	NativeTransferCount  int        `json:"native_transfer_count"`
	ERC20TransferCount   int        `json:"erc20_transfer_count"`
}

type AddressDataset struct {
	Address     string     `json:"address"`
	Chain       string     `json:"chain"`
	Label       string     `json:"label,omitempty"`
	GeneratedAt time.Time  `json:"generated_at"`
	Summary     Summary    `json:"summary"`
	Transfers   []Transfer `json:"transfers"`
}
