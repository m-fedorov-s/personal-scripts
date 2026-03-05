package handler

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
	"strconv"
	"strings"
)

type SetLimitHandler struct {
	BaseHandler
}

func (h *SetLimitHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.IsGroupChat(update) {
		return
	}
	parts := strings.Fields(update.Message.Text)
	if len(parts) != 2 {
		return
	}
	limit, err := strconv.ParseFloat(parts[1], 32)
	if err != nil {
		return
	}
	chatID := update.Message.Chat.ID
	chatSettings, err := h.Storage.GetChat(chatID)
	if err != nil {
		slog.Error("Error reading chat profile", "chatID", chatID, "error", err)
		return
	}
	chatSettings.DayLimit = int64(limit)
	err = h.Storage.SetChat(chatID, &chatSettings)
	if err != nil {
		slog.Error("Error setting limits for chat", "chatID", chatID, "error", err)
		return
	}
	h.Reply(ctx, b, update, fmt.Sprintf("New daily limit: %v", limit))
}
