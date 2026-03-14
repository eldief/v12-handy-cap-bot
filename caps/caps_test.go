package caps

import (
	"math"
	"math/big"
	"testing"
	"time"

	"v12-handy-cap-bot/model"
)

var testAsset = &model.AssetsResponse{
	Symbol:     "UETH",
	Underlying: "ETH",
	Address:    "0xbe6727b535545c67d5caa73dea54865b92cf7907",
	Active:     true,
	Price:      "2000000000000000000000",
}

var testCaps = []model.SLCapsStatus{
	{Name: "GLOBAL", Type: "NOTIONAL", Cap: "4000000000000000000000000", Usage: "1000000000000000000000000"},
	{Name: "ETH", Type: "CONTRACTS", Cap: "500000000000000000000", Usage: "250000000000000000000"},
	{Name: "ETH", Type: "NOTIONAL", Cap: "1400000000000000000000000", Usage: "700000000000000000000000"},
	{Name: "0xbe6727b535545c67d5caa73dea54865b92cf7907", Type: "CONTRACTS", Cap: "500000000000000000000", Usage: "100000000000000000000"},
	{Name: "0xbe6727b535545c67d5caa73dea54865b92cf7907", Type: "NOTIONAL", Cap: "1400000000000000000000000", Usage: "280000000000000000000000"},
	{Name: "0xbe6727b535545c67d5caa73dea54865b92cf7907-false", Type: "CONTRACTS", Cap: "500000000000000000000", Usage: "50000000000000000000"},
	{Name: "0xbe6727b535545c67d5caa73dea54865b92cf7907-false", Type: "NOTIONAL", Cap: "1400000000000000000000000", Usage: "140000000000000000000000"},
	{Name: "0xbe6727b535545c67d5caa73dea54865b92cf7907-true", Type: "CONTRACTS", Cap: "500000000000000000000000", Usage: "100000000000000000000"},
	{Name: "0xbe6727b535545c67d5caa73dea54865b92cf7907-true", Type: "NOTIONAL", Cap: "500000000000000000000000", Usage: "200000000000000000000000"},
}

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

// --- findCap ---

func TestFindCap_MatchesNameAndType(t *testing.T) {
	c := findCap(testCaps, "ETH", "CONTRACTS")
	if c == nil {
		t.Fatal("expected to find ETH CONTRACTS cap")
	}
	if c.Cap != "500000000000000000000" {
		t.Errorf("unexpected cap value: %s", c.Cap)
	}
}

func TestFindCap_MatchesNameOnly(t *testing.T) {
	c := findCap(testCaps, "GLOBAL", "")
	if c == nil {
		t.Fatal("expected to find GLOBAL cap")
	}
}

func TestFindCap_NotFound(t *testing.T) {
	c := findCap(testCaps, "NONEXISTENT", "CONTRACTS")
	if c != nil {
		t.Error("expected nil for nonexistent cap")
	}
}

// --- relevantCaps ---

func TestRelevantCaps_Call(t *testing.T) {
	entries := relevantCaps(testCaps, testAsset, false)
	if len(entries) != 7 {
		t.Errorf("expected 7 entries for call, got %d", len(entries))
	}
}

func TestRelevantCaps_Put(t *testing.T) {
	entries := relevantCaps(testCaps, testAsset, true)
	if len(entries) != 7 {
		t.Errorf("expected 7 entries for put, got %d", len(entries))
	}
}

func TestRelevantCaps_FiltersZeroCap(t *testing.T) {
	caps := []model.SLCapsStatus{
		{Name: "GLOBAL", Type: "NOTIONAL", Cap: "0", Usage: "0"},
		{Name: "ETH", Type: "CONTRACTS", Cap: "100", Usage: "50"},
	}
	entries := relevantCaps(caps, testAsset, false)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (GLOBAL filtered out), got %d", len(entries))
	}
}

// --- GetCapUsageRatio ---

func TestGetCapUsageRatio_NilInputs(t *testing.T) {
	if r := GetCapUsageRatio(nil, testAsset, false); r != 0 {
		t.Errorf("expected 0 for nil caps, got %f", r)
	}
	if r := GetCapUsageRatio(testCaps, nil, false); r != 0 {
		t.Errorf("expected 0 for nil asset, got %f", r)
	}
}

func TestGetCapUsageRatio_ReturnsMaxRatio(t *testing.T) {
	ratio := GetCapUsageRatio(testCaps, testAsset, false)
	if !almostEqual(ratio, 50.0, 0.01) {
		t.Errorf("expected ~50%%, got %.2f%%", ratio)
	}
}

