package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

type SetBalanceHandler struct {
	BaseHandler
}

func (h *SetBalanceHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.IsGroupChat(update) {
		return
	}
	parts := strings.Fields(update.Message.Text)
	if len(parts) != 2 {
		return
	}
	newBalance, err := strconv.ParseFloat(parts[1], 32)
	if err != nil {
		return
	}
	chatID := update.Message.Chat.ID
	chatSettings, err := h.Storage.GetChat(chatID)
	if err != nil {
		slog.Error("Error reading chat profile", "chatID", chatID, "error", err)
		return
	}
	chatSettings.CurrentBalance = int64(newBalance)
	chatSettings.LastRecordTime = time.Now()
	err = h.Storage.SetChat(chatID, &chatSettings)
	if err != nil {
		slog.Error("Error setting balance for chat", "chatID", chatID, "error", err)
		return
	}
	h.Reply(ctx, b, update, fmt.Sprintf("Set current balance: %v", newBalance))
}
