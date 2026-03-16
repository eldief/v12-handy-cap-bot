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

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
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

func (t *Bot) BroadcastCapRatios(ratios []model.AssetCapRatio) {
	msg := FormatCapRatios(ratios)
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

func FormatCapRatios(ratios []model.AssetCapRatio) string {
	if len(ratios) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("*Rysk v12 Caps*\n\n")

	writeGroupedRatios(&b, ratios)

	return b.String()
}

// FormatSingleCapRatio formats a single asset's cap ratio response.
func FormatSingleCapRatio(name string, ratios []model.AssetCapRatio) string {
	if len(ratios) == 0 {
		return fmt.Sprintf("No cap data for `%s`", EscMD(name))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "*Cap: %s*\n\n", EscMD(name))

	writeGroupedRatios(&b, ratios)

	return b.String()
}


// assetGroup holds Call and Put ratios for a single asset.
type assetGroup struct {
	symbol   string
	callPct  float64
	putPct   float64
	hasCall  bool
	hasPut   bool
}

func groupRatios(ratios []model.AssetCapRatio) []assetGroup {
	order := []string{}
	groups := make(map[string]*assetGroup)

	for _, r := range ratios {
		key := r.Asset.Address
		g, exists := groups[key]
		if !exists {
			g = &assetGroup{symbol: r.Asset.Symbol}
			groups[key] = g
			order = append(order, key)
		}
		if r.IsPut {
			g.putPct = r.Ratio
			g.hasPut = true
		} else {
			g.callPct = r.Ratio
			g.hasCall = true
		}
	}

	result := make([]assetGroup, 0, len(order))
	for _, key := range order {
		result = append(result, *groups[key])
	}
	return result
}

func writeGroupedRatios(b *strings.Builder, ratios []model.AssetCapRatio) {
	groups := groupRatios(ratios)
	if len(groups) == 0 {
		return
	}

	// Header row
	hasAnyCall := false
	hasAnyPut := false
	for _, g := range groups {
		if g.hasCall {
			hasAnyCall = true
		}
		if g.hasPut {
			hasAnyPut = true
		}
	}

	header := "`            "
	if hasAnyCall {
		header += "  Call    "
	}
	if hasAnyPut {
		header += "  Put     "
	}
	header += "`\n"
	b.WriteString(header)

	for _, g := range groups {
		maxRatio := g.callPct
		if g.putPct > maxRatio {
			maxRatio = g.putPct
		}

		emoji := ratioEmoji(maxRatio)
		line := fmt.Sprintf("%s  `%-8s", emoji, EscMD(g.symbol))

		if hasAnyCall {
			if g.hasCall {
				line += fmt.Sprintf("  %6.2f%%", g.callPct)
			} else {
				line += "      -  "
			}
		}
		if hasAnyPut {
			if g.hasPut {
				line += fmt.Sprintf("  %6.2f%%", g.putPct)
			} else {
				line += "      -  "
			}
		}

		line += "`\n"
		b.WriteString(line)
	}
}

func ratioEmoji(pct float64) string {
	switch {
	case pct >= 80:
		return "\xf0\x9f\x94\xb4" // 🔴
	case pct >= 50:
		return "\xf0\x9f\x9f\xa1" // 🟡
	default:
		return "\xf0\x9f\x9f\xa2" // 🟢
	}
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
		addr, isPut, hasDir := parseFreedName(f.Name)

		var matched []*model.AssetsResponse
		if a := caps.FindAssetByAddress(assets, addr); a != nil {
			matched = []*model.AssetsResponse{a}
		} else {
			matched = caps.FindAssetsByUnderlying(assets, addr)
		}
		if len(matched) == 0 {
			continue
		}

		dirs := []bool{isPut}
		if !hasDir {
			dirs = []bool{false, true}
		}
		for _, asset := range matched {
			for _, d := range dirs {
				key := fmt.Sprintf("%s-%t", asset.Address, d)
				if seen[key] {
					continue
				}
				seen[key] = true
				entries = append(entries, assetDir{asset: asset, isPut: d})
			}
		}
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

	writeGroupedRatios(&b, ratios)

	return b.String()
}

// parseFreedName extracts the address and direction from a FreedCap.Name
// which may be "0xaddr", "0xaddr-false", "0xaddr-true", or an underlying name.
// When no direction suffix is present, hasDir is false meaning both directions apply.
func parseFreedName(name string) (addr string, isPut bool, hasDir bool) {
	if rest, ok := strings.CutSuffix(name, "-true"); ok {
		return rest, true, true
	}
	if rest, ok := strings.CutSuffix(name, "-false"); ok {
		return rest, false, true
	}
	return name, false, false
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