func TestGetCapUsageRatio_Put(t *testing.T) {
	ratio := GetCapUsageRatio(testCaps, testAsset, true)
	if !almostEqual(ratio, 50.0, 0.01) {
		t.Errorf("expected ~50%%, got %.2f%%", ratio)
	}
}

func TestGetCapUsageRatio_FullUsage(t *testing.T) {
	caps := []model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "1000"},
	}
	ratio := GetCapUsageRatio(caps, testAsset, false)
	if !almostEqual(ratio, 100.0, 0.01) {
		t.Errorf("expected 100%%, got %.2f%%", ratio)
	}
}

func TestGetCapUsageRatio_ZeroUsage(t *testing.T) {
	caps := []model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "0"},
	}
	ratio := GetCapUsageRatio(caps, testAsset, false)
	if ratio != 0 {
		t.Errorf("expected 0%%, got %.2f%%", ratio)
	}
}

// --- GetGlobalCapUsageRatio ---

func TestGetGlobalCapUsageRatio_Normal(t *testing.T) {
	ratio := GetGlobalCapUsageRatio(testCaps)
	if !almostEqual(ratio, 25.0, 0.01) {
		t.Errorf("expected ~25%%, got %.2f%%", ratio)
	}
}

func TestGetGlobalCapUsageRatio_NoCaps(t *testing.T) {
	if r := GetGlobalCapUsageRatio(nil); r != 0 {
		t.Errorf("expected 0, got %f", r)
	}
}

func TestGetGlobalCapUsageRatio_NoGlobalEntry(t *testing.T) {
	caps := []model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "100", Usage: "50"},
	}
	if r := GetGlobalCapUsageRatio(caps); r != 0 {
		t.Errorf("expected 0, got %f", r)
	}
}

func TestGetGlobalCapUsageRatio_ZeroCap(t *testing.T) {
	caps := []model.SLCapsStatus{
		{Name: "GLOBAL", Type: "NOTIONAL", Cap: "0", Usage: "100"},
	}
	if r := GetGlobalCapUsageRatio(caps); r != 0 {
		t.Errorf("expected 0 for zero cap, got %f", r)
	}
}

// --- ComputeAllCapRatios ---

func TestComputeAllCapRatios_Basic(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{
		999: {testAsset},
	}
	ratios, globalRatio := ComputeAllCapRatios(testCaps, assets)

	if !almostEqual(globalRatio, 25.0, 0.01) {
		t.Errorf("expected global ~25%%, got %.2f%%", globalRatio)
	}

	if len(ratios) != 2 {
		t.Errorf("expected 2 ratios, got %d", len(ratios))
	}

	for _, r := range ratios {
		if r.Asset.Symbol != "UETH" {
			t.Errorf("unexpected asset symbol: %s", r.Asset.Symbol)
		}
	}
}

func TestComputeAllCapRatios_SkipsInactive(t *testing.T) {
	inactive := &model.AssetsResponse{
		Symbol:     "DEAD",
		Underlying: "DEAD",
		Address:    "0xdead",
		Active:     false,
	}
	assets := map[int][]*model.AssetsResponse{
		999: {testAsset, inactive},
	}
	ratios, _ := ComputeAllCapRatios(testCaps, assets)

	for _, r := range ratios {
		if r.Asset.Symbol == "DEAD" {
			t.Error("inactive asset should be skipped")
		}
	}
}

func TestComputeAllCapRatios_DeduplicatesSameAddress(t *testing.T) {
	dup := *testAsset
	dup.Symbol = "UETH2"
	assets := map[int][]*model.AssetsResponse{
		999: {testAsset, &dup},
	}
	ratios, _ := ComputeAllCapRatios(testCaps, assets)

	if len(ratios) != 2 {
		t.Errorf("expected 2 ratios (deduplicated), got %d", len(ratios))
	}
}

func TestComputeAllCapRatios_EmptyInputs(t *testing.T) {
	ratios, globalRatio := ComputeAllCapRatios(nil, nil)
	if len(ratios) != 0 {
		t.Errorf("expected 0 ratios, got %d", len(ratios))
	}
	if globalRatio != 0 {
		t.Errorf("expected 0 global ratio, got %f", globalRatio)
	}
}

// --- FindAssetsByName ---

