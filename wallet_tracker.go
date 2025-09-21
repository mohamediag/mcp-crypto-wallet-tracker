package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

const (
	etherscanAPIKey  = "8C1S8AXN7CRBGGUE2VT6493UPBI9ZCAT9W"
	etherscanBaseURL = "https://api.etherscan.io/api"

	// Token field names
	fieldContractAddress = "contractAddress"
	fieldTokenName       = "tokenName"
	fieldTokenNameAlt    = "TokenName"
	fieldTokenSymbol     = "tokenSymbol"
	fieldTokenSymbolAlt  = "TokenSymbol"
)

type TokenBalance struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type WalletResponse struct {
	Address string         `json:"address"`
	Tokens  []TokenBalance `json:"tokens"`
}

type TokenInfo struct {
	TokenName       string `json:"TokenName"`
	TokenSymbol     string `json:"TokenSymbol"`
	TokenQuantity   string `json:"TokenQuantity"`
	TokenDivisor    string `json:"TokenDivisor"`
	ContractAddress string `json:"contractAddress"`
	TokenDecimal    string `json:"TokenDecimal"`
}

type EtherscanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  any    `json:"result"`
}

func getWalletTokens(walletAddress string) (*WalletResponse, error) {
	etherscanResp, err := fetchTokenTransactions(walletAddress)
	if err != nil {
		return nil, err
	}

	tokens, err := processTokenTransactions(etherscanResp.Result)
	if err != nil {
		return nil, err
	}

	return &WalletResponse{
		Address: walletAddress,
		Tokens:  tokens,
	}, nil
}

func fetchTokenTransactions(walletAddress string) (*EtherscanResponse, error) {
	url := fmt.Sprintf("%s?module=account&action=tokentx&address=%s&startblock=0&endblock=999999999&sort=asc&apikey=%s",
		etherscanBaseURL, walletAddress, etherscanAPIKey)

	log.Printf("Fetching token transactions for wallet: %s", walletAddress)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token transactions from Etherscan: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Etherscan response body: %w", err)
	}

	var etherscanResp EtherscanResponse
	if err := json.Unmarshal(body, &etherscanResp); err != nil {
		return nil, fmt.Errorf("failed to parse Etherscan JSON response: %w", err)
	}

	if etherscanResp.Status == "0" {
		return nil, fmt.Errorf("Etherscan API returned error (status: %s): %s",
			etherscanResp.Status, etherscanResp.Message)
	}

	log.Printf("Successfully fetched token transactions, result type: %T", etherscanResp.Result)

	return &etherscanResp, nil
}

func processTokenTransactions(result any) ([]TokenBalance, error) {
	tokenMap := make(map[string]TokenBalance)

	switch result := result.(type) {
	case string:
		log.Printf("No token transactions found")
		return []TokenBalance{}, nil
	case []any:
		log.Printf("Processing %d token transactions", len(result))
		for _, item := range result {
			if tokenData, ok := item.(map[string]any); ok {
				token := createTokenFromData(tokenData)
				tokenMap[token.Address] = token
			}
		}
	default:
		log.Printf("Unexpected result type: %T", result)
		return nil, fmt.Errorf("unexpected result type from Etherscan API: %T, expected string or []any", result)
	}

	// Convert token map to slice
	var tokens []TokenBalance
	for _, token := range tokenMap {
		tokens = append(tokens, token)
	}

	log.Printf("Found %d unique tokens", len(tokens))
	return tokens, nil
}

func createTokenFromData(tokenData map[string]any) TokenBalance {
	contractAddress := getString(tokenData, fieldContractAddress)
	tokenName := getString(tokenData, fieldTokenName)
	tokenSymbol := getString(tokenData, fieldTokenSymbol)

	// Try alternative field names if standard ones are empty
	if tokenName == "" {
		tokenName = getString(tokenData, fieldTokenNameAlt)
	}
	if tokenSymbol == "" {
		tokenSymbol = getString(tokenData, fieldTokenSymbolAlt)
	}

	// Use symbol as name if name is still empty
	displayName := tokenName
	if displayName == "" && tokenSymbol != "" {
		displayName = tokenSymbol
	}

	log.Printf("Processing token: %s (%s)", contractAddress, displayName)

	return TokenBalance{
		Address: contractAddress,
		Name:    displayName,
	}
}

func getString(data map[string]any, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func walletHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	walletAddress := vars["address"]

	// Basic validation for Ethereum address
	if len(walletAddress) != 42 || !strings.HasPrefix(walletAddress, "0x") {
		log.Printf("Invalid Ethereum address format received: %s", walletAddress)
		http.Error(w, "Invalid Ethereum address format. Expected 42 characters starting with 0x", http.StatusBadRequest)
		return
	}

	walletData, err := getWalletTokens(walletAddress)
	if err != nil {
		log.Printf("Error fetching wallet data for address %s: %v", walletAddress, err)
		http.Error(w, "Failed to fetch wallet token data. Please try again later.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(walletData); err != nil {
		log.Printf("Error encoding JSON response for address %s: %v", walletAddress, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func setupRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/wallet/{address}", walletHandler).Methods("GET")
	return r
}

func startServer() {
	router := setupRoutes()
	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
