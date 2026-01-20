package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// makeRPCCall handles JSON-RPC POST requests and checks for HTTP errors
func makeRPCCall(ctx context.Context, url string, payload interface{}) (string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshalling error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	// NEW: Check for HTTP errors (e.g., 401 Unauthorized, 429 Too Many Requests)
	if resp.StatusCode >= 400 {
		bodyDump, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyDump))
	}

	respBytes, _ := io.ReadAll(resp.Body)

	var rpcResp struct {
		Result interface{} `json:"result"`
		Error  interface{} `json:"error"`
	}

	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		// If we can't parse JSON, return the raw string so we see the error (e.g. HTML body)
		return "", fmt.Errorf("bad response format: %s", string(respBytes))
	}

	if rpcResp.Error != nil {
		return "", fmt.Errorf("RPC Error: %v", rpcResp.Error)
	}

	// Handle different result types
	switch v := rpcResp.Result.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	default:
		// Re-marshal complex objects (like Solana context results) back to string for the strategy to parse
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes), nil
	}
}