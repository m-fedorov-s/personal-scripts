package handler

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

type ToggleReportHandler struct {
	BaseHandler
}

func (h *ToggleReportHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.IsGroupChat(update) {
		return
	}
	chatID := update.Message.Chat.ID
	chatSettings, err := h.Storage.GetChat(chatID)
	if err != nil {
		slog.Error("Error reading chat profile", "chatID", chatID, "error", err)
		return
	}
	chatSettings.ImmediateReports = !chatSettings.ImmediateReports
	err = h.Storage.SetChat(chatID, &chatSettings)
	if err != nil {
		slog.Error("Error setting balance for chat", "chatID", chatID, "error", err)
		return
	}
	h.Reply(ctx, b, update, "Done")
}