func TestFindAssetsByName_ByUnderlying(t *testing.T) {
	other := &model.AssetsResponse{
		Symbol: "WETH", Underlying: "ETH", Address: "0xother", Active: true,
	}
	inactive := &model.AssetsResponse{
		Symbol: "DEAD", Underlying: "ETH", Address: "0xdead", Active: false,
	}
	assets := map[int][]*model.AssetsResponse{999: {testAsset, other, inactive}}

	found := FindAssetsByName(assets, "ETH")
	if len(found) != 2 {
		t.Errorf("expected 2 active assets matching ETH underlying, got %d", len(found))
	}
}

func TestFindAssetsByName_BySymbol(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{999: {testAsset}}

	found := FindAssetsByName(assets, "UETH")
	if len(found) != 1 || found[0].Symbol != "UETH" {
		t.Error("expected to find UETH by symbol")
	}
}

func TestFindAssetsByName_NotFound(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{999: {testAsset}}

	found := FindAssetsByName(assets, "NONEXISTENT")
	if len(found) != 0 {
		t.Errorf("expected 0 results, got %d", len(found))
	}
}

// --- FreedCapsTracker ---

func TestTracker_FirstUpdateNoAlert(t *testing.T) {
	tracker := NewFreedCapsTracker(0) // 0 debounce for instant tests
	caps := []model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "500"},
	}
	freed := tracker.Update(caps)
	if len(freed) != 0 {
		t.Errorf("expected no alerts on first update, got %d", len(freed))
	}
}

func TestTracker_DebounceWaitsBeforeAlert(t *testing.T) {
	tracker := NewFreedCapsTracker(50 * time.Millisecond)

	tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "500"},
	})

	// Decrease — should NOT alert yet (debouncing)
	freed := tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "300"},
	})
	if len(freed) != 0 {
		t.Errorf("expected 0 during debounce, got %d", len(freed))
	}

	// Wait for debounce to expire
	time.Sleep(60 * time.Millisecond)

	// Same value — debounce expired, should fire now
	freed = tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "300"},
	})
	if len(freed) != 1 {
		t.Fatalf("expected 1 after debounce, got %d", len(freed))
	}
	if freed[0].OldUsage != "500" || freed[0].NewUsage != "300" {
		t.Errorf("unexpected freed: %+v", freed[0])
	}
}

func TestTracker_FurtherDecreaseResetsDebounce(t *testing.T) {
	tracker := NewFreedCapsTracker(50 * time.Millisecond)

	tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "500"},
	})

	// First decrease
	tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "400"},
	})

	time.Sleep(30 * time.Millisecond)

	// Further decrease before debounce expires — resets timer
	freed := tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "200"},
	})
	if len(freed) != 0 {
		t.Errorf("expected 0 during debounce reset, got %d", len(freed))
	}

	time.Sleep(60 * time.Millisecond)

	// Debounce expired — should fire with original start and final value
	freed = tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "200"},
	})
	if len(freed) != 1 {
		t.Fatalf("expected 1, got %d", len(freed))
	}
	if freed[0].OldUsage != "500" || freed[0].NewUsage != "200" {
		t.Errorf("expected 500->200, got %s->%s", freed[0].OldUsage, freed[0].NewUsage)
	}
}

func TestTracker_IncreaseResetsDebounce(t *testing.T) {
	tracker := NewFreedCapsTracker(0)

	tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "500"},
	})

	// Decrease — fires immediately with 0 debounce
	freed := tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "300"},
	})
	if len(freed) != 1 {
		t.Fatalf("expected 1 on decrease, got %d", len(freed))
	}

	// Increase — no alert
	freed = tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "600"},
	})
	if len(freed) != 0 {
		t.Errorf("expected 0 on increase, got %d", len(freed))
	}

	// New decrease — should fire again
	freed = tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "400"},
	})
	if len(freed) != 1 {
		t.Errorf("expected 1 after reset+decrease, got %d", len(freed))
	}
}

func TestTracker_IncreasedUsageNoAlert(t *testing.T) {
	tracker := NewFreedCapsTracker(0)

	tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "300"},
	})

	freed := tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "500"},
	})
	if len(freed) != 0 {
		t.Errorf("expected 0 for increased usage, got %d", len(freed))
	}
}

func TestTracker_ZeroDebounceFiresImmediately(t *testing.T) {
	tracker := NewFreedCapsTracker(0)

	tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "500"},
	})

	// Decrease — with 0 debounce, fires in same Update call
	freed := tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "300"},
	})
	if len(freed) != 1 {
		t.Fatalf("expected 1 immediately with 0 debounce, got %d", len(freed))
	}

	// No more alerts for same value
	freed = tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "300"},
	})
	if len(freed) != 0 {
		t.Errorf("expected 0 after already fired, got %d", len(freed))
	}
}

