package datasets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CaseMetadata struct {
	CaseID      string `json:"case_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	RiskPosture string `json:"risk_posture,omitempty"`
}

type CounterpartySummary struct {
	Address       string `json:"address"`
	Label         string `json:"label,omitempty"`
	Interactions  int    `json:"interactions"`
	InboundCount  int    `json:"inbound_count"`
	OutboundCount int    `json:"outbound_count"`
}

type CuratedCase struct {
	CaseID              string                `json:"case_id"`
	Title               string                `json:"title"`
	Description         string                `json:"description,omitempty"`
	RiskPosture         string                `json:"risk_posture,omitempty"`
	Address             string                `json:"address"`
	Chain               string                `json:"chain"`
	Label               string                `json:"label,omitempty"`
	GeneratedAt         time.Time             `json:"generated_at"`
	Summary             Summary               `json:"summary"`
	TopCounterparties   []CounterpartySummary `json:"top_counterparties"`
	SampleTransfers     []Transfer            `json:"sample_transfers"`
	SourceTransferCount int                   `json:"source_transfer_count"`
}

type counterpartyAgg struct {
	Address       string
	Label         string
	Interactions  int
	InboundCount  int
	OutboundCount int
}

func LoadCaseManifest(path string) (map[string]CaseMetadata, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var manifest map[string]CaseMetadata
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}

	out := make(map[string]CaseMetadata, len(manifest))
	for addr, meta := range manifest {
		out[NormalizeHexAddress(addr)] = meta
	}

	return out, nil
}

func LoadAddressDataset(path string) (*AddressDataset, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var ds AddressDataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, err
	}

	return &ds, nil
}

func CurateDataset(ds *AddressDataset, meta CaseMetadata, topN, sampleLimit int) *CuratedCase {
	if topN <= 0 {
		topN = 20
	}
	if sampleLimit <= 0 {
		sampleLimit = 50
	}

	return &CuratedCase{
		CaseID:              meta.CaseID,
		Title:               meta.Title,
		Description:         meta.Description,
		RiskPosture:         meta.RiskPosture,
		Address:             ds.Address,
		Chain:               ds.Chain,
		Label:               ds.Label,
		GeneratedAt:         time.Now().UTC(),
		Summary:             ds.Summary,
		TopCounterparties:   summarizeCounterparties(ds, topN),
		SampleTransfers:     curateTransferSample(ds.Transfers, sampleLimit),
		SourceTransferCount: len(ds.Transfers),
	}
}

func summarizeCounterparties(ds *AddressDataset, topN int) []CounterpartySummary {
	stats := map[string]*counterpartyAgg{}

	for _, tr := range ds.Transfers {
		var cpAddr, cpLabel string

		switch {
		case strings.EqualFold(tr.From, ds.Address):
			cpAddr = NormalizeHexAddress(tr.To)
			cpLabel = tr.LabelTo
			if cpAddr == "" || strings.EqualFold(cpAddr, ds.Address) {
				continue
			}
			entry := ensureCounterparty(stats, cpAddr, cpLabel)
			entry.Interactions++
			entry.OutboundCount++

		case strings.EqualFold(tr.To, ds.Address):
			cpAddr = NormalizeHexAddress(tr.From)
			cpLabel = tr.LabelFrom
			if cpAddr == "" || strings.EqualFold(cpAddr, ds.Address) {
				continue
			}
			entry := ensureCounterparty(stats, cpAddr, cpLabel)
			entry.Interactions++
			entry.InboundCount++
		}
	}

	out := make([]CounterpartySummary, 0, len(stats))
	for _, v := range stats {
		out = append(out, CounterpartySummary{
			Address:       v.Address,
			Label:         v.Label,
			Interactions:  v.Interactions,
			InboundCount:  v.InboundCount,
			OutboundCount: v.OutboundCount,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Interactions != out[j].Interactions {
			return out[i].Interactions > out[j].Interactions
		}
		return out[i].Address < out[j].Address
	})

	if len(out) > topN {
		out = out[:topN]
	}

	return out
}

func ensureCounterparty(stats map[string]*counterpartyAgg, address, label string) *counterpartyAgg {
	if existing, ok := stats[address]; ok {
		if existing.Label == "" && label != "" {
			existing.Label = label
		}
		return existing
	}

	entry := &counterpartyAgg{
		Address: address,
		Label:   label,
	}
	stats[address] = entry
	return entry
}

func curateTransferSample(transfers []Transfer, sampleLimit int) []Transfer {
	if len(transfers) <= sampleLimit {
		return transfers
	}

	firstCount := sampleLimit / 2
	lastCount := sampleLimit - firstCount

	selected := make([]Transfer, 0, sampleLimit)
	seen := map[string]struct{}{}

	add := func(tr Transfer) {
		key := transferKey(tr)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		selected = append(selected, tr)
	}

	for i := 0; i < firstCount && i < len(transfers); i++ {
		add(transfers[i])
	}

	start := len(transfers) - lastCount
	if start < 0 {
		start = 0
	}
	for i := start; i < len(transfers); i++ {
		add(transfers[i])
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Timestamp.Before(selected[j].Timestamp)
	})

	return selected
}

func transferKey(tr Transfer) string {
	return strings.Join([]string{
		tr.TxHash,
		tr.From,
		tr.To,
		tr.AssetType,
		tr.AssetAddress,
		tr.Timestamp.Format(time.RFC3339),
	}, "|")
}
