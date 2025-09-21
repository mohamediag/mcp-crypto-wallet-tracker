package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const (
	etherscanBaseURL   = "https://api.etherscan.io/api"
	defaultHTTPTimeout = 10 * time.Second
)

var (
	ErrInvalidWalletAddress = errors.New("invalid ethereum address")
	ErrNoTransactions       = errors.New("no token transactions found")
)

type WalletTracker struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func NewWalletTracker(apiKey string) (*WalletTracker, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("api key must not be empty")
	}

	return &WalletTracker{
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		baseURL: etherscanBaseURL,
		apiKey:  apiKey,
	}, nil
}

type TokenBalance struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Balance string `json:"balance"`
}

type WalletResponse struct {
	Address string         `json:"address"`
	Tokens  []TokenBalance `json:"tokens"`
}

func (t *WalletTracker) GetWalletTokens(ctx context.Context, walletAddress string) (*WalletResponse, error) {
	if err := validateWalletAddress(walletAddress); err != nil {
		return nil, err
	}

	txs, err := t.fetchTokenTransactions(ctx, walletAddress)
	if err != nil {
		if errors.Is(err, ErrNoTransactions) {
			return &WalletResponse{
				Address: walletAddress,
				Tokens:  []TokenBalance{},
			}, nil
		}
		return nil, err
	}

	tokens := summarizeTokenBalances(walletAddress, txs)
	return &WalletResponse{
		Address: walletAddress,
		Tokens:  tokens,
	}, nil
}

func (t *WalletTracker) fetchTokenTransactions(ctx context.Context, walletAddress string) ([]tokenTransaction, error) {
	endpoint, err := url.Parse(t.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing etherscan base URL: %w", err)
	}

	query := endpoint.Query()
	query.Set("module", "account")
	query.Set("action", "tokentx")
	query.Set("address", walletAddress)
	query.Set("startblock", "0")
	query.Set("endblock", "999999999")
	query.Set("sort", "asc")
	query.Set("apikey", t.apiKey)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating etherscan request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling etherscan: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("etherscan responded with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var apiResp etherscanResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding etherscan response: %w", err)
	}

	txs, err := apiResp.tokenTransactions()
	if err != nil {
		if errors.Is(err, ErrNoTransactions) {
			return nil, ErrNoTransactions
		}
		return nil, err
	}

	if apiResp.Status == "0" {
		if strings.EqualFold(apiResp.Message, "No transactions found") {
			return nil, ErrNoTransactions
		}
		return nil, fmt.Errorf("etherscan api error: %s", apiResp.Message)
	}

	return txs, nil
}

type etherscanResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

func (r etherscanResponse) tokenTransactions() ([]tokenTransaction, error) {
	if len(r.Result) == 0 {
		return []tokenTransaction{}, nil
	}

	var text string
	if err := json.Unmarshal(r.Result, &text); err == nil {
		if strings.EqualFold(text, "No transactions found") {
			return nil, ErrNoTransactions
		}
		return nil, fmt.Errorf("unexpected result text: %s", text)
	}

	var txs []tokenTransaction
	if err := json.Unmarshal(r.Result, &txs); err != nil {
		return nil, fmt.Errorf("parsing token transactions: %w", err)
	}
	return txs, nil
}

type tokenTransaction struct {
	ContractAddress  string `json:"contractAddress"`
	TokenName        string `json:"tokenName"`
	TokenNameAlt     string `json:"TokenName"`
	TokenSymbol      string `json:"tokenSymbol"`
	TokenSymbolAlt   string `json:"TokenSymbol"`
	TokenDecimal     string `json:"tokenDecimal"`
	TokenDecimalAlt  string `json:"TokenDecimal"`
	TokenQuantity    string `json:"value"`
	TokenQuantityAlt string `json:"TokenQuantity"`
	From             string `json:"from"`
	To               string `json:"to"`
}

func (t tokenTransaction) displayName() string {
	if t.TokenName != "" {
		return t.TokenName
	}
	if t.TokenNameAlt != "" {
		return t.TokenNameAlt
	}
	if sym := t.displaySymbol(); sym != "" {
		return sym
	}
	return t.ContractAddress
}

func (t tokenTransaction) displaySymbol() string {
	if t.TokenSymbol != "" {
		return t.TokenSymbol
	}
	if t.TokenSymbolAlt != "" {
		return t.TokenSymbolAlt
	}
	return ""
}

