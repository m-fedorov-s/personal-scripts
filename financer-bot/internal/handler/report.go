package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"financer/internal/report"
)

type ReportHandler struct {
	BaseHandler
}

func (h *ReportHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.IsGroupChat(update) {
		return
	}
	chatID := update.Message.Chat.ID
	report.Send(ctx, b, h.Storage, report.Daily, chatID, update.Message.MessageThreadID, &update.Message.ID)
}
