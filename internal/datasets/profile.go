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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func LoadCuratedCase(path string) (*CuratedCase, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var cc CuratedCase
	if err := json.Unmarshal(raw, &cc); err != nil {
		return nil, err
	}

	return &cc, nil
}

func BuildWalletProfileFromCuratedCase(cc *CuratedCase) *model.WalletProfile {
	details := "Dataset Mode"
	if cc.Title != "" {
		details += " | " + cc.Title
	}
	if cc.Label != "" {
		details += " | Label: " + cc.Label
	}
	if cc.RiskPosture != "" {
		details += " | Risk Posture: " + cc.RiskPosture
	}

	return &model.WalletProfile{
		Address:           strings.ToLower(strings.TrimSpace(cc.Address)),
		Network:           cc.Chain,
		IsValid:           true,
		ValidationDetails: details,
		IsActive:          cc.SourceTransferCount > 0,
		Balance:           "",
		TxCount:           cc.SourceTransferCount,
		FirstSeen:         cc.Summary.FirstSeen,
		LastSeen:          cc.Summary.LastSeen,
	}
}

func BuildTransactionsFromCuratedCase(cc *CuratedCase) []model.Transaction {
	out := make([]model.Transaction, 0, len(cc.SampleTransfers))

	for _, tr := range cc.SampleTransfers {
		out = append(out, model.Transaction{
			TimeStamp: tr.Timestamp.Unix(),
			From:      tr.From,
			To:        tr.To,
			Value:     tr.Amount,
			Hash:      tr.TxHash,
		})
	}

	return out
}
