package datasets

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func NormalizeHexAddress(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == `\n` || s == `\N` {
		return ""
	}
	s = strings.TrimPrefix(s, "0x")
	if len(s) == 40 {
		return "0x" + s
	}
	return s
}

func LoadLegacyLabels(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var labels map[string]string
	if err := json.Unmarshal(raw, &labels); err != nil {
		return nil, err
	}

	out := make(map[string]string, len(labels))
	for addr, label := range labels {
		norm := NormalizeHexAddress(addr)
		if norm == "" {
			continue
		}
		out[norm] = label
	}

	return out, nil
}

func LoadAddressList(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var out []string
	seen := map[string]struct{}{}

	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		norm := NormalizeHexAddress(line)
		if norm == "" {
			continue
		}
		if _, exists := seen[norm]; exists {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}

	sort.Strings(out)
	return out, nil
}

func ExtractEVMCaseDatasets(ethPaths, erc20Paths []string, targetAddresses []string, labels map[string]string) (map[string]*AddressDataset, error) {
	targetSet := map[string]struct{}{}
	results := map[string]*AddressDataset{}

	for _, addr := range targetAddresses {
		targetSet[addr] = struct{}{}
		results[addr] = &AddressDataset{
			Address:     addr,
			Chain:       "EVM",
			Label:       labels[addr],
			GeneratedAt: time.Now().UTC(),
			Transfers:   make([]Transfer, 0),
		}
	}

	for _, ethPath := range ethPaths {
		if err := extractEthereumTransactions(ethPath, targetSet, labels, results); err != nil {
			return nil, fmt.Errorf("extract ethereum transactions from %s: %w", ethPath, err)
		}
	}

	for _, erc20Path := range erc20Paths {
		if err := extractERC20Transactions(erc20Path, targetSet, labels, results); err != nil {
			return nil, fmt.Errorf("extract erc20 transactions from %s: %w", erc20Path, err)
		}
	}

	for _, ds := range results {
		finalizeSummary(ds)
		sort.Slice(ds.Transfers, func(i, j int) bool {
			return ds.Transfers[i].Timestamp.Before(ds.Transfers[j].Timestamp)
		})
	}

	return results, nil
}

func extractEthereumTransactions(path string, targetSet map[string]struct{}, labels map[string]string, results map[string]*AddressDataset) error {
	f, err := openMaybeGzip(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		return err
	}
	index := headerIndex(header)

	for {
		row, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		txHash := field(row, index, "hash")
		sender := NormalizeHexAddress(field(row, index, "sender"))
		recipient := NormalizeHexAddress(field(row, index, "recipient"))
		timeStr := field(row, index, "time")
		blockID := parseInt64(field(row, index, "block_id"))
		value := field(row, index, "value")

		if txHash == "" || sender == "" || recipient == "" || timeStr == "" {
			continue
		}

		ts, err := parseBlockchairTime(timeStr)
		if err != nil {
			continue
		}

		if _, ok := targetSet[sender]; ok {
			results[sender].Transfers = append(results[sender].Transfers, Transfer{
				Chain:       "EVM",
				AssetType:   "NATIVE",
				TxHash:      txHash,
				BlockID:     blockID,
				Timestamp:   ts,
				From:        sender,
				To:          recipient,
				Amount:      value,
				LabelFrom:   labels[sender],
				LabelTo:     labels[recipient],
				AssetSymbol: "ETH",
				AssetName:   "Ethereum",
			})
		}

		if _, ok := targetSet[recipient]; ok {
			results[recipient].Transfers = append(results[recipient].Transfers, Transfer{
				Chain:       "EVM",
				AssetType:   "NATIVE",
				TxHash:      txHash,
				BlockID:     blockID,
				Timestamp:   ts,
				From:        sender,
				To:          recipient,
				Amount:      value,
				LabelFrom:   labels[sender],
				LabelTo:     labels[recipient],
				AssetSymbol: "ETH",
				AssetName:   "Ethereum",
			})
		}
	}
}

