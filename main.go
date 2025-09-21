package main

import (
	"fmt"
	"log"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func main() {
	// Start the HTTP server
	//startServer()
	log.Println("Starting MCP Server...")

	// Initialize MCP server with stdio transport
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	// Register tools, prompts, and resources here...
	registerWalletTracker(server)

	// Start the server
	log.Println("MCP Server is now running and waiting for requests...")
	err := server.Serve()
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}

	select {} // Keeps the server running
}

type WalletTrackerRequest struct {
	WalletAddress string `json:"wallet_address" description:"The cryptocurrency wallet address to track"`
}

func registerWalletTracker(server *mcp_golang.Server) {
	// Register "wallet tracker" tool
	err := server.RegisterTool("wallet_tracker", "Track the balance of a cryptocurrency wallet", func(req WalletTrackerRequest) (*mcp_golang.ToolResponse, error) {
		walletResp, err := getWalletTokens(req.WalletAddress)
		if err != nil {
			return nil, err
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Wallet Address: %s\nTokens: %+v", walletResp.Address, walletResp.Tokens))), nil
	})

	if err != nil {
		log.Fatalf("Failed to register wallet tracker tool: %v", err)
	}
}
