package validator

import (
	"context"
	"encoding/json"
	"fmt"
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

func (e *EVMStrategy) FetchState(ctx context.Context, address string, apiKey string) (*WalletProfile, error) {
	cleanAddr := strings.TrimSpace(address)
	
	profile := &WalletProfile{
		Address: cleanAddr,
		Network: "EVM",
		IsValid: true,
	}

	if apiKey == "" {
		profile.ValidationDetails = "Offline: No Etherscan API Key provided"
		return profile, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	
	// V2 Endpoint
	baseURL := "https://api.etherscan.io/v2/api"
	chainID := "1" // Ethereum Mainnet

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
		Status  string `json:"status"`
		Message string `json:"message"`
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

	var txs []struct {
		TimeStamp string `json:"timeStamp"`
	}
	if err := json.Unmarshal(txResp.Result, &txs); err != nil {
		profile.ValidationDetails += " | Error parsing tx list"
		return profile, nil
	}

	if len(txs) > 0 {
		profile.IsActive = true
		profile.TxCount = len(txs)

		firstTx := txs[0]
		ts, _ := strconv.ParseInt(firstTx.TimeStamp, 10, 64)
		firstTime := time.Unix(ts, 0)
		profile.FirstSeen = &firstTime

		lastTx := txs[len(txs)-1]
		tsLast, _ := strconv.ParseInt(lastTx.TimeStamp, 10, 64)
		lastTime := time.Unix(tsLast, 0)
		profile.LastSeen = &lastTime

		profile.ValidationDetails = fmt.Sprintf("Active | First Seen: %s", firstTime.Format("2006-01-02"))
	}

	return profile, nil
}

// ---------------------------------------------------------
// MISSING HELPER FUNCTION ADDED BELOW
// ---------------------------------------------------------

// getJSON handles GET requests and decodes the response
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