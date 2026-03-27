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

package watchlist

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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

func allowedWatchlistHosts() map[string]struct{} {
	raw := os.Getenv("WATCHLIST_ENGINE_ALLOWED_HOSTS")
	if raw == "" {
		raw = "localhost,127.0.0.1,::1"
	}

	allowed := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		host := strings.ToLower(strings.TrimSpace(part))
		if host == "" {
			continue
		}
		allowed[host] = struct{}{}
	}

	return allowed
}

func validatedEngineBaseURL() (*url.URL, error) {
	engineURL := os.Getenv("WATCHLIST_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080"
	}

	parsed, err := url.Parse(engineURL)
	if err != nil {
		return nil, fmt.Errorf("invalid watchlist engine url")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("invalid watchlist engine url")
	}

	if parsed.User != nil {
		return nil, fmt.Errorf("invalid watchlist engine url")
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return nil, fmt.Errorf("invalid watchlist engine url")
	}

	if _, ok := allowedWatchlistHosts()[host]; !ok {
		return nil, fmt.Errorf("watchlist engine host not allowed")
	}

	return parsed, nil
}

func buildCheckURL(base *url.URL, address string) *url.URL {
	u := *base
	u.Path = strings.TrimRight(base.Path, "/") + "/check"

	q := u.Query()
	q.Set("address", address)
	u.RawQuery = q.Encode()

	return &u
}

func CheckWatchlist(address string) (*EngineResponse, error) {
	baseURL, err := validatedEngineBaseURL()
	if err != nil {
		return nil, fmt.Errorf("connection refused")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	checkURL := buildCheckURL(baseURL, address)

	req := &http.Request{
		Method: http.MethodGet,
		URL:    checkURL,
		Header: make(http.Header),
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection refused")
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
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