func (t tokenTransaction) decimals() int {
	if raw := firstNonEmpty(t.TokenDecimal, t.TokenDecimalAlt); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			return parsed
		}
	}
	return 0
}

func (t tokenTransaction) quantity() *big.Int {
	raw := firstNonEmpty(t.TokenQuantity, t.TokenQuantityAlt)
	if raw == "" {
		return nil
	}

	value, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil
	}
	return value
}

type tokenAggregate struct {
	address  string
	name     string
	symbol   string
	decimals int
	balance  *big.Int
}

func summarizeTokenBalances(walletAddress string, txs []tokenTransaction) []TokenBalance {
	if len(txs) == 0 {
		return []TokenBalance{}
	}

	wallet := strings.ToLower(walletAddress)
	aggregates := make(map[string]*tokenAggregate)

	for _, tx := range txs {
		qty := tx.quantity()
		if qty == nil {
			log.Printf("Skipping transaction with invalid quantity for contract %s", tx.ContractAddress)
			continue
		}

		agg, ok := aggregates[tx.ContractAddress]
		if !ok {
			agg = &tokenAggregate{
				address:  tx.ContractAddress,
				name:     tx.displayName(),
				symbol:   tx.displaySymbol(),
				decimals: tx.decimals(),
				balance:  big.NewInt(0),
			}
			aggregates[tx.ContractAddress] = agg
		}

		to := strings.ToLower(tx.To)
		from := strings.ToLower(tx.From)

		switch {
		case to == wallet:
			agg.balance.Add(agg.balance, qty)
		case from == wallet:
			agg.balance.Sub(agg.balance, qty)
		}
	}

	result := make([]TokenBalance, 0, len(aggregates))
	for _, agg := range aggregates {
		if agg.balance.Sign() == 0 {
			continue
		}
		result = append(result, TokenBalance{
			Address: agg.address,
			Name:    agg.name,
			Symbol:  agg.symbol,
			Balance: formatTokenBalance(agg.balance, agg.decimals),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result
}

func formatTokenBalance(balance *big.Int, decimals int) string {
	if balance == nil {
		return "0"
	}

	sign := ""
	value := new(big.Int).Set(balance)
	if value.Sign() < 0 {
		sign = "-"
		value.Abs(value)
	}

	if decimals <= 0 {
		return sign + value.String()
	}

	str := value.String()
	if len(str) <= decimals {
		str = strings.Repeat("0", decimals-len(str)+1) + str
	}

	split := len(str) - decimals
	intPart := str[:split]
	if intPart == "" {
		intPart = "0"
	}
	fracPart := strings.TrimRight(str[split:], "0")
	if fracPart == "" {
		return sign + intPart
	}
	return sign + intPart + "." + fracPart
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func validateWalletAddress(address string) error {
	if len(address) != 42 || !strings.HasPrefix(address, "0x") {
		return ErrInvalidWalletAddress
	}
	return nil
}

func walletHandler(tracker *WalletTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		walletAddress := vars["address"]

		if err := validateWalletAddress(walletAddress); err != nil {
			log.Printf("Invalid Ethereum address format received: %s", walletAddress)
			http.Error(w, "Invalid Ethereum address format. Expected 42 characters starting with 0x", http.StatusBadRequest)
			return
		}

		walletData, err := tracker.GetWalletTokens(r.Context(), walletAddress)
		if err != nil {
			if errors.Is(err, ErrNoTransactions) {
				walletData = &WalletResponse{Address: walletAddress, Tokens: []TokenBalance{}}
			} else if errors.Is(err, ErrInvalidWalletAddress) {
				http.Error(w, "Invalid Ethereum address format. Expected 42 characters starting with 0x", http.StatusBadRequest)
				return
			} else {
				log.Printf("Error fetching wallet data for address %s: %v", walletAddress, err)
				http.Error(w, "Failed to fetch wallet token data. Please try again later.", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(walletData); err != nil {
			log.Printf("Error encoding JSON response for address %s: %v", walletAddress, err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func setupRoutes(tracker *WalletTracker) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/wallet/{address}", walletHandler(tracker)).Methods("GET")
	return r
}

func startServer(tracker *WalletTracker) {
	router := setupRoutes(tracker)
	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
