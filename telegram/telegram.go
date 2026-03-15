package telegram

import (
	"fmt"
	"log"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"v12-handy-cap-bot/caps"
	"v12-handy-cap-bot/chatstore"
	"v12-handy-cap-bot/model"
)

// CapQueryFunc is called by the bot to resolve a /cap command.
// name is the asset/underlying name, isPut is nil if no direction specified.
type CapQueryFunc func(name string, isPut *bool) string

type Bot struct {
	bot      *tgbotapi.BotAPI
	store    *chatstore.ChatStore
	chats    map[int64]bool
	mu       sync.RWMutex
	onCapCmd CapQueryFunc
}

func NewBot(token string, store *chatstore.ChatStore) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("telegram bot init: %w", err)
	}
	log.Printf("Authorized as @%s", bot.Self.UserName)

	return &Bot{
		bot:   bot,
		store: store,
		chats: store.Load(),
	}, nil
}

// SetCapHandler registers the callback for /cap commands.
func (t *Bot) SetCapHandler(fn CapQueryFunc) {
	t.onCapCmd = fn
}

// ListenForUpdates processes incoming messages to track group membership and commands.
func (t *Bot) ListenForUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := t.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.MyChatMember != nil {
			chatID := update.MyChatMember.Chat.ID
			status := update.MyChatMember.NewChatMember.Status

			switch status {
			case "member", "administrator":
				t.addChat(chatID)
				log.Printf("Added to chat %d (%s)", chatID, update.MyChatMember.Chat.Title)
			case "left", "kicked":
				t.removeChat(chatID)
				log.Printf("Removed from chat %d (%s)", chatID, update.MyChatMember.Chat.Title)
			}
			continue
		}

		if update.Message != nil {
			t.addChat(update.Message.Chat.ID)

			if update.Message.IsCommand() {
				t.handleCommand(update.Message)
			}
		}
	}
}

func (t *Bot) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "cap":
		t.handleCapCommand(msg)
	}
}

func (t *Bot) handleCapCommand(msg *tgbotapi.Message) {
	if t.onCapCmd == nil {
		return
	}

	args := strings.Fields(msg.CommandArguments())
	if len(args) == 0 {
		t.reply(msg.Chat.ID, msg.MessageID, "Usage: `/cap <name> [put/call]`")
		return
	}

	name := args[0]
	var isPut *bool

	if len(args) >= 2 {
		dir := strings.ToLower(args[1])
		switch dir {
		case "put":
			v := true
			isPut = &v
		case "call":
			v := false
			isPut = &v
		default:
			t.reply(msg.Chat.ID, msg.MessageID, "Direction must be `put` or `call`")
			return
		}
	}

	response := t.onCapCmd(name, isPut)
	t.reply(msg.Chat.ID, msg.MessageID, response)
}

func (t *Bot) reply(chatID int64, replyTo int, text string) {
	tgMsg := tgbotapi.NewMessage(chatID, text)
	tgMsg.ParseMode = "MarkdownV2"
	tgMsg.ReplyToMessageID = replyTo
	if _, err := t.bot.Send(tgMsg); err != nil {
		log.Printf("telegram reply: %v", err)
	}
}

func (t *Bot) addChat(chatID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.chats[chatID] {
		return
	}
	t.chats[chatID] = true
	t.store.Save(t.chats)
}

func (t *Bot) removeChat(chatID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.chats[chatID] {
		return
	}
	delete(t.chats, chatID)
	t.store.Save(t.chats)
}

func (t *Bot) BroadcastCapRatios(ratios []model.AssetCapRatio, globalRatio float64) {
	msg := FormatCapRatios(ratios, globalRatio)
	if msg == "" {
		return
	}

	t.mu.RLock()
	chatIDs := make([]int64, 0, len(t.chats))
	for id := range t.chats {
		chatIDs = append(chatIDs, id)
	}
	t.mu.RUnlock()

	for _, chatID := range chatIDs {
		tgMsg := tgbotapi.NewMessage(chatID, msg)
		tgMsg.ParseMode = "MarkdownV2"
		if _, err := t.bot.Send(tgMsg); err != nil {
			log.Printf("telegram send to %d: %v", chatID, err)
			if isChatGone(err) {
				t.removeChat(chatID)
				log.Printf("Removed unreachable chat %d", chatID)
			}
		}
	}
}

func isChatGone(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Forbidden") ||
		strings.Contains(s, "chat not found") ||
		strings.Contains(s, "bot was blocked") ||
		strings.Contains(s, "bot was kicked")
}

