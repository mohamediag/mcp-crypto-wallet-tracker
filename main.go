package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func main() {
	log.Println("Starting MCP Server...")

	apiKey, ok := os.LookupEnv("ETHERSCAN_API_KEY")
	if !ok || apiKey == "" {
		log.Fatal("ETHERSCAN_API_KEY environment variable is required")
	}

	walletTracker, err := NewWalletTracker(apiKey)
	if err != nil {
		log.Fatalf("Failed to initialize wallet tracker: %v", err)
	}

	// Start the HTTP server
	//startServer(walletTracker)

	// Initialize MCP server with stdio transport
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	// Register tools, prompts, and resources here...
	if err := registerWalletTracker(server, walletTracker); err != nil {
		log.Fatalf("Failed to register wallet tracker tool: %v", err)
	}

	// Start the server
	log.Println("MCP Server is now running and waiting for requests...")
	if err := server.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	select {}
}

type WalletTrackerRequest struct {
	WalletAddress string `json:"wallet_address" description:"The cryptocurrency wallet address to track"`
}

func registerWalletTracker(server *mcp_golang.Server, tracker *WalletTracker) error {
	// Register "wallet tracker" tool
	return server.RegisterTool("wallet_tracker", "Track the balance of a cryptocurrency wallet", func(req WalletTrackerRequest) (*mcp_golang.ToolResponse, error) {
		walletResp, err := tracker.GetWalletTokens(context.Background(), req.WalletAddress)
		if err != nil {
			return nil, err
		}

		content := formatWalletResponse(walletResp)
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(content)), nil
	})
}

func formatWalletResponse(resp *WalletResponse) string {
	if len(resp.Tokens) == 0 {
		return fmt.Sprintf("Wallet Address: %s\nNo token balances found.", resp.Address)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Wallet Address: %s\nTokens:\n", resp.Address))
	for _, token := range resp.Tokens {
		name := token.Name
		if name == "" {
			name = token.Address
		}
		if token.Symbol != "" {
			builder.WriteString(fmt.Sprintf("- %s (%s): %s\n", name, token.Symbol, token.Balance))
			continue
		}
		builder.WriteString(fmt.Sprintf("- %s: %s\n", name, token.Balance))
	}

	return strings.TrimRight(builder.String(), "\n")
}
