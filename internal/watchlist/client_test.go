package watchlist

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func setEnvForTest(t *testing.T, key, value string) {
	t.Helper()

	original, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if existed {
			err = os.Setenv(key, original)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("failed to restore env %s: %v", key, err)
		}
	})
}

func TestCheckWatchlist_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/check" {
			http.NotFound(w, r)
			return
		}

		resp := EngineResponse{
			Sanctioned: true,
			Currency:   "ETH",
			Source:     "OFAC",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)

	got, err := CheckWatchlist("0xd90e2f925da726b50c4ed8d0fb90ad053324f31b")
	if err != nil {
		t.Fatalf("CheckWatchlist() returned unexpected error: %v", err)
	}

	if !got.Sanctioned {
		t.Fatalf("expected sanctioned=true")
	}
	if got.Currency != "ETH" {
		t.Fatalf("expected currency ETH, got %q", got.Currency)
	}
	if got.Source != "OFAC" {
		t.Fatalf("expected source OFAC, got %q", got.Source)
	}
}

func TestCheckWatchlist_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)

	_, err := CheckWatchlist("0xd90e2f925da726b50c4ed8d0fb90ad053324f31b")
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestCheckWatchlist_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)

	_, err := CheckWatchlist("0xd90e2f925da726b50c4ed8d0fb90ad053324f31b")
	if err == nil {
		t.Fatalf("expected non-200 error, got nil")
	}

	if !strings.Contains(err.Error(), "server error") {
		t.Fatalf("expected server error, got %v", err)
	}
}

func TestCheckWatchlist_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sanctioned": tru`))
	}))
	defer server.Close()

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)

	_, err := CheckWatchlist("0xd90e2f925da726b50c4ed8d0fb90ad053324f31b")
	if err == nil {
		t.Fatalf("expected json decode error, got nil")
	}
}
