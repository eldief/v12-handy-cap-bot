//go:build integration

package main

import (
	"testing"

	"v12-handy-cap-bot/caps"
	"v12-handy-cap-bot/rpc"
)

const testWSURL = "wss://v12.rysk.finance/taker"

func TestComputeAllCapRatios_Integration(t *testing.T) {
	ws, err := rpc.NewWSClient(testWSURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer ws.Close()

	assets, err := ws.FetchAssets()
	if err != nil {
		t.Fatalf("fetch assets: %v", err)
	}

	capData, err := ws.FetchCaps()
	if err != nil {
		t.Fatalf("fetch caps: %v", err)
	}

	ratios, globalRatio := caps.ComputeAllCapRatios(capData, assets)

	t.Logf("Global ratio: %.2f%%", globalRatio)
	if globalRatio < 0 || globalRatio > 100 {
		t.Errorf("global ratio out of range: %.2f", globalRatio)
	}

	if len(ratios) == 0 {
		t.Fatal("expected at least one asset ratio")
	}

	for _, r := range ratios {
		dir := "Call"
		if r.IsPut {
			dir = "Put"
		}
		t.Logf("%-8s %-4s %.2f%%", r.Asset.Symbol, dir, r.Ratio)

		if r.Ratio < 0 {
			t.Errorf("%s %s: negative ratio %.2f", r.Asset.Symbol, dir, r.Ratio)
		}
	}
}
