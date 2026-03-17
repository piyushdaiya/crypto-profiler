package datasets

import "time"

type Transfer struct {
	Chain        string    `json:"chain"`
	AssetType    string    `json:"asset_type"` // NATIVE, ERC20
	TxHash       string    `json:"tx_hash"`
	BlockID      int64     `json:"block_id"`
	Timestamp    time.Time `json:"timestamp"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	Amount       string    `json:"amount"`
	AssetAddress string    `json:"asset_address,omitempty"`
	AssetName    string    `json:"asset_name,omitempty"`
	AssetSymbol  string    `json:"asset_symbol,omitempty"`
	LabelFrom    string    `json:"label_from,omitempty"`
	LabelTo      string    `json:"label_to,omitempty"`
}

type Summary struct {
	FirstSeen            *time.Time `json:"first_seen,omitempty"`
	LastSeen             *time.Time `json:"last_seen,omitempty"`
	InboundCount         int        `json:"inbound_count"`
	OutboundCount        int        `json:"outbound_count"`
	UniqueCounterparties int        `json:"unique_counterparties"`
	NativeTransferCount  int        `json:"native_transfer_count"`
	ERC20TransferCount   int        `json:"erc20_transfer_count"`
}

type AddressDataset struct {
	Address     string     `json:"address"`
	Chain       string     `json:"chain"`
	Label       string     `json:"label,omitempty"`
	GeneratedAt time.Time  `json:"generated_at"`
	Summary     Summary    `json:"summary"`
	Transfers   []Transfer `json:"transfers"`
}
