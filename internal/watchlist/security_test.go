package watchlist

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckWatchlist_QueryEscapesAddress(t *testing.T) {
	rawAddress := `0xabc123&debug=true`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("address")
		if got != rawAddress {
			t.Fatalf("expected escaped address %q, got %q", rawAddress, got)
		}

		resp := EngineResponse{
			Sanctioned: false,
			Currency:   "ETH",
			Source:     "OFAC",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)

	_, err := CheckWatchlist(rawAddress)
	if err != nil {
		t.Fatalf("CheckWatchlist() returned unexpected error: %v", err)
	}
}

func TestCheckWatchlist_ConnectionErrorIsGeneric(t *testing.T) {
	setEnvForTest(t, "WATCHLIST_ENGINE_URL", "http://127.0.0.1:1")

	_, err := CheckWatchlist("0xd90e2f925da726b50c4ed8d0fb90ad053324f31b")
	if err == nil {
		t.Fatalf("expected connection error, got nil")
	}

	if err.Error() != "connection refused" {
		t.Fatalf("expected generic connection error, got %q", err.Error())
	}

	if strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("error should not leak internal endpoint details: %q", err.Error())
	}
}
