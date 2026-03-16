package watchlist

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Response from the Watchlist Engine Service
type EngineResponse struct {
	Sanctioned bool   `json:"sanctioned"`
	Currency   string `json:"currency"`
	Source     string `json:"source"`
}

// ---------------------------------------------------------
// CLIENT: Check Watchlist (HTTP)
// ---------------------------------------------------------

func CheckWatchlist(address string) (*EngineResponse, error) {
	// Get Engine URL from Env (defaults to local for dev, or docker service name)
	engineURL := os.Getenv("WATCHLIST_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	// Short timeout - we don't want validation to hang if profiler is down
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("%s/check?address=%s", engineURL, address)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("connection refused")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server error %d", resp.StatusCode)
	}

	var result EngineResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
