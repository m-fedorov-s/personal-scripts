package handler

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

type StartHandler struct {
	BaseHandler
}

func (h *StartHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.IsGroupChat(update) {
		return
	}
	chatID := update.Message.Chat.ID
	chatSettings, err := h.Storage.GetChat(chatID)
	if err != nil {
		slog.Error("Error starting chat", "chatID", chatID, "error", err)
		return
	}
	chatSettings.TopicID = update.Message.MessageThreadID
	err = h.Storage.SetChat(chatID, &chatSettings)
	if err != nil {
		slog.Error("Error saving chat settings", "chatID", chatID, "error", err)
		return
	}
	h.Reply(ctx, b, update, "Your chat is successfully registered!")
}