func extractERC20Transactions(path string, targetSet map[string]struct{}, labels map[string]string, results map[string]*AddressDataset) error {
	f, err := openMaybeGzip(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		return err
	}
	index := headerIndex(header)

	for {
		row, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		txHash := field(row, index, "transaction_hash")
		sender := NormalizeHexAddress(field(row, index, "sender"))
		recipient := NormalizeHexAddress(field(row, index, "recipient"))
		tokenAddress := NormalizeHexAddress(field(row, index, "token_address"))
		tokenName := field(row, index, "token_name")
		tokenSymbol := field(row, index, "token_symbol")
		value := field(row, index, "value")
		blockID := parseInt64(field(row, index, "block_id"))
		timeStr := field(row, index, "time")

		if txHash == "" || sender == "" || recipient == "" || timeStr == "" {
			continue
		}

		ts, err := parseBlockchairTime(timeStr)
		if err != nil {
			continue
		}

		if _, ok := targetSet[sender]; ok {
			results[sender].Transfers = append(results[sender].Transfers, Transfer{
				Chain:        "EVM",
				AssetType:    "ERC20",
				TxHash:       txHash,
				BlockID:      blockID,
				Timestamp:    ts,
				From:         sender,
				To:           recipient,
				Amount:       value,
				AssetAddress: tokenAddress,
				AssetName:    tokenName,
				AssetSymbol:  tokenSymbol,
				LabelFrom:    labels[sender],
				LabelTo:      labels[recipient],
			})
		}

		if _, ok := targetSet[recipient]; ok {
			results[recipient].Transfers = append(results[recipient].Transfers, Transfer{
				Chain:        "EVM",
				AssetType:    "ERC20",
				TxHash:       txHash,
				BlockID:      blockID,
				Timestamp:    ts,
				From:         sender,
				To:           recipient,
				Amount:       value,
				AssetAddress: tokenAddress,
				AssetName:    tokenName,
				AssetSymbol:  tokenSymbol,
				LabelFrom:    labels[sender],
				LabelTo:      labels[recipient],
			})
		}
	}
}

func finalizeSummary(ds *AddressDataset) {
	counterparties := map[string]struct{}{}

	for _, tr := range ds.Transfers {
		if tr.To == ds.Address {
			ds.Summary.InboundCount++
			if tr.From != "" && tr.From != ds.Address {
				counterparties[tr.From] = struct{}{}
			}
		}
		if tr.From == ds.Address {
			ds.Summary.OutboundCount++
			if tr.To != "" && tr.To != ds.Address {
				counterparties[tr.To] = struct{}{}
			}
		}

		switch tr.AssetType {
		case "NATIVE":
			ds.Summary.NativeTransferCount++
		case "ERC20":
			ds.Summary.ERC20TransferCount++
		}

		ts := tr.Timestamp
		if ds.Summary.FirstSeen == nil || ts.Before(*ds.Summary.FirstSeen) {
			t := ts
			ds.Summary.FirstSeen = &t
		}
		if ds.Summary.LastSeen == nil || ts.After(*ds.Summary.LastSeen) {
			t := ts
			ds.Summary.LastSeen = &t
		}
	}

	ds.Summary.UniqueCounterparties = len(counterparties)
}

func headerIndex(header []string) map[string]int {
	out := make(map[string]int, len(header))
	for i, h := range header {
		out[strings.TrimSpace(h)] = i
	}
	return out
}

func field(row []string, index map[string]int, key string) string {
	i, ok := index[key]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func parseBlockchairTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", s)
}

func parseInt64(s string) int64 {
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return n
		}
		n = n*10 + int64(ch-'0')
	}
	return n
}
func openMaybeGzip(path string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return &combinedReadCloser{
			Reader:  gz,
			closers: []io.Closer{gz, f},
		}, nil
	}

	return f, nil
}

type combinedReadCloser struct {
	io.Reader
	closers []io.Closer
}

func (c *combinedReadCloser) Close() error {
	var firstErr error
	for _, closer := range c.closers {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
