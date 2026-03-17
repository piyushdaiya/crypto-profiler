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

package address

import (
	"context"
	"fmt"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type BitcoinStrategy struct{}

func (b *BitcoinStrategy) Name() string {
	return "BITCOIN"
}

func (b *BitcoinStrategy) IsValidSyntax(address string) bool {
	cleanAddr := strings.TrimSpace(address)
	// Regex covers Legacy (1...), Script (3...), Segwit (bc1q...), Taproot (bc1p...)
	legacy := regexp.MustCompile(`^[1][a-km-zA-HJ-NP-Z1-9]{25,34}$`)
	script := regexp.MustCompile(`^[3][a-km-zA-HJ-NP-Z1-9]{25,34}$`)
	bech32 := regexp.MustCompile(`(?i)^bc1[a-z0-9]{25,87}$`)

	return legacy.MatchString(cleanAddr) || script.MatchString(cleanAddr) || bech32.MatchString(cleanAddr)
}

func (b *BitcoinStrategy) FetchState(ctx context.Context, address string, _ string) (*model.WalletProfile, error) {
	// Note: Blockchain.com public API does not require an API Key for basic usage.
	// We ignore the configParam (API Key) here.

	cleanAddr := strings.TrimSpace(address)
	profile := &model.WalletProfile{
		Address: cleanAddr,
		Network: "BITCOIN",
		IsValid: true,
	}

	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://blockchain.info/rawaddr/%s", cleanAddr)

	var respObj struct {
		FinalBalance int64 `json:"final_balance"` // Satoshis
		NTx          int   `json:"n_tx"`          // Transaction Count
		Txs          []struct {
			Time int64 `json:"time"` // Unix Timestamp
		} `json:"txs"`
	}

	// 1. Fetch Data
	// Note: Blockchain.com returns 429 if rate limited (limit is strict for free tier).
	if err := getJSON(ctx, client, url, &respObj); err != nil {
		profile.ValidationDetails = fmt.Sprintf("Blockchain.com Error: %v", err)
		return profile, nil
	}

	// 2. Parse Balance (Satoshis -> BTC)
	profile.Balance = fmt.Sprintf("%.8f BTC", float64(respObj.FinalBalance)/1e8)
	profile.TxCount = respObj.NTx

	// 3. Determine Status and Dates
	if respObj.NTx > 0 {
		profile.IsActive = true
		profile.ValidationDetails = "Active Account (History Found)"

		// API returns transactions sorted by time (newest first usually),
		// but we scan to be safe or just take first/last if confident.
		// Blockchain.com rawaddr default sort is newest first.

		if len(respObj.Txs) > 0 {
			// Last Seen = Time of the first tx in the list (Newest)
			lastTime := time.Unix(respObj.Txs[0].Time, 0)
			profile.LastSeen = &lastTime

			// First Seen = Time of the last tx in the list (Oldest)
			// Note: rawaddr has a limit (default 50). If n_tx > 50, this is the "First Seen *recently*".
			// To get absolute first seen for huge wallets, you'd need to paginate.
			// For this implementation, we take the oldest returned in the batch.
			firstTime := time.Unix(respObj.Txs[len(respObj.Txs)-1].Time, 0)
			profile.FirstSeen = &firstTime

			profile.ValidationDetails += fmt.Sprintf(" | Last Active: %s", lastTime.Format("2006-01-02"))
		}
	} else {
		profile.IsActive = false
		profile.ValidationDetails = "Inactive Account (Zero Transactions)"
	}

	return profile, nil
}
