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
	"github.com/piyushdaiya/crypto-profiler/internal/validator"
)

func main() {
	_ = godotenv.Load()

	// 1. Get API Keys
	etherscanKey := os.Getenv("ETHERSCAN_API_KEY")
	coinstatsKey := os.Getenv("COINSTATS_API_KEY") // <--- NEW

	// 2. Register Strategies
	strategies := []validator.ChainStrategy{
		&validator.EVMStrategy{},     
		&validator.SolanaStrategy{},  
		&validator.BitcoinStrategy{}, 
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: ./validator <address>")
	}
	address := strings.TrimSpace(os.Args[1])

	var result *validator.WalletProfile

	for _, strategy := range strategies {
		if strategy.IsValidSyntax(address) {
			
			configParam := ""
			switch strategy.Name() {
			case "EVM (Etherscan)":
				configParam = etherscanKey
			case "SOLANA":
				configParam = coinstatsKey // <--- Use CoinStats Key here
			case "BITCOIN":
				configParam = "" // Blockchain.com (Free)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			res, err := strategy.FetchState(ctx, address, configParam)
			if err != nil {
				log.Printf("⚠️ Error validating with %s: %v", strategy.Name(), err)
			}
			result = res
			break
		}
	}

	if result == nil {
		result = &validator.WalletProfile{
			Address:           address,
			Network:           "UNKNOWN",
			IsValid:           false,
			ValidationDetails: "Invalid Format",
		}
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}