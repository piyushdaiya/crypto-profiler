package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/piyushdaiya/crypto-profiler/internal/address"
	"github.com/piyushdaiya/crypto-profiler/internal/analyzer"
	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func main() {
	// Load .env if present. In Docker Compose, env vars are injected directly.
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		log.Fatal("Usage: ./validator <wallet-address>")
	}

	walletAddress := strings.TrimSpace(os.Args[1])

	etherscanKey := os.Getenv("ETHERSCAN_API_KEY")
	coinstatsKey := os.Getenv("COINSTATS_API_KEY")

	strategies := []address.ChainStrategy{
		&address.EVMStrategy{},
		&address.BitcoinStrategy{},
		&address.SolanaStrategy{},
	}

	var result *model.WalletProfile

	for _, strategy := range strategies {
		if !strategy.IsValidSyntax(walletAddress) {
			continue
		}

		configParam := ""
		switch strategy.Name() {
		case "EVM (Etherscan)":
			configParam = etherscanKey
		case "SOLANA":
			configParam = coinstatsKey
		case "BITCOIN":
			configParam = ""
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

		fmt.Printf("🔍 Analyzing %s on %s...\n", walletAddress, strategy.Name())

		res, err := strategy.FetchState(ctx, walletAddress, configParam)
		cancel()

		if err != nil {
			log.Printf("⚠️ Error validating: %v", err)
		}

		// Safety net: run analysis if the selected strategy did not do so.
		if res != nil && res.RiskScore == 0 && len(res.RiskReasons) == 0 {
			analyzer.Investigate(res, nil)
		}

		result = res
		break
	}

	if result == nil {
		result = &model.WalletProfile{
			Address:           walletAddress,
			Network:           "UNKNOWN",
			IsValid:           false,
			ValidationDetails: "Invalid format or no matching chain strategy",
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(result); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}
