package rpc

import (
	"encoding/json"
	"testing"

	"v12-handy-cap-bot/model"
)

func TestParseAssetsResponse(t *testing.T) {
	raw := `{
		"42161": [
			{
				"underlying": "ETH",
				"underlyingAssetAddress": "0xabc",
				"decimals": 18,
				"minTradeSize": "1",
				"maxTradeSize": "100",
				"symbol": "UETH",
				"chainId": 42161,
				"address": "0xdef",
				"active": true,
				"price": "2000",
				"multipliers": null
			}
		],
		"1": []
	}`

	var rawMap map[string][]*model.AssetsResponse
	if err := json.Unmarshal([]byte(raw), &rawMap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(rawMap) != 2 {
		t.Fatalf("expected 2 chains, got %d", len(rawMap))
	}
	if len(rawMap["42161"]) != 1 {
		t.Fatalf("expected 1 asset on chain 42161, got %d", len(rawMap["42161"]))
	}
	if rawMap["42161"][0].Symbol != "UETH" {
		t.Errorf("expected UETH, got %s", rawMap["42161"][0].Symbol)
	}
	if len(rawMap["1"]) != 0 {
		t.Errorf("expected 0 assets on chain 1, got %d", len(rawMap["1"]))
	}
}

func TestParseCapsResponse(t *testing.T) {
	raw := `[
		{"name": "GLOBAL", "type": "NOTIONAL", "cap": "4000", "usage": "1000"},
		{"name": "ETH", "type": "CONTRACTS", "cap": "500", "usage": "250"}
	]`

	var caps []model.SLCapsStatus
	if err := json.Unmarshal([]byte(raw), &caps); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(caps) != 2 {
		t.Fatalf("expected 2 caps, got %d", len(caps))
	}
	if caps[0].Name != "GLOBAL" {
		t.Errorf("expected GLOBAL, got %s", caps[0].Name)
	}
	if caps[1].Usage != "250" {
		t.Errorf("expected usage 250, got %s", caps[1].Usage)
	}
}

func TestParseErrorResponse(t *testing.T) {
	raw := `{"jsonrpc": "2.0", "id": "1", "error": {"code": -32600, "message": "Invalid request"}}`

	var resp model.JsonRPCResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("expected code -32600, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Invalid request" {
		t.Errorf("expected 'Invalid request', got %s", resp.Error.Message)
	}
}

func TestChainIDParsing_ValidKeys(t *testing.T) {
	raw := map[string][]*model.AssetsResponse{
		"42161": {{Symbol: "UETH"}},
		"1":     {{Symbol: "WETH"}},
	}

	assets := convertChainKeys(raw)

	if len(assets) != 2 {
		t.Fatalf("expected 2 chains, got %d", len(assets))
	}
	if len(assets[42161]) != 1 || assets[42161][0].Symbol != "UETH" {
		t.Error("expected UETH on chain 42161")
	}
	if len(assets[1]) != 1 || assets[1][0].Symbol != "WETH" {
		t.Error("expected WETH on chain 1")
	}
}

func TestChainIDParsing_InvalidKeys(t *testing.T) {
	raw := map[string][]*model.AssetsResponse{
		"abc":   {{Symbol: "BAD"}},
		"42161": {{Symbol: "GOOD"}},
		"":      {{Symbol: "EMPTY"}},
	}

	assets := convertChainKeys(raw)

	if len(assets) != 1 {
		t.Fatalf("expected 1 valid chain, got %d", len(assets))
	}
	if assets[42161][0].Symbol != "GOOD" {
		t.Error("expected GOOD on chain 42161")
	}
}

func TestGetAssets_ReturnsEmptyOnInit(t *testing.T) {
	c := &WSClient{
		assets: nil,
	}
	got := c.GetAssets()
	if got == nil {
		t.Fatal("expected non-nil map")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

func TestGetAssets_ReturnsCopy(t *testing.T) {
	c := &WSClient{
		assets: map[int][]*model.AssetsResponse{
			1: {{Symbol: "ETH"}},
		},
	}

	got := c.GetAssets()
	got[999] = []*model.AssetsResponse{{Symbol: "INJECTED"}}

	// Original should be unaffected
	if _, exists := c.assets[999]; exists {
		t.Error("GetAssets should return a copy, not the original map")
	}
}
