package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"financer/internal/storage"
)

type SetReportTimeHandler struct {
	BaseHandler
}

func (h *SetReportTimeHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.IsGroupChat(update) {
		return
	}
	parts := strings.Fields(update.Message.Text)
	if len(parts) != 2 {
		h.Reply(ctx, b, update, "Usage: /setReportTime HH:MM (e.g., /setReportTime 09:00)")
		return
	}

	timeStr := parts[1]
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		h.Reply(ctx, b, update, "Invalid time format. Please use HH:MM (e.g., 09:00)")
		return
	}

	chatID := update.Message.Chat.ID
	chatSettings, err := h.Storage.GetChat(chatID)
	if err != nil {
		slog.Error("Error reading chat profile", "chatID", chatID, "error", err)
		return
	}

	dt := &storage.DayTime{
		Hour:   t.Hour(),
		Minute: t.Minute(),
	}
	chatSettings.ReportTime = dt
	err = h.Storage.SetChat(chatID, &chatSettings)
	if err != nil {
		slog.Error("Error setting report time for chat", "chatID", chatID, "error", err)
		return
	}

	// Signal worker to reschedule
	if h.Reschedule != nil {
		h.Reschedule <- dt
	}

	// Get current server time and its timezone for clarity
	now := time.Now()
	zone, offset := now.Zone()
	offsetHours := offset / 3600

	h.Reply(ctx, b, update, fmt.Sprintf("Report time set to %s (%s, UTC%+d)",
		t.Format("15:04"),
		zone,
		offsetHours,
	))
}
