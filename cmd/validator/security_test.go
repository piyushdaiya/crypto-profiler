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
