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
	"errors"
	"net"
	"net/http"
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

type localHTTPTestServer struct {
	URL   string
	close func() error
}

func (s *localHTTPTestServer) Close() {
	if s != nil && s.close != nil {
		_ = s.close()
	}
}

func newLocalHTTPTestServer(t *testing.T, handler http.Handler) *localHTTPTestServer {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on local test port: %v", err)
	}

	server := &http.Server{
		Handler: handler,
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("test server stopped unexpectedly: %v", err)
		}
	}()

	ts := &localHTTPTestServer{
		URL: "http://" + listener.Addr().String(),
		close: func() error {
			defer listener.Close()
			return server.Close()
		},
	}

	t.Cleanup(ts.Close)

	return ts
}

func TestCheckWatchlist_Success(t *testing.T) {
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)

	_, err := CheckWatchlist("0xd90e2f925da726b50c4ed8d0fb90ad053324f31b")
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestCheckWatchlist_Non200(t *testing.T) {
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))

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
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sanctioned": tru`))
	}))

	setEnvForTest(t, "WATCHLIST_ENGINE_URL", server.URL)

	_, err := CheckWatchlist("0xd90e2f925da726b50c4ed8d0fb90ad053324f31b")
	if err == nil {
		t.Fatalf("expected json decode error, got nil")
	}
}
