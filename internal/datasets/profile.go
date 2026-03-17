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
