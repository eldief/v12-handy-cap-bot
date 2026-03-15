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
