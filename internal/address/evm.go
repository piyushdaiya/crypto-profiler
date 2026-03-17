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
	"encoding/json"
	"fmt"
	"github.com/piyushdaiya/crypto-profiler/internal/analyzer"
	"github.com/piyushdaiya/crypto-profiler/internal/model"

	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type EVMStrategy struct{}

func (e *EVMStrategy) Name() string {
	return "EVM (Etherscan)"
}

func (e *EVMStrategy) IsValidSyntax(address string) bool {
	cleanAddr := strings.TrimSpace(address)
	regex := regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)
	return regex.MatchString(cleanAddr)
}

func (e *EVMStrategy) FetchState(ctx context.Context, address string, apiKey string) (*model.WalletProfile, error) {
	cleanAddr := strings.TrimSpace(address)

	profile := &model.WalletProfile{
		Address: cleanAddr,
		Network: "EVM",
		IsValid: true,
	}

	if apiKey == "" {
		profile.ValidationDetails = "Offline: No Etherscan API Key provided"
		return profile, nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	baseURL := "https://api.etherscan.io/v2/api"
	chainID := "1"

	// ---------------------------------------------------------
	// CALL 1: Get Balance
	// ---------------------------------------------------------
	balURL := fmt.Sprintf("%s?chainid=%s&module=account&action=balance&address=%s&tag=latest&apikey=%s", baseURL, chainID, cleanAddr, apiKey)

	var balResp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	if err := getJSON(ctx, client, balURL, &balResp); err != nil {
		profile.ValidationDetails = fmt.Sprintf("Network Error (Balance): %v", err)
		return profile, nil
	}

	if balResp.Status == "0" && balResp.Message != "OK" {
		profile.ValidationDetails = fmt.Sprintf("Etherscan API Error: %s", balResp.Result)
		return profile, nil
	}

	wei := new(big.Float)
	wei.SetString(balResp.Result)
	ethValue := new(big.Float).Quo(wei, big.NewFloat(1e18))
	profile.Balance = fmt.Sprintf("%.4f ETH", ethValue)

	if balResp.Result != "0" {
		profile.IsActive = true
	}

	// ---------------------------------------------------------
	// CALL 2: Get Transaction History
	// ---------------------------------------------------------
	txURL := fmt.Sprintf("%s?chainid=%s&module=account&action=txlist&address=%s&startblock=0&endblock=99999999&sort=asc&apikey=%s", baseURL, chainID, cleanAddr, apiKey)

	var txResp struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Result  json.RawMessage `json:"result"`
	}

	if err := getJSON(ctx, client, txURL, &txResp); err != nil {
		profile.ValidationDetails += fmt.Sprintf(" | History Fetch Failed: %v", err)
		return profile, nil
	}

	if txResp.Status == "0" {
		if txResp.Message == "No transactions found" {
			if !profile.IsActive {
				profile.ValidationDetails = "Inactive Account (No Tx History)"
			}
			return profile, nil
		} else {
			var errorMsg string
			_ = json.Unmarshal(txResp.Result, &errorMsg)
			profile.ValidationDetails += fmt.Sprintf(" | API Error: %s - %s", txResp.Message, errorMsg)
			return profile, nil
		}
	}

	// ---------------------------------------------------------
	// PREPARE FOR INVESTIGATOR
	// ---------------------------------------------------------
	var rawTxs []struct {
		TimeStamp string `json:"timeStamp"`
		From      string `json:"from"`
		To        string `json:"to"`
		Value     string `json:"value"`
		Hash      string `json:"hash"`
	}

	if err := json.Unmarshal(txResp.Result, &rawTxs); err != nil {
		profile.ValidationDetails += " | Error parsing tx list"
		return profile, nil
	}

	var investigationTxs []model.Transaction
	for _, t := range rawTxs {
		ts, _ := strconv.ParseInt(t.TimeStamp, 10, 64)
		investigationTxs = append(investigationTxs, model.Transaction{
			TimeStamp: ts,
			From:      t.From,
			To:        t.To,
			Value:     t.Value,
			Hash:      t.Hash,
		})
	}

	if len(investigationTxs) > 0 {
		profile.IsActive = true
		profile.TxCount = len(investigationTxs)

		firstTime := time.Unix(investigationTxs[0].TimeStamp, 0)
		profile.FirstSeen = &firstTime

		lastTime := time.Unix(investigationTxs[len(investigationTxs)-1].TimeStamp, 0)
		profile.LastSeen = &lastTime

		profile.ValidationDetails = fmt.Sprintf("Active | First Seen: %s", firstTime.Format("2006-01-02"))
	}

	// ---------------------------------------------------------
	// CALL 3: THE INVESTIGATOR
	// ---------------------------------------------------------
	// UPDATED: Now calls Investigate with only 2 arguments.
	// The HTTP client inside Investigate handles the profiler connection.
	analyzer.Investigate(profile, investigationTxs)

	return profile, nil
}

// getJSON Helper
func getJSON(ctx context.Context, client *http.Client, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