// --- FindAssetByAddress / FindAssetByUnderlying ---

func TestFindAssetByAddress(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{999: {testAsset}}

	a := FindAssetByAddress(assets, testAsset.Address)
	if a == nil || a.Symbol != "UETH" {
		t.Error("expected to find UETH by address")
	}

	a = FindAssetByAddress(assets, "0xBE6727B535545C67D5CAA73DEA54865B92CF7907")
	if a == nil || a.Symbol != "UETH" {
		t.Error("expected case-insensitive match")
	}

	a = FindAssetByAddress(assets, "0xnonexistent")
	if a != nil {
		t.Error("expected nil for nonexistent address")
	}
}

func TestFindAssetByUnderlying(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{999: {testAsset}}

	a := FindAssetByUnderlying(assets, "ETH")
	if a == nil || a.Symbol != "UETH" {
		t.Error("expected to find UETH by underlying")
	}

	a = FindAssetByUnderlying(assets, "NONEXISTENT")
	if a != nil {
		t.Error("expected nil for nonexistent underlying")
	}
}

// --- Additional edge case tests ---

func TestTracker_InvalidUsageString(t *testing.T) {
	tracker := NewFreedCapsTracker(0)

	// Non-numeric usage should be silently skipped
	caps := []model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "not_a_number"},
	}
	freed := tracker.Update(caps)
	if len(freed) != 0 {
		t.Errorf("expected 0 for invalid usage, got %d", len(freed))
	}

	// Subsequent valid update should work without prior state pollution
	freed = tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "500"},
	})
	if len(freed) != 0 {
		t.Errorf("expected 0 (first valid update), got %d", len(freed))
	}
}

func TestTracker_EmptyCapsSlice(t *testing.T) {
	tracker := NewFreedCapsTracker(0)

	// Seed some state
	tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "500"},
	})

	// Empty update should not lose pending or panic
	freed := tracker.Update(nil)
	if len(freed) != 0 {
		t.Errorf("expected 0, got %d", len(freed))
	}

	// Previous state should still be intact
	freed = tracker.Update([]model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "300"},
	})
	if len(freed) != 1 {
		t.Errorf("expected 1 after decrease, got %d", len(freed))
	}
}

func TestRelevantCaps_NilAssetFields(t *testing.T) {
	// Asset with empty fields — should still work, just find fewer matches
	asset := &model.AssetsResponse{
		Symbol:     "",
		Underlying: "",
		Address:    "",
		Active:     true,
	}
	caps := []model.SLCapsStatus{
		{Name: "GLOBAL", Type: "NOTIONAL", Cap: "1000", Usage: "500"},
	}
	entries := relevantCaps(caps, asset, false)
	if len(entries) != 1 {
		t.Errorf("expected 1 (GLOBAL only), got %d", len(entries))
	}
}

func TestGetCapUsageRatio_MalformedCapUsage(t *testing.T) {
	caps := []model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "not_a_number", Usage: "500"},
	}
	ratio := GetCapUsageRatio(caps, testAsset, false)
	if ratio != 0 {
		t.Errorf("expected 0 for malformed cap, got %f", ratio)
	}

	caps = []model.SLCapsStatus{
		{Name: "ETH", Type: "CONTRACTS", Cap: "1000", Usage: "not_a_number"},
	}
	ratio = GetCapUsageRatio(caps, testAsset, false)
	if ratio != 0 {
		t.Errorf("expected 0 for malformed usage, got %f", ratio)
	}
}

func TestTracker_MalformedKeyNoPipe(t *testing.T) {
	tracker := NewFreedCapsTracker(0)

	// Manually inject a pending entry with a key that has no pipe
	tracker.lastUsage["nopipe"] = big.NewInt(500)
	tracker.pending["nopipe"] = &pendingFree{
		startUsage: big.NewInt(500),
		lastUsage:  big.NewInt(300),
		cap:        "1000",
		lastSeen:   time.Now().Add(-time.Hour),
	}

	// Update should not panic; the malformed key should be cleaned up
	freed := tracker.Update(nil)
	if len(freed) != 0 {
		t.Errorf("expected 0 for malformed key, got %d", len(freed))
	}

	// pending should have been cleaned up
	if _, exists := tracker.pending["nopipe"]; exists {
		t.Error("expected malformed key to be deleted from pending")
	}
}
