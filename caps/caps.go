package caps

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"v12-handy-cap-bot/model"
)

var (
	scale = big.NewInt(1_000_000)
	zero  = big.NewInt(0)
)

// pendingFree tracks a cap that is being gathered.
type pendingFree struct {
	startUsage *big.Int // usage when decrease was first detected
	lastUsage  *big.Int // latest (lowest) usage seen
	cap        string
}

// FreedCapsTracker gathers cap usage decreases into a single batch.
// Once the first decrease is detected a gathering window starts. All decreases
// observed during that window are collected. When the window expires the entire
// batch is flushed as one notification.
type FreedCapsTracker struct {
	window    time.Duration
	lastUsage map[string]*big.Int
	pending   map[string]*pendingFree
	// windowStart is zero when no gathering is in progress.
	windowStart time.Time
}

func NewFreedCapsTracker(window time.Duration) *FreedCapsTracker {
	return &FreedCapsTracker{
		window:    window,
		lastUsage: make(map[string]*big.Int),
		pending:   make(map[string]*pendingFree),
	}
}

// Update processes new caps data. It returns all gathered freed caps once the
// gathering window (started from the first decrease) has elapsed.
func (t *FreedCapsTracker) Update(caps []model.SLCapsStatus) []model.FreedCap {
	now := time.Now()

	for _, c := range caps {
		key := c.Name + "|" + c.Type

		currUsage, ok := new(big.Int).SetString(c.Usage, 10)
		if !ok {
			continue
		}

		prev, hasPrev := t.lastUsage[key]

		if hasPrev && currUsage.Cmp(prev) < 0 {
			// Usage decreased — start window if not already running
			if t.windowStart.IsZero() {
				t.windowStart = now
			}

			p, exists := t.pending[key]
			if exists {
				// Already tracking — update to latest lower value
				p.lastUsage = new(big.Int).Set(currUsage)
			} else {
				t.pending[key] = &pendingFree{
					startUsage: new(big.Int).Set(prev),
					lastUsage:  new(big.Int).Set(currUsage),
					cap:        c.Cap,
				}
			}
		} else if hasPrev && currUsage.Cmp(prev) > 0 {
			// Usage increased — cancel this entry
			delete(t.pending, key)
		}

		t.lastUsage[key] = new(big.Int).Set(currUsage)
	}

	// Flush everything once the gathering window expires
	if t.windowStart.IsZero() || now.Sub(t.windowStart) < t.window {
		return nil
	}

	var freed []model.FreedCap
	for key, p := range t.pending {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) < 2 {
			continue
		}
		freed = append(freed, model.FreedCap{
			Name:     parts[0],
			Type:     parts[1],
			OldUsage: p.startUsage.String(),
			NewUsage: p.lastUsage.String(),
			Cap:      p.cap,
		})
	}

	// Reset for next gathering cycle
	t.pending = make(map[string]*pendingFree)
	t.windowStart = time.Time{}

	return freed
}

func findCap(caps []model.SLCapsStatus, name, capType string) *model.SLCapsStatus {
	for i := range caps {
		if capType != "" && caps[i].Type != capType {
			continue
		}
		if caps[i].Name == name {
			return &caps[i]
		}
	}
	return nil
}

func relevantCaps(caps []model.SLCapsStatus, asset *model.AssetsResponse, isPut bool) []model.SLCapsStatus {
	putStr := fmt.Sprintf("%s-%t", asset.Address, isPut)

	candidates := []*model.SLCapsStatus{
		findCap(caps, "GLOBAL", ""),
		findCap(caps, asset.Underlying, "CONTRACTS"),
		findCap(caps, asset.Underlying, "NOTIONAL"),
		findCap(caps, asset.Address, "CONTRACTS"),
		findCap(caps, asset.Address, "NOTIONAL"),
		findCap(caps, putStr, "CONTRACTS"),
		findCap(caps, putStr, "NOTIONAL"),
	}

	var entries []model.SLCapsStatus
	for _, c := range candidates {
		if c != nil && c.Cap != "0" {
			entries = append(entries, *c)
		}
	}
	return entries
}

// GetCapUsageRatio returns the max usage/cap ratio (as percentage) across all
// relevant cap entries for the given asset and direction.
func GetCapUsageRatio(caps []model.SLCapsStatus, asset *model.AssetsResponse, isPut bool) float64 {
	if len(caps) == 0 || asset == nil {
		return 0
	}

	entries := relevantCaps(caps, asset, isPut)
	maxRatio := new(big.Int)

	for _, entry := range entries {
		capVal, ok := new(big.Int).SetString(entry.Cap, 10)
		if !ok || capVal.Cmp(zero) == 0 {
			continue
		}
		usageVal, ok := new(big.Int).SetString(entry.Usage, 10)
		if !ok {
			continue
		}

		ratio := new(big.Int).Mul(usageVal, scale)
		ratio.Div(ratio, capVal)

		if ratio.Cmp(maxRatio) > 0 {
			maxRatio = ratio
		}
	}

	f := new(big.Float).SetInt(maxRatio)
	f.Quo(f, big.NewFloat(10_000))
	result, _ := f.Float64()
	return result
}

// GetGlobalCapUsageRatio returns the GLOBAL cap usage ratio as percentage.
func GetGlobalCapUsageRatio(caps []model.SLCapsStatus) float64 {
	if len(caps) == 0 {
		return 0
	}

	global := findCap(caps, "GLOBAL", "")
	if global == nil || global.Cap == "0" {
		return 0
	}

	capVal, ok := new(big.Int).SetString(global.Cap, 10)
	if !ok || capVal.Cmp(zero) == 0 {
		return 0
	}
	usageVal, ok := new(big.Int).SetString(global.Usage, 10)
	if !ok {
		return 0
	}

	ratio := new(big.Int).Mul(usageVal, scale)
	ratio.Div(ratio, capVal)

	f := new(big.Float).SetInt(ratio)
	f.Quo(f, big.NewFloat(10_000))
	result, _ := f.Float64()
	return result
}

// ComputeAllCapRatios computes cap usage ratios for every active asset
// (both isPut=true and isPut=false), plus the global ratio.
func ComputeAllCapRatios(caps []model.SLCapsStatus, assets map[int][]*model.AssetsResponse) ([]model.AssetCapRatio, float64) {
	globalRatio := GetGlobalCapUsageRatio(caps)

	var ratios []model.AssetCapRatio
	seen := make(map[string]bool)

	for _, assetList := range assets {
		for _, asset := range assetList {
			if !asset.Active {
				continue
			}

			for _, isPut := range []bool{false, true} {
				key := fmt.Sprintf("%s-%t", asset.Address, isPut)
				if seen[key] {
					continue
				}
				seen[key] = true

				ratio := GetCapUsageRatio(caps, asset, isPut)
				ratios = append(ratios, model.AssetCapRatio{
					Asset: asset,
					IsPut: isPut,
					Ratio: ratio,
				})
			}
		}
	}

	return ratios, globalRatio
}

// FindAssetsByName returns all active assets matching by symbol, underlying, or address (case-insensitive).
func FindAssetsByName(assets map[int][]*model.AssetsResponse, name string) []*model.AssetsResponse {
	lower := strings.ToLower(name)
	var result []*model.AssetsResponse
	for _, assetList := range assets {
		for _, a := range assetList {
			if !a.Active {
				continue
			}
			if strings.ToLower(a.Symbol) == lower ||
				strings.ToLower(a.Underlying) == lower ||
				strings.ToLower(a.Address) == lower {
				result = append(result, a)
			}
		}
	}
	return result
}

// FindAssetByAddress returns the first asset matching the given address
// across all chains.
func FindAssetByAddress(assets map[int][]*model.AssetsResponse, address string) *model.AssetsResponse {
	addr := strings.ToLower(address)
	for _, assetList := range assets {
		for _, a := range assetList {
			if !a.Active {
				continue
			}
			if strings.ToLower(a.Address) == addr {
				return a
			}
		}
	}
	return nil
}

// FindAssetsByUnderlying returns all active assets matching the given
// underlying across all chains.
func FindAssetsByUnderlying(assets map[int][]*model.AssetsResponse, underlying string) []*model.AssetsResponse {
	var result []*model.AssetsResponse
	for _, assetList := range assets {
		for _, a := range assetList {
			if !a.Active {
				continue
			}
			if a.Underlying == underlying {
				result = append(result, a)
			}
		}
	}
	return result
}
