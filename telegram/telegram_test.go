package telegram

import (
	"strings"
	"testing"

	"v12-handy-cap-bot/model"
)

var testAsset = &model.AssetsResponse{
	Symbol:     "UETH",
	Underlying: "ETH",
	Address:    "0xbe6727b535545c67d5caa73dea54865b92cf7907",
	Active:     true,
	Price:      "2000000000000000000000",
}

// HYPE underlying with multiple assets — mirrors real data.
var (
	whypeAsset = &model.AssetsResponse{
		Symbol:     "WHYPE",
		Underlying: "HYPE",
		Address:    "0xwhype0000000000000000000000000000000001",
		Active:     true,
		Price:      "1000000000000000000",
	}
	khypeAsset = &model.AssetsResponse{
		Symbol:     "KHYPE",
		Underlying: "HYPE",
		Address:    "0xkhype0000000000000000000000000000000002",
		Active:     true,
		Price:      "1000000000000000000",
	}
	lhypeAsset = &model.AssetsResponse{
		Symbol:     "LHYPE",
		Underlying: "HYPE",
		Address:    "0xlhype0000000000000000000000000000000003",
		Active:     true,
		Price:      "1000000000000000000",
	}
	hypeAssets = map[int][]*model.AssetsResponse{
		999: {whypeAsset, khypeAsset, lhypeAsset},
	}
)

func TestFormatCapRatios_Empty(t *testing.T) {
	result := FormatCapRatios(nil)
	if result != "" {
		t.Error("expected empty string for nil ratios")
	}
}

func TestFormatCapRatios_Basic(t *testing.T) {
	ratios := []model.AssetCapRatio{
		{Asset: testAsset, IsPut: false, Ratio: 50.0},
		{Asset: testAsset, IsPut: true, Ratio: 40.0},
	}
	result := FormatCapRatios(ratios)

	if !strings.Contains(result, "Rysk v12 Caps") {
		t.Error("expected header")
	}
	if !strings.Contains(result, "50.00%") {
		t.Error("expected call ratio")
	}
	if !strings.Contains(result, "40.00%") {
		t.Error("expected put ratio")
	}
	if !strings.Contains(result, "Call") {
		t.Error("expected Call direction")
	}
	if !strings.Contains(result, "Put") {
		t.Error("expected Put direction")
	}
}

