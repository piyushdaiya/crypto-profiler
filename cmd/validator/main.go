package main

import (
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present. In Docker Compose, env vars are injected directly.
	_ = godotenv.Load()

	code := run(os.Args[1:], os.Stdout, os.Stderr, defaultStrategies())
	if code != 0 {
		os.Exit(code)
	}
}
