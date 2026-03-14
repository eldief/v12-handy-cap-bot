//go:build integration

package rpc

import (
	"testing"
)

const testWSURL = "wss://v12.rysk.finance/taker"

func TestFetchCaps(t *testing.T) {
	ws, err := NewWSClient(testWSURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer ws.Close()

	caps, err := ws.FetchCaps()
	if err != nil {
		t.Fatalf("fetch caps: %v", err)
	}

	if len(caps) == 0 {
		t.Fatal("expected at least one cap, got none")
	}

	for _, c := range caps {
		if c.Name == "" {
			t.Error("cap has empty name")
		}
		if c.Type == "" {
			t.Error("cap has empty type")
		}
		if c.Cap == "" {
			t.Errorf("cap %q has empty cap value", c.Name)
		}
		if c.Usage == "" {
			t.Errorf("cap %q has empty usage value", c.Name)
		}
		t.Logf("cap: name=%s type=%s cap=%s usage=%s", c.Name, c.Type, c.Cap, c.Usage)
	}
}

func TestFetchAssets(t *testing.T) {
	ws, err := NewWSClient(testWSURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer ws.Close()

	assets, err := ws.FetchAssets()
	if err != nil {
		t.Fatalf("fetch assets: %v", err)
	}

	if len(assets) == 0 {
		t.Fatal("expected at least one chain, got none")
	}

	for chainID, list := range assets {
		if len(list) == 0 {
			t.Errorf("chain %d has no assets", chainID)
			continue
		}
		for _, a := range list {
			if a.Symbol == "" {
				t.Errorf("chain %d: asset has empty symbol", chainID)
			}
			t.Logf("chain=%d symbol=%s underlying=%s active=%v price=%s",
				chainID, a.Symbol, a.Underlying, a.Active, a.Price)
		}
	}
}