func FormatCapRatios(ratios []model.AssetCapRatio, globalRatio float64) string {
	if len(ratios) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("*Rysk v12 Caps*\n\n")
	fmt.Fprintf(&b, "Global: `%.2f%%`\n\n", globalRatio)

	for _, r := range ratios {
		dir := "Call"
		if r.IsPut {
			dir = "Put"
		}
		fmt.Fprintf(&b, "`%-8s` %s  `%.2f%%`\n",
			EscMD(r.Asset.Symbol),
			EscMD(dir),
			r.Ratio,
		)
	}

	return b.String()
}

// FormatSingleCapRatio formats a single asset's cap ratio response.
func FormatSingleCapRatio(name string, ratios []model.AssetCapRatio) string {
	if len(ratios) == 0 {
		return fmt.Sprintf("No cap data for `%s`", EscMD(name))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "*Cap: %s*\n\n", EscMD(name))

	for _, r := range ratios {
		dir := "Call"
		if r.IsPut {
			dir = "Put"
		}
		fmt.Fprintf(&b, "`%-8s` %s  `%.2f%%`\n",
			EscMD(r.Asset.Symbol),
			EscMD(dir),
			r.Ratio,
		)
	}

	return b.String()
}

// FormatGlobalCap formats the global cap ratio.
func FormatGlobalCap(ratio float64) string {
	return fmt.Sprintf("*Global Cap*\n\n`%.2f%%`", ratio)
}

// FormatFreedCaps formats a notification about freed caps using the same
// layout as /cap responses: asset symbol, direction, and current usage ratio.
func FormatFreedCaps(freed []model.FreedCap, capData []model.SLCapsStatus, assets map[int][]*model.AssetsResponse) string {
	if len(freed) == 0 {
		return ""
	}

	// Deduplicate by asset address — multiple cap types may fire for the same asset.
	type assetDir struct {
		asset *model.AssetsResponse
		isPut bool
	}
	seen := make(map[string]bool)
	var entries []assetDir

	for _, f := range freed {
		addr, isPut := parseFreedName(f.Name)
		key := fmt.Sprintf("%s-%t", addr, isPut)
		if seen[key] {
			continue
		}
		seen[key] = true

		asset := caps.FindAssetByAddress(assets, addr)
		if asset == nil {
			asset = caps.FindAssetByUnderlying(assets, addr)
		}
		if asset == nil {
			continue
		}
		entries = append(entries, assetDir{asset: asset, isPut: isPut})
	}

	if len(entries) == 0 {
		return ""
	}

	var ratios []model.AssetCapRatio
	for _, e := range entries {
		ratio := caps.GetCapUsageRatio(capData, e.asset, e.isPut)
		ratios = append(ratios, model.AssetCapRatio{
			Asset: e.asset,
			IsPut: e.isPut,
			Ratio: ratio,
		})
	}

	var b strings.Builder
	b.WriteString("*Cap Freed\\!*\n\n")

	for _, r := range ratios {
		dir := "Call"
		if r.IsPut {
			dir = "Put"
		}
		fmt.Fprintf(&b, "`%-8s` %s  `%.2f%%`\n",
			EscMD(r.Asset.Symbol),
			EscMD(dir),
			r.Ratio,
		)
	}

	return b.String()
}

// parseFreedName extracts the address and direction from a FreedCap.Name
// which may be "0xaddr", "0xaddr-false", "0xaddr-true", or an underlying name.
func parseFreedName(name string) (addr string, isPut bool) {
	if rest, ok := strings.CutSuffix(name, "-true"); ok {
		return rest, true
	}
	if rest, ok := strings.CutSuffix(name, "-false"); ok {
		return rest, false
	}
	return name, false
}

// BroadcastFreedCaps sends a notification about freed caps to all chats.
func (t *Bot) BroadcastFreedCaps(freed []model.FreedCap, capData []model.SLCapsStatus, assets map[int][]*model.AssetsResponse) {
	msg := FormatFreedCaps(freed, capData, assets)
	if msg == "" {
		return
	}

	t.mu.RLock()
	chatIDs := make([]int64, 0, len(t.chats))
	for id := range t.chats {
		chatIDs = append(chatIDs, id)
	}
	t.mu.RUnlock()

	for _, chatID := range chatIDs {
		tgMsg := tgbotapi.NewMessage(chatID, msg)
		tgMsg.ParseMode = "MarkdownV2"
		if _, err := t.bot.Send(tgMsg); err != nil {
			log.Printf("telegram send to %d: %v", chatID, err)
			if isChatGone(err) {
				t.removeChat(chatID)
				log.Printf("Removed unreachable chat %d", chatID)
			}
		}
	}
}

func EscMD(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(s)
}
