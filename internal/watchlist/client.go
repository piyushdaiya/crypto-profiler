package watchlist

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type EngineResponse struct {
	Sanctioned bool   `json:"sanctioned"`
	Currency   string `json:"currency"`
	Source     string `json:"source"`
}

// ---------------------------------------------------------
// CLIENT: Check Watchlist (HTTP)
// ---------------------------------------------------------

func CheckWatchlist(address string) (*EngineResponse, error) {
	engineURL := os.Getenv("WATCHLIST_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	checkURL := fmt.Sprintf("%s/check?address=%s", engineURL, url.QueryEscape(address))

	resp, err := client.Get(checkURL)
	if err != nil {
		return nil, fmt.Errorf("connection refused")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error %d", resp.StatusCode)
	}

	var result EngineResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
