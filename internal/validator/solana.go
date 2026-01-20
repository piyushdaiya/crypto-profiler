package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type SolanaStrategy struct{}

func (s *SolanaStrategy) Name() string {
	return "SOLANA"
}

func (s *SolanaStrategy) IsValidSyntax(address string) bool {
	cleanAddr := strings.TrimSpace(address)
	matched, _ := regexp.MatchString(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`, cleanAddr)
	return matched
}

func (s *SolanaStrategy) FetchState(ctx context.Context, address string, apiKey string) (*WalletProfile, error) {
	cleanAddr := strings.TrimSpace(address)
	profile := &WalletProfile{
		Address: cleanAddr,
		Network: "SOLANA",
		IsValid: true,
	}

	if apiKey == "" {
		profile.ValidationDetails = "Offline: No CoinStats API Key provided"
		return profile, nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	baseURL := "https://openapiv1.coinstats.app/wallet"
	connectionID := "solana"

	// STEP 1: Sync (Trigger and ignore error)
	syncURL := fmt.Sprintf("%s/transactions", baseURL)
	syncPayload := map[string]interface{}{
		"wallets": []map[string]string{{"address": cleanAddr, "connectionId": connectionID}},
	}
	_ = makeHTTPRequest(ctx, client, "PATCH", syncURL, apiKey, syncPayload, nil)

	// STEP 2: Get Balance
	balURL := fmt.Sprintf("%s/balance?address=%s&connectionId=%s", baseURL, cleanAddr, connectionID)
	
	// FIX: Use Slice for Balance Response
	var balResp []struct {
		CoinId string  `json:"coinId"`
		Amount float64 `json:"amount"`
		Symbol string  `json:"symbol"`
	}

	if err := makeHTTPRequest(ctx, client, "GET", balURL, apiKey, nil, &balResp); err != nil {
		profile.ValidationDetails = fmt.Sprintf("CoinStats Error: %v", err)
		return profile, nil
	}

	foundSol := false
	for _, coin := range balResp {
		if coin.Symbol == "SOL" {
			profile.Balance = fmt.Sprintf("%.9f SOL", coin.Amount)
			if coin.Amount > 0 { profile.IsActive = true }
			foundSol = true
			break
		}
	}
	if !foundSol { profile.Balance = "0.00000000 SOL" }

	// STEP 3: Get Transaction History (WITH RETRY LOGIC)
	txURL := fmt.Sprintf("%s/transactions?address=%s&connectionId=%s&limit=50", baseURL, cleanAddr, connectionID)

	var txResp struct {
		Meta struct { TotalCount int `json:"totalCount"` } `json:"meta"`
		Result []struct { Date string `json:"date"` } `json:"result"`
	}

	// Retry Loop: Try 3 times, waiting 2 seconds between tries
	var err error
	for i := 0; i < 3; i++ {
		err = makeHTTPRequest(ctx, client, "GET", txURL, apiKey, nil, &txResp)
		if err == nil {
			break // Success!
		}
		// Wait before retrying (simulating sync time)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		// If it fails after 3 tries, then report Pending
		profile.ValidationDetails += " | History Sync Pending (Try again in 1 min)"
		return profile, nil
	}

	profile.TxCount = txResp.Meta.TotalCount
	if len(txResp.Result) > 0 {
		profile.IsActive = true
		lastTx := txResp.Result[0]
		parsedLast, _ := time.Parse(time.RFC3339, lastTx.Date)
		profile.LastSeen = &parsedLast

		firstTx := txResp.Result[len(txResp.Result)-1]
		parsedFirst, _ := time.Parse(time.RFC3339, firstTx.Date)
		profile.FirstSeen = &parsedFirst
		
		profile.ValidationDetails = fmt.Sprintf("Active | Last Seen: %s", parsedLast.Format("2006-01-02"))
	} else {
		if !profile.IsActive {
			profile.ValidationDetails = "Inactive Account (No Tx History)"
		}
	}

	return profile, nil
}

func makeHTTPRequest(ctx context.Context, client *http.Client, method, url, apiKey string, payload interface{}, target interface{}) error {
	var body *bytes.Buffer
	if payload != nil {
		jsonBytes, _ := json.Marshal(payload)
		body = bytes.NewBuffer(jsonBytes)
	} else {
		body = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil { return err }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", apiKey)

	resp, err := client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()

	if resp.StatusCode >= 400 { return fmt.Errorf("HTTP %d", resp.StatusCode) }

	if target != nil { return json.NewDecoder(resp.Body).Decode(target) }
	return nil
}