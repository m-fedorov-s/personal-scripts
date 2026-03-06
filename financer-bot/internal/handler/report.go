package handler

import (
	"bytes"
	"context"
	"log/slog"
	"time"

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
	chatSettings, err := h.Storage.GetChat(chatID)
	if err != nil {
		slog.Error("Error reading chat profile", "chatID", chatID, "error", err)
		return
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	records, err := h.Storage.GetRecords(chatID, monthStart)
	if err != nil {
		slog.Error("Error reading records for report", "chatID", chatID, "error", err)
		return
	}

	stats := report.Generate(chatSettings, records)
	msg := report.Format(stats)

	if len(stats.Chart) > 0 {
		_, err = b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:          chatID,
			MessageThreadID: update.Message.MessageThreadID,
			Photo: &models.InputFileUpload{
				Filename: "spending_chart.png",
				Data:     bytes.NewReader(stats.Chart),
			},
			Caption:          msg,
			ParseMode:        models.ParseModeHTML,
			ReplyToMessageID: update.Message.ID,
		})
		if err != nil {
			slog.Error("Error sending chart photo", "chatID", chatID, "error", err)
		}
		return
	}

	h.Reply(ctx, b, update, msg)
}
