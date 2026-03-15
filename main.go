package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"v12-handy-cap-bot/caps"
	"v12-handy-cap-bot/chatstore"
	"v12-handy-cap-bot/model"
	"v12-handy-cap-bot/rpc"
	"v12-handy-cap-bot/telegram"
)

const defaultPollSec = 2

func main() {
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN env var is required")
	}

	wsURL := os.Getenv("WS_URL")
	if wsURL == "" {
		log.Fatal("WS_URL env var is required")
	}

	pollSec := defaultPollSec
	if v := os.Getenv("POLL_INTERVAL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pollSec = n
		}
	}

	// --- Telegram bot ---
	chatPath := os.Getenv("CHAT_STORE_PATH")
	if chatPath == "" {
		chatPath = "chats.txt"
	}
	store := chatstore.NewChatStore(chatPath)
	tg, err := telegram.NewBot(token, store)
	if err != nil {
		log.Fatalf("telegram: %v", err)
	}

	// --- Shared state for caps/assets ---
	var (
		mu           sync.RWMutex
		latestCaps   []model.SLCapsStatus
		latestAssets = make(map[int][]*model.AssetsResponse)
	)

	// --- /cap command handler ---
	tg.SetCapHandler(func(name string, isPut *bool) string {
		mu.RLock()
		defer mu.RUnlock()

		if strings.ToLower(name) == "global" {
			ratio := caps.GetGlobalCapUsageRatio(latestCaps)
			return telegram.FormatGlobalCap(ratio)
		}

		assets := caps.FindAssetsByName(latestAssets, name)
		if len(assets) == 0 {
			return "Asset not found: `" + telegram.EscMD(name) + "`"
		}

		var ratios []model.AssetCapRatio
		for _, asset := range assets {
			if isPut != nil {
				ratio := caps.GetCapUsageRatio(latestCaps, asset, *isPut)
				ratios = append(ratios, model.AssetCapRatio{
					Asset: asset,
					IsPut: *isPut,
					Ratio: ratio,
				})
			} else {
				for _, put := range []bool{false, true} {
					ratio := caps.GetCapUsageRatio(latestCaps, asset, put)
					ratios = append(ratios, model.AssetCapRatio{
						Asset: asset,
						IsPut: put,
						Ratio: ratio,
					})
				}
			}
		}

		return telegram.FormatSingleCapRatio(name, ratios)
	})

	go tg.ListenForUpdates()

	// --- WebSocket ---
	ws, err := rpc.NewWSClient(wsURL)
	if err != nil {
		log.Fatalf("websocket: %v", err)
	}

	// --- Poll loop ---
	ticker := time.NewTicker(time.Duration(pollSec) * time.Second)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	tracker := caps.NewFreedCapsTracker(30 * time.Second)

	poll := func() {
		fetchedAssets, err := ws.FetchAssets()
		if err != nil {
			log.Printf("fetch assets: %v", err)
		}

		capData, err := ws.FetchCaps()
		if err != nil {
			log.Printf("fetch caps: %v", err)
			return
		}

		mu.Lock()
		latestCaps = capData
		if fetchedAssets != nil {
			latestAssets = fetchedAssets
		}
		mu.Unlock()

		if freed := tracker.Update(capData); len(freed) > 0 {
			mu.RLock()
			assets := latestAssets
			mu.RUnlock()
			tg.BroadcastFreedCaps(freed, capData, assets)
		}
	}

	// Initial poll
	poll()

	for {
		select {
		case <-ticker.C:
			poll()
		case <-quit:
			log.Println("Shutting down...")
			ws.Close()
			return
		}
	}
}
