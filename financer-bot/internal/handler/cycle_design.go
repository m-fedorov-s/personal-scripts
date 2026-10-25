package handler

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"financer/internal/report"
)

// CycleDesignHandler handles the button under a report's chart: it moves the chat to its next
// chart design and sends a new report, which replaces the chat's latest one.
type CycleDesignHandler struct {
	BaseHandler
}

func (designHandler *CycleDesignHandler) Handle(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	query := update.CallbackQuery
	if _, err := tgBot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: query.ID}); err != nil {
		slog.Error("Error answering callback query", "queryID", query.ID, "error", err)
	}
	if query.Message == nil {
		slog.Error("Callback query has no message", "queryID", query.ID)
		return
	}
	chatID := query.Message.Chat.ID
	chatSettings, err := designHandler.Storage.GetChat(chatID)
	if err != nil {
		slog.Error("Error reading chat profile", "chatID", chatID, "error", err)
		return
	}
	chatSettings.ChartDesign = chatSettings.ChartDesign.Next(chatSettings.DayLimit)
	err = designHandler.Storage.SetChat(chatID, &chatSettings)
	if err != nil {
		slog.Error("Error setting chart design for chat", "chatID", chatID, "error", err)
		return
	}
	report.Send(ctx, tgBot, designHandler.Storage, report.Daily, chatID, query.Message.MessageThreadID, nil)
}
