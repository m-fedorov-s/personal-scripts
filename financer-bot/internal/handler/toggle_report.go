package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"financer/internal/storage"
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
	if chatSettings.ImmediateReports && chatSettings.ReportTime == nil {
		chatSettings.ReportTime = &storage.DayTime{
			Hour:   0,
			Minute: 0,
		}
	}
	err = h.Storage.SetChat(chatID, &chatSettings)
	if err != nil {
		slog.Error("Error togling reports for chat", "chatID", chatID, "error", err)
		return
	}

	// Signal worker to reschedule (full rescan if enabled)
	if chatSettings.ImmediateReports && h.Reschedule != nil {
		h.Reschedule <- chatSettings.ReportTime
	}

	status := "disabled"
	if chatSettings.ImmediateReports {
		status = "enabled"
	}
	h.Reply(ctx, b, update, fmt.Sprintf("Daily reports %s", status))
}
