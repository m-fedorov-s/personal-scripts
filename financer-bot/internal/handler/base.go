package handler

import (
	"context"
	"financer/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type BaseHandler struct {
	Storage storage.Storage
}

func (h *BaseHandler) IsGroupChat(update *models.Update) bool {
	return update.Message != nil && update.Message.Chat.Type != "private"
}

func (h *BaseHandler) Reply(ctx context.Context, b *bot.Bot, update *models.Update, text string) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:           update.Message.Chat.ID,
		MessageThreadID:  update.Message.MessageThreadID,
		ReplyToMessageID: update.Message.ID,
		Text:             text,
	})
}
