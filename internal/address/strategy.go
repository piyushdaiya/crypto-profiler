package address

import (
	"context"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type ChainStrategy interface {
	Name() string
	IsValidSyntax(address string) bool
	FetchState(ctx context.Context, address string, apiKey string) (*model.WalletProfile, error)
}
