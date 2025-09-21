# MCP Crypto Wallet Tracker

A Model Context Protocol (MCP) server that provides cryptocurrency wallet tracking functionality using the Etherscan API.

## Features

- Track token balances for any Ethereum wallet address
- Support for ERC-20 tokens and other Ethereum-based assets
- Real-time balance calculation based on transaction history
- Clean, formatted output with token names, symbols, and balances
- Built as an MCP server for integration with Claude and other AI tools

## Prerequisites

- Go 1.19 or higher
- Etherscan API key

## Installation

1. Clone the repository:
```bash
git clone https://github.com/mohamediag/mcp-crypto-wallet-tracker.git
cd mcp-crypto-wallet-tracker
```

2. Install dependencies:
```bash
go mod tidy
```

3. Set up your Etherscan API key:
```bash
export ETHERSCAN_API_KEY="your_etherscan_api_key_here"
```

## Usage

### As MCP Server

Run the server:
```bash
go run .
```

The server will start and listen for MCP requests via stdio transport.

### Available Tools

#### wallet_tracker
Track the balance of a cryptocurrency wallet.

**Parameters:**
- `wallet_address` (string): The Ethereum wallet address to track

**Example:**
```json
{
  "wallet_address": "0x742b6BdCc5c2846E6c31b95A1DCE69e00C9fC7c6"
}
```

## Configuration

The server requires an `ETHERSCAN_API_KEY` environment variable. You can obtain a free API key from [Etherscan.io](https://etherscan.io/apis).

## API Response Format

The wallet tracker returns token information in the following format:

```
Wallet Address: 0x...
Tokens:
- Token Name (SYMBOL): balance
- Another Token (SYMBOL): balance
```

## Error Handling

- Invalid wallet addresses are rejected with appropriate error messages
- API rate limits and network errors are handled gracefully
- Empty wallets return a clean "No token balances found" message

## Dependencies

- [mcp-golang](https://github.com/metoro-io/mcp-golang) - MCP server implementation
- [gorilla/mux](https://github.com/gorilla/mux) - HTTP router (for future HTTP endpoint support)
