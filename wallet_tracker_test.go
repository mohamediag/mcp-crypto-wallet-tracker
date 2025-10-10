package main

import (
	"context"
	"os"
	"testing"
)

func TestGetWalletTokens(t *testing.T) {
	apiKey := os.Getenv("ETHERSCAN_API_KEY")
	if apiKey == "" {
		t.Skip("ETHERSCAN_API_KEY not set, skipping test")
	}

	tracker, err := NewWalletTracker(apiKey)
	if err != nil {
		t.Fatalf("Failed to create wallet tracker: %v", err)
	}

	// Test with the provided address
	walletAddress := "0xab66485175E65993F217B7470EA433574473A760"

	ctx := context.Background()
	resp, err := tracker.GetWalletTokens(ctx, walletAddress)
	if err != nil {
		t.Fatalf("Failed to get wallet tokens: %v", err)
	}

	t.Logf("Wallet Address: %s", resp.Address)
	t.Logf("Number of tokens: %d", len(resp.Tokens))

	for _, token := range resp.Tokens {
		t.Logf("  - %s (%s): %s (Contract: %s)", token.Name, token.Symbol, token.Balance, token.Address)
	}
}
