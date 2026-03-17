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

package address

import "testing"

func TestEVMStrategy_IsValidSyntax(t *testing.T) {
	strategy := &EVMStrategy{}

	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "valid mixed-case checksum address",
			address: "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
			want:    true,
		},
		{
			name:    "valid lowercase address",
			address: "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
			want:    true,
		},
		{
			name:    "too short",
			address: "0x12345",
			want:    false,
		},
		{
			name:    "missing 0x prefix",
			address: "d8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
			want:    false,
		},
		{
			name:    "non-hex characters",
			address: "0xZZZZ6BF26964aF9D7eEd9e03E53415D37aA96045",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.IsValidSyntax(tt.address)
			if got != tt.want {
				t.Fatalf("IsValidSyntax(%q) = %v, want %v", tt.address, got, tt.want)
			}
		})
	}
}

func TestBitcoinStrategy_IsValidSyntax(t *testing.T) {
	strategy := &BitcoinStrategy{}

	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "valid bech32 address",
			address: "bc1qcp6fr7gtyukympl6unr7uv78h3vprycwj455zx",
			want:    true,
		},
		{
			name:    "valid legacy address",
			address: "1BoatSLRHtKNngkdXEeobR76b53LETtpyT",
			want:    true,
		},
		{
			name:    "valid p2sh address",
			address: "3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy",
			want:    true,
		},
		{
			name:    "invalid random string",
			address: "not-a-bitcoin-address",
			want:    false,
		},
		{
			name:    "evm address should not pass",
			address: "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.IsValidSyntax(tt.address)
			if got != tt.want {
				t.Fatalf("IsValidSyntax(%q) = %v, want %v", tt.address, got, tt.want)
			}
		})
	}
}

func TestSolanaStrategy_IsValidSyntax(t *testing.T) {
	strategy := &SolanaStrategy{}

	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "valid solana address",
			address: "So11111111111111111111111111111111111111112",
			want:    true,
		},
		{
			name:    "another valid solana-style address",
			address: "Vote111111111111111111111111111111111111111",
			want:    true,
		},
		{
			name:    "invalid random string",
			address: "not-a-solana-address",
			want:    false,
		},
		{
			name:    "contains invalid base58 chars",
			address: "O0Il-not-valid-base58",
			want:    false,
		},
		{
			name:    "evm address should not pass",
			address: "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.IsValidSyntax(tt.address)
			if got != tt.want {
				t.Fatalf("IsValidSyntax(%q) = %v, want %v", tt.address, got, tt.want)
			}
		})
	}
}
