package report

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"financer/internal/storage"
)

// Kind selects what Send sends.
type Kind int

const (
	// Daily is the stats text, captioning the spending chart when there is one.
	Daily Kind = iota
	// Balance is the one-line current balance.
	Balance
)

// Send sends a report of the given kind to chatID, in threadID, replying to
// replyTo unless it is nil. It then deletes the chat's previous report, so a
// chat only shows the latest one.
//
// It returns an error only when the report was not sent. Failing to save the
// new message ID or to delete the previous report is logged and ignored.
func Send(ctx context.Context, tgBot *bot.Bot, store storage.Storage, kind Kind, chatID int64, threadID int, replyTo *int) error {
	text, chart, err := build(store, kind, chatID)
	if err != nil {
		slog.Error("Failed to build report", "chatID", chatID, "error", err)
		return err
	}

	var replyToID int
	if replyTo != nil {
		replyToID = *replyTo
	}

	var msg *models.Message
	if len(chart) > 0 {
		msg, err = tgBot.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:          chatID,
			MessageThreadID: threadID,
			Photo: &models.InputFileUpload{
				Filename: "spending_chart.png",
				Data:     bytes.NewReader(chart),
			},
			Caption:          text,
			ParseMode:        models.ParseModeHTML,
			ReplyToMessageID: replyToID,
		})
	} else {
		msg, err = tgBot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:           chatID,
			MessageThreadID:  threadID,
			Text:             text,
			ParseMode:        models.ParseModeHTML,
			ReplyToMessageID: replyToID,
		})
	}
	if err != nil {
		slog.Error("Failed to send report", "chatID", chatID, "error", err)
		return err
	}

	// Load the profile fresh: it may have changed while the report was sent.
	cp, err := store.GetChat(chatID)
	if err != nil {
		slog.Error("Failed to read chat profile to save report ID", "chatID", chatID, "error", err)
		return nil
	}
	prevID := cp.LastReportID
	cp.LastReportID = &msg.ID
	if err := store.SetChat(chatID, &cp); err != nil {
		slog.Error("Failed to save report ID", "chatID", chatID, "error", err)
		return nil
	}

	if prevID != nil {
		// Telegram refuses to delete messages older than 48h or already deleted.
		_, err := tgBot.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: chatID, MessageID: *prevID})
		if err != nil {
			slog.Warn("Failed to delete previous report", "chatID", chatID, "messageID", *prevID, "error", err)
		}
	}
	return nil
}

// build returns the text of a report and, for a daily report, its chart.
func build(store storage.Storage, kind Kind, chatID int64) (string, []byte, error) {
	cp, err := store.GetChat(chatID)
	if err != nil {
		return "", nil, fmt.Errorf("reading chat profile: %w", err)
	}
	if kind == Balance {
		return cp.Report(), nil, nil
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	records, err := store.GetRecords(chatID, monthStart)
	if err != nil {
		return "", nil, fmt.Errorf("reading records: %w", err)
	}
	stats := Generate(cp, records, now)
	return Format(stats), stats.Chart, nil
}
