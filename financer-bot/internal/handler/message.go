package handler

import (
	"context"
	"financer/internal/parser"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

type MessageHandler struct {
	BaseHandler
}

func (h *MessageHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.IsGroupChat(update) {
		return
	}
	chatID := update.Message.Chat.ID
	chatSettings, err := h.Storage.GetChat(chatID)
	if err != nil {
		slog.Error("Failed to get chat settings", "chatID", chatID, "error", err)
		return
	}
	if chatSettings.TopicID != 0 && update.Message.MessageThreadID != chatSettings.TopicID {
		return
	}
	records := parser.ParseMessage(update.Message.Text)
	if len(records) == 0 {
		return
	}
	for idx := range records {
		records[idx].ID = chatSettings.MaxRecordID + 1
		chatSettings.MaxRecordID += 1
		chatSettings.ConsumeRecord(records[idx])
		err := h.Storage.SaveRecord(chatID, records[idx])
		if err != nil {
			slog.Error("Failed to save record", "chatID", chatID, "error", err)
		}
	}
	err = h.Storage.SetChat(chatID, &chatSettings)
	if err != nil {
		slog.Error("Failed to save chat settings", "chatID", chatID, "error", err)
	}
	if chatSettings.ImmediateReports {
		h.Reply(ctx, b, update, chatSettings.Report())
	}
}
