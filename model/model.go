package model

import "encoding/json"

// JSON-RPC request
type JsonRPCRequest struct {
	JsonRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// JSON-RPC response wrapper
type JsonRPCResponse struct {
	JsonRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorData      `json:"error,omitempty"`
}

type ErrorData struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type SLCapsStatus struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Cap   string `json:"cap"`
	Usage string `json:"usage"`
}

type AssetsResponse struct {
	Underlying             string          `json:"underlying"`
	UnderlyingAssetAddress string          `json:"underlyingAssetAddress"`
	Decimals               uint8           `json:"decimals"`
	MinTradeSize           string          `json:"minTradeSize"`
	MaxTradeSize           string          `json:"maxTradeSize"`
	Symbol                 string          `json:"symbol"`
	ChainID                int             `json:"chainId"`
	Address                string          `json:"address"`
	Active                 bool            `json:"active"`
	Price                  string          `json:"price"`
	Multipliers            json.RawMessage `json:"multipliers"`
}

// Trade represents an open position from the "positions" RPC method.
type Trade struct {
	Address    string `json:"address"`
	APR        string `json:"apr"`
	ChainID    int    `json:"chainId"`
	CreatedAt  int    `json:"createdAt"`
	Expiry     int    `json:"expiry"`
	IsBuy      bool   `json:"isBuy"`
	IsPut      bool   `json:"isPut"`
	Quantity   string `json:"quantity"`
	Strike     string `json:"strike"`
	Price      string `json:"price"`
	TxHash     string `json:"txHash"`
	USD        string `json:"usd"`
	Collateral string `json:"collateral"`
	Fees       string `json:"fees"`
	Status     string `json:"status"`
	Symbol     string `json:"symbol"`
	Premium    string `json:"premium"`
}

// EnrichedTrade wraps a Trade with additional data resolved from assets.
type EnrichedTrade struct {
	Trade
	AssetSymbol      string
	Underlying       string
	CollateralSymbol string
	MarketPrice      string
}

// AssetCapRatio holds the computed cap usage ratio for an asset+direction.
type AssetCapRatio struct {
	Asset *AssetsResponse
	IsPut bool
	Ratio float64 // percentage (0-100)
}

// FreedCap represents a cap entry where usage decreased.
type FreedCap struct {
	Name     string
	Type     string
	OldUsage string
	NewUsage string
	Cap      string
}
