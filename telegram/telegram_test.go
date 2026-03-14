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
	result := FormatCapRatios(nil, 0)
	if result != "" {
		t.Error("expected empty string for nil ratios")
	}
}

func TestFormatCapRatios_Basic(t *testing.T) {
	ratios := []model.AssetCapRatio{
		{Asset: testAsset, IsPut: false, Ratio: 50.0},
		{Asset: testAsset, IsPut: true, Ratio: 40.0},
	}
	result := FormatCapRatios(ratios, 25.0)

	if !strings.Contains(result, "Rysk v12 Caps") {
		t.Error("expected header")
	}
	if !strings.Contains(result, "25.00%") {
		t.Error("expected global ratio")
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
	result := FormatFreedCaps(nil)
	if result != "" {
		t.Error("expected empty string for nil freed caps")
	}

	result = FormatFreedCaps([]model.FreedCap{})
	if result != "" {
		t.Error("expected empty string for empty freed caps")
	}
}

func TestFormatFreedCaps_Populated(t *testing.T) {
	freed := []model.FreedCap{
		{Name: "ETH", Type: "CONTRACTS", OldUsage: "500", NewUsage: "300", Cap: "1000"},
		{Name: "BTC", Type: "NOTIONAL", OldUsage: "100", NewUsage: "50", Cap: "200"},
	}
	result := FormatFreedCaps(freed)

	if !strings.Contains(result, "Cap Freed") {
		t.Error("expected header")
	}
	if !strings.Contains(result, "ETH") {
		t.Error("expected ETH")
	}
	if !strings.Contains(result, "BTC") {
		t.Error("expected BTC")
	}
	if !strings.Contains(result, "CONTRACTS") {
		t.Error("expected CONTRACTS type")
	}
	if !strings.Contains(result, "NOTIONAL") {
		t.Error("expected NOTIONAL type")
	}
}

// --- FormatGlobalCap ---

func TestFormatGlobalCap_ZeroPercent(t *testing.T) {
	result := FormatGlobalCap(0)
	if !strings.Contains(result, "0.00%") {
		t.Errorf("expected 0.00%%, got %s", result)
	}
	if !strings.Contains(result, "Global Cap") {
		t.Error("expected header")
	}
}

func TestFormatGlobalCap_HundredPercent(t *testing.T) {
	result := FormatGlobalCap(100)
	if !strings.Contains(result, "100.00%") {
		t.Errorf("expected 100.00%%, got %s", result)
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
