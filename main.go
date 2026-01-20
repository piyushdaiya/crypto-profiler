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
	// 1. Load .env file (Optional)
	// We check for error, but we don't crash. This allows the code to run
	// in Docker Compose (where env vars are injected directly) without a .env file.
	if err := godotenv.Load(); err != nil {
		// Only log if you really want to see it, otherwise silent is better for Docker
		// log.Println("No .env file found, relying on OS environment variables")
	}

	// 2. Input Validation
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./validator <address>")
	}
	address := strings.TrimSpace(os.Args[1])

	// 3. Load Keys (os.Getenv works for both .env files AND Docker Compose)
	etherscanKey := os.Getenv("ETHERSCAN_API_KEY")
	coinstatsKey := os.Getenv("COINSTATS_API_KEY")

	// 4. Register Strategies
	strategies := []validator.ChainStrategy{
		&validator.EVMStrategy{},     // Check EVM (0x...)
		&validator.BitcoinStrategy{}, // Check Bitcoin (Starts with 1, 3, bc1) <--- MOVED UP
		&validator.SolanaStrategy{},  // Check Solana (Generic Base58)         <--- MOVED DOWN
	}

	var result *validator.WalletProfile

	// 5. Run Strategy Matching
	for _, strategy := range strategies {
		if strategy.IsValidSyntax(address) {
			
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
			defer cancel()

			fmt.Printf("🔍 Analyzing %s on %s...\n", address, strategy.Name())
			
			// EVM Strategy calls Investigate() internally.
			// Others might not, so we handle that below.
			res, err := strategy.FetchState(ctx, address, configParam)
			if err != nil {
				log.Printf("⚠️ Error validating: %v", err)
			}
			
			// 6. Post-Process Safety Net
			// Ensure Sanctions check runs even if the strategy didn't call it.
			if res != nil && res.RiskScore == 0 && len(res.RiskReasons) == 0 {
				validator.Investigate(res, nil)
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
			ValidationDetails: "Invalid Format or No Matching Chain Strategy",
		}
	}

	// 7. Output Result
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}