package main

import (
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

	// Start the server
	log.Println("MCP Server is now running and waiting for requests...")
	err := server.Serve()
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}

	select {} // Keeps the server running
}
