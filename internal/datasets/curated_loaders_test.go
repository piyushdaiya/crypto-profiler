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

package datasets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSolanaCuratedStablecoinCase_ValidJSON(t *testing.T) {
	path := filepath.Join("..", "..", "data", "cases", "curated-solana", "solana-stablecoin-authority-operator.json")

	got, err := LoadSolanaCuratedStablecoinCase(path)
	if err != nil {
		t.Fatalf("expected valid Solana curated case, got error: %v", err)
	}

	if got.CaseID != "solana-stablecoin-authority-operator" {
		t.Fatalf("unexpected case id %q", got.CaseID)
	}

	if got.Chain != "SOLANA" {
		t.Fatalf("expected SOLANA chain, got %q", got.Chain)
	}
}

func TestLoadSolanaCuratedStablecoinCase_InvalidJSON(t *testing.T) {
	path := writeDatasetTestFile(t, "{not-json")

	_, err := LoadSolanaCuratedStablecoinCase(path)
	if err == nil {
		t.Fatalf("expected invalid JSON error")
	}
}

func TestLoadSolanaCuratedStablecoinCase_MissingRequiredFields(t *testing.T) {
	path := writeDatasetTestFile(t, `{"case_id":"","address":"","chain":"SOLANA","stablecoin_summary":{}}`)

	_, err := LoadSolanaCuratedStablecoinCase(path)
	if err == nil || !strings.Contains(err.Error(), "missing case_id") {
		t.Fatalf("expected missing case_id error, got %v", err)
	}
}

func TestLoadBitcoinCuratedLayer1Case_ValidJSON(t *testing.T) {
	path := filepath.Join("..", "..", "data", "cases", "curated-bitcoin", "bitcoin-broad-spend-heavy-operational-hub.json")

	got, err := LoadBitcoinCuratedLayer1Case(path)
	if err != nil {
		t.Fatalf("expected valid Bitcoin curated case, got error: %v", err)
	}

	if got.CaseID != "bitcoin-broad-spend-heavy-operational-hub" {
		t.Fatalf("unexpected case id %q", got.CaseID)
	}

	if got.Chain != "BITCOIN" {
		t.Fatalf("expected BITCOIN chain, got %q", got.Chain)
	}
}

func TestLoadBitcoinCuratedLayer1Case_InvalidJSON(t *testing.T) {
	path := writeDatasetTestFile(t, "{not-json")

	_, err := LoadBitcoinCuratedLayer1Case(path)
	if err == nil {
		t.Fatalf("expected invalid JSON error")
	}
}

func TestLoadBitcoinCuratedLayer1Case_MissingRequiredFields(t *testing.T) {
	path := writeDatasetTestFile(t, `{"case_id":"btc-case","address":"","chain":"BITCOIN","utxo_summary":{}}`)

	_, err := LoadBitcoinCuratedLayer1Case(path)
	if err == nil || !strings.Contains(err.Error(), "missing address") {
		t.Fatalf("expected missing address error, got %v", err)
	}
}

func TestLoadERC20CuratedLayer1Case_ValidJSON(t *testing.T) {
	path := filepath.Join("..", "..", "data", "cases", "curated-erc20", "erc20-exchange-like-broad-service-surface.json")

	got, err := LoadERC20CuratedLayer1Case(path)
	if err != nil {
		t.Fatalf("expected valid ERC-20 curated case, got error: %v", err)
	}

	if got.CaseID != "erc20-exchange-like-broad-service-surface" {
		t.Fatalf("unexpected case id %q", got.CaseID)
	}

	if got.Chain != "EVM" {
		t.Fatalf("expected EVM chain, got %q", got.Chain)
	}
}

func TestLoadERC20CuratedLayer1Case_InvalidJSON(t *testing.T) {
	path := writeDatasetTestFile(t, "{not-json")

	_, err := LoadERC20CuratedLayer1Case(path)
	if err == nil {
		t.Fatalf("expected invalid JSON error")
	}
}

func TestLoadERC20CuratedLayer1Case_MissingRequiredFields(t *testing.T) {
	path := writeDatasetTestFile(t, `{"case_id":"erc20-case","address":"0xabc","chain":"","erc20_summary":{}}`)

	_, err := LoadERC20CuratedLayer1Case(path)
	if err == nil || !strings.Contains(err.Error(), "missing chain") {
		t.Fatalf("expected missing chain error, got %v", err)
	}
}

func TestLoadExtractedTraceDatasetByAddress_MissingFileReturnsNil(t *testing.T) {
	got, err := LoadExtractedTraceDatasetByAddress(t.TempDir(), "0xe592427a0aece92de3edee1f18e0157c05861564")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil dataset when file does not exist")
	}
}

func TestLoadExtractedTraceDatasetByAddress_RejectsPathTraversalInput(t *testing.T) {
	_, err := LoadExtractedTraceDatasetByAddress(t.TempDir(), "../../etc/passwd")
	if err == nil || !strings.Contains(err.Error(), "invalid address") {
		t.Fatalf("expected invalid address error, got %v", err)
	}
}

func writeDatasetTestFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "case.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	return path
}
