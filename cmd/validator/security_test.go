package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func TestRun_NoArgumentsReturnsUsageError(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run(nil, &out, &errOut, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if !strings.Contains(errOut.String(), "Usage: ./validator <wallet-address>") {
		t.Fatalf("expected usage message, got %q", errOut.String())
	}
}

func TestRun_MalformedLongInputFallsBackSafely(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	input := strings.Repeat("A", 5000)
	code := run([]string{input}, &out, &errOut, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, errOut.String())
	}

	var got model.WalletProfile
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode stdout json: %v\nstdout=%s", err, out.String())
	}

	if got.Network != "UNKNOWN" {
		t.Fatalf("expected UNKNOWN network, got %q", got.Network)
	}

	if got.IsValid {
		t.Fatalf("expected invalid wallet")
	}

	if !strings.Contains(got.ValidationDetails, "Invalid format") {
		t.Fatalf("expected invalid-format fallback, got %q", got.ValidationDetails)
	}
}