func TestEscMD(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"hello", "hello"},
		{"hello_world", "hello\\_world"},
		{"1.5", "1\\.5"},
		{"a*b", "a\\*b"},
		{"(test)", "\\(test\\)"},
		{"a|b", "a\\|b"},
	}
	for _, tt := range tests {
		got := EscMD(tt.input)
		if got != tt.expected {
			t.Errorf("EscMD(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsChatGone(t *testing.T) {
	tests := []struct {
		msg  string
		gone bool
	}{
		{"Forbidden: bot was blocked by the user", true},
		{"Bad Request: chat not found", true},
		{"Forbidden: bot was kicked from the group", true},
		{"timeout", false},
		{"rate limit exceeded", false},
	}
	for _, tt := range tests {
		err := &testError{msg: tt.msg}
		if got := isChatGone(err); got != tt.gone {
			t.Errorf("isChatGone(%q) = %v, want %v", tt.msg, got, tt.gone)
		}
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// --- FormatFreedCaps ---

func TestFormatFreedCaps_Empty(t *testing.T) {
	result := FormatFreedCaps(nil, nil, nil)
	if result != "" {
		t.Error("expected empty string for nil freed caps")
	}

	result = FormatFreedCaps([]model.FreedCap{}, nil, nil)
	if result != "" {
		t.Error("expected empty string for empty freed caps")
	}
}

func TestFormatFreedCaps_ResolvesAssetName(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{
		999: {testAsset},
	}
	capData := []model.SLCapsStatus{
		{Name: testAsset.Address, Type: "CONTRACTS", Cap: "1000", Usage: "300"},
	}
	freed := []model.FreedCap{
		{Name: testAsset.Address, Type: "CONTRACTS", OldUsage: "500", NewUsage: "300", Cap: "1000"},
	}
	result := FormatFreedCaps(freed, capData, assets)

	if !strings.Contains(result, "Cap Freed") {
		t.Error("expected header")
	}
	if !strings.Contains(result, "UETH") {
		t.Error("expected address resolved to UETH")
	}
	if strings.Contains(result, testAsset.Address) {
		t.Error("expected raw address to be replaced by symbol")
	}
	if !strings.Contains(result, "Call") {
		t.Error("expected Call direction")
	}
}

func TestFormatFreedCaps_DirectionSuffix(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{
		999: {testAsset},
	}
	capData := []model.SLCapsStatus{
		{Name: testAsset.Address + "-false", Type: "NOTIONAL", Cap: "200", Usage: "50"},
		{Name: testAsset.Address + "-true", Type: "CONTRACTS", Cap: "200", Usage: "40"},
	}
	freed := []model.FreedCap{
		{Name: testAsset.Address + "-false", Type: "NOTIONAL", OldUsage: "100", NewUsage: "50", Cap: "200"},
		{Name: testAsset.Address + "-true", Type: "CONTRACTS", OldUsage: "80", NewUsage: "40", Cap: "200"},
	}
	result := FormatFreedCaps(freed, capData, assets)

	if !strings.Contains(result, "Call") {
		t.Error("expected Call direction for -false suffix")
	}
	if !strings.Contains(result, "Put") {
		t.Error("expected Put direction for -true suffix")
	}
	if !strings.Contains(result, "UETH") {
		t.Error("expected address resolved to UETH")
	}
}

func TestFormatFreedCaps_UnknownAddress(t *testing.T) {
	freed := []model.FreedCap{
		{Name: "0xunknown", Type: "CONTRACTS", OldUsage: "500", NewUsage: "300", Cap: "1000"},
	}
	result := FormatFreedCaps(freed, nil, nil)

	if result != "" {
		t.Error("expected empty string when asset not found")
	}
}

func TestFormatFreedCaps_ResolvesUnderlying(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{
		999: {testAsset},
	}
	capData := []model.SLCapsStatus{
		{Name: "ETH", Type: "NOTIONAL", Cap: "1000", Usage: "250"},
	}
	freed := []model.FreedCap{
		{Name: "ETH", Type: "NOTIONAL", OldUsage: "500", NewUsage: "250", Cap: "1000"},
	}
	result := FormatFreedCaps(freed, capData, assets)

	if !strings.Contains(result, "UETH") {
		t.Error("expected underlying resolved to UETH symbol")
	}
}

func TestFormatFreedCaps_DeduplicatesSameAsset(t *testing.T) {
	assets := map[int][]*model.AssetsResponse{
		999: {testAsset},
	}
	capData := []model.SLCapsStatus{
		{Name: testAsset.Address, Type: "CONTRACTS", Cap: "1000", Usage: "300"},
		{Name: testAsset.Address, Type: "NOTIONAL", Cap: "2000", Usage: "600"},
	}
	freed := []model.FreedCap{
		{Name: testAsset.Address, Type: "CONTRACTS", OldUsage: "500", NewUsage: "300", Cap: "1000"},
		{Name: testAsset.Address, Type: "NOTIONAL", OldUsage: "1000", NewUsage: "600", Cap: "2000"},
	}
	result := FormatFreedCaps(freed, capData, assets)

	// Should only appear once (deduplicated by address+direction)
	count := strings.Count(result, "UETH")
	if count != 1 {
		t.Errorf("expected 1 UETH entry (deduplicated), got %d", count)
	}
}

// --- FormatSingleCapRatio ---

func TestFormatSingleCapRatio_EmptyRatios(t *testing.T) {
	result := FormatSingleCapRatio("ETH", nil)
	if !strings.Contains(result, "No cap data") {
		t.Errorf("expected 'No cap data', got %s", result)
	}
}

func TestFormatSingleCapRatio_EmptyName(t *testing.T) {
	ratios := []model.AssetCapRatio{
		{Asset: testAsset, IsPut: false, Ratio: 50.0},
	}
	result := FormatSingleCapRatio("", ratios)
	if !strings.Contains(result, "Cap:") {
		t.Error("expected header even with empty name")
	}
	if !strings.Contains(result, "50.00%") {
		t.Error("expected ratio value")
	}
}

// --- Multi-asset underlying (HYPE) freed cap tests ---
// These replicate real scenarios where an underlying has multiple assets
// (WHYPE, KHYPE, LHYPE) and caps fire at different layers.

// Layer 1: GLOBAL cap freed — should show all assets, both directions.
func TestFormatFreedCaps_GlobalCap(t *testing.T) {
	capData := []model.SLCapsStatus{
		{Name: "GLOBAL", Type: "NOTIONAL", Cap: "10000", Usage: "3000"},
	}
	freed := []model.FreedCap{
		{Name: "GLOBAL", Type: "NOTIONAL", OldUsage: "5000", NewUsage: "3000", Cap: "10000"},
	}

	// GLOBAL has no address or underlying — it won't resolve to any asset,
	// so the message should be empty (GLOBAL is direction/asset agnostic).
	result := FormatFreedCaps(freed, capData, hypeAssets)
	if result != "" {
		t.Error("GLOBAL cap should not resolve to specific assets")
	}
}

// Layer 2: Underlying cap freed — should show ALL assets sharing the underlying
// with both Call and Put directions.
func TestFormatFreedCaps_UnderlyingCap_AllAssets(t *testing.T) {
	capData := []model.SLCapsStatus{
		{Name: "HYPE", Type: "NOTIONAL", Cap: "5000", Usage: "1500"},
	}
	freed := []model.FreedCap{
		{Name: "HYPE", Type: "NOTIONAL", OldUsage: "3000", NewUsage: "1500", Cap: "5000"},
	}
	result := FormatFreedCaps(freed, capData, hypeAssets)

	if !strings.Contains(result, "Cap Freed") {
		t.Fatal("expected header")
	}
	// All three HYPE assets should appear
	for _, sym := range []string{"WHYPE", "KHYPE", "LHYPE"} {
		if !strings.Contains(result, sym) {
			t.Errorf("expected %s in underlying freed cap notification", sym)
		}
	}
	// No direction suffix → both Call and Put columns
	if !strings.Contains(result, "Call") {
		t.Error("expected Call column for underlying cap")
	}
	if !strings.Contains(result, "Put") {
		t.Error("expected Put column for underlying cap")
	}
}

// Layer 3: Asset-level cap freed (address, no direction suffix) — should show
// only that asset but with both Call and Put.
func TestFormatFreedCaps_AssetCap_BothDirections(t *testing.T) {
	capData := []model.SLCapsStatus{
		{Name: whypeAsset.Address, Type: "CONTRACTS", Cap: "1000", Usage: "200"},
	}
	freed := []model.FreedCap{
		{Name: whypeAsset.Address, Type: "CONTRACTS", OldUsage: "500", NewUsage: "200", Cap: "1000"},
	}
	result := FormatFreedCaps(freed, capData, hypeAssets)

	if !strings.Contains(result, "WHYPE") {
		t.Error("expected WHYPE in asset-level freed cap")
	}
	// Asset-level cap (no direction suffix) → both directions
	if !strings.Contains(result, "Call") {
		t.Error("expected Call column for asset-level cap")
	}
	if !strings.Contains(result, "Put") {
		t.Error("expected Put column for asset-level cap")
	}
	// Other HYPE assets should NOT appear
	if strings.Contains(result, "KHYPE") || strings.Contains(result, "LHYPE") {
		t.Error("expected only WHYPE, not other HYPE assets")
	}
}

// Layer 4: Direction-specific cap freed — should show only that asset + direction.
func TestFormatFreedCaps_DirectionCap_CallOnly(t *testing.T) {
	capData := []model.SLCapsStatus{
		{Name: whypeAsset.Address + "-false", Type: "NOTIONAL", Cap: "500", Usage: "100"},
	}
	freed := []model.FreedCap{
		{Name: whypeAsset.Address + "-false", Type: "NOTIONAL", OldUsage: "300", NewUsage: "100", Cap: "500"},
	}
	result := FormatFreedCaps(freed, capData, hypeAssets)

	if !strings.Contains(result, "WHYPE") {
		t.Error("expected WHYPE")
	}
	if !strings.Contains(result, "Call") {
		t.Error("expected Call column")
	}
	// Only Call direction was freed — Put column should not appear
	if strings.Contains(result, "Put") {
		t.Error("expected no Put column for Call-only freed cap")
	}
}

func TestFormatFreedCaps_DirectionCap_PutOnly(t *testing.T) {
	capData := []model.SLCapsStatus{
		{Name: whypeAsset.Address + "-true", Type: "CONTRACTS", Cap: "500", Usage: "80"},
	}
	freed := []model.FreedCap{
		{Name: whypeAsset.Address + "-true", Type: "CONTRACTS", OldUsage: "200", NewUsage: "80", Cap: "500"},
	}
	result := FormatFreedCaps(freed, capData, hypeAssets)

	if !strings.Contains(result, "WHYPE") {
		t.Error("expected WHYPE")
	}
	if !strings.Contains(result, "Put") {
		t.Error("expected Put column")
	}
	if strings.Contains(result, "Call") {
		t.Error("expected no Call column for Put-only freed cap")
	}
}

// Mixed scenario: underlying cap + direction-specific cap fire together.
// Underlying should expand to all assets (both dirs), direction-specific should
// be deduplicated if it overlaps.
func TestFormatFreedCaps_UnderlyingAndDirectionMixed(t *testing.T) {
	capData := []model.SLCapsStatus{
		{Name: "HYPE", Type: "NOTIONAL", Cap: "5000", Usage: "1500"},
		{Name: whypeAsset.Address + "-false", Type: "CONTRACTS", Cap: "500", Usage: "100"},
	}
	freed := []model.FreedCap{
		{Name: "HYPE", Type: "NOTIONAL", OldUsage: "3000", NewUsage: "1500", Cap: "5000"},
		{Name: whypeAsset.Address + "-false", Type: "CONTRACTS", OldUsage: "300", NewUsage: "100", Cap: "500"},
	}
	result := FormatFreedCaps(freed, capData, hypeAssets)

	// All assets should appear (from underlying expansion)
	for _, sym := range []string{"WHYPE", "KHYPE", "LHYPE"} {
		if !strings.Contains(result, sym) {
			t.Errorf("expected %s in mixed freed cap notification", sym)
		}
	}
	// WHYPE should appear only once (deduplicated)
	if count := strings.Count(result, "WHYPE"); count != 1 {
		t.Errorf("expected WHYPE once (deduplicated), got %d", count)
	}
}

// Inactive assets should be excluded from freed cap notifications.
func TestFormatFreedCaps_InactiveAssetExcluded(t *testing.T) {
	inactiveAsset := &model.AssetsResponse{
		Symbol:     "XHYPE",
		Underlying: "HYPE",
		Address:    "0xxhype0000000000000000000000000000000004",
		Active:     false,
		Price:      "1000000000000000000",
	}
	assets := map[int][]*model.AssetsResponse{
		999: {whypeAsset, inactiveAsset},
	}
	capData := []model.SLCapsStatus{
		{Name: "HYPE", Type: "NOTIONAL", Cap: "5000", Usage: "1500"},
	}
	freed := []model.FreedCap{
		{Name: "HYPE", Type: "NOTIONAL", OldUsage: "3000", NewUsage: "1500", Cap: "5000"},
	}
	result := FormatFreedCaps(freed, capData, assets)

	if !strings.Contains(result, "WHYPE") {
		t.Error("expected active WHYPE")
	}
	if strings.Contains(result, "XHYPE") {
		t.Error("inactive XHYPE should be excluded")
	}
}
