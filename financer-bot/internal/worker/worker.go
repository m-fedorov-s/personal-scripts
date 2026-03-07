package worker

import (
	"bytes"
	"context"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"financer/internal/report"
	"financer/internal/storage"
)

type reportTask struct {
	chatID int64
	cp     storage.ChatProfile
}

type ReportWorker struct {
	bot         *bot.Bot
	storage     storage.Storage
	concurrency int
	tasks       chan reportTask
	Reschedule  chan *storage.DayTime
}

func NewReportWorker(b *bot.Bot, s storage.Storage, concurrency int) *ReportWorker {
	return &ReportWorker{
		bot:         b,
		storage:     s,
		concurrency: concurrency,
		tasks:       make(chan reportTask),
		Reschedule:  make(chan *storage.DayTime, 1),
	}
}

func (w *ReportWorker) Start(ctx context.Context) {
	slog.Info("Starting report worker...", "concurrency", w.concurrency)

	// Start worker pool
	for i := 0; i < w.concurrency; i++ {
		go w.worker(ctx)
	}

	nextWakeup := w.nextRun()
	timer := time.NewTimer(time.Until(nextWakeup))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping report worker...")
			return
		case dt := <-w.Reschedule:
			// If dt is nil, it means we should do a full rescan (e.g. toggle report)
			if dt == nil {
				nextWakeup = w.nextRun()
				timer.Reset(time.Until(nextWakeup))
				continue
			}

			// Calculate next run for this specific DayTime
			now := time.Now()
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), dt.Hour, dt.Minute, 0, 0, now.Location())
			if nextRun.Before(now) || nextRun.Equal(now) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			// If this new time is sooner than our current soonest, reschedule
			if nextRun.Before(nextWakeup) {
				slog.Info("Rescheduling due to new report time", "newSoonest", nextRun)
				nextWakeup = nextRun
				timer.Reset(time.Until(nextWakeup))
			}
		case <-timer.C:
			w.dispatchReports(ctx)
			nextWakeup = w.nextRun()
			timer.Reset(time.Until(nextWakeup))
		}
	}
}

func (w *ReportWorker) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			w.sendReport(ctx, task.chatID, task.cp)
		}
	}
}

func (w *ReportWorker) nextRun() time.Time {
	now := time.Now()
	it := w.storage.Chats()
	defer it.Close()

	var soonest *time.Time

	for it.Next() {
		_, cp, err := it.Value()
		if err != nil || cp.ReportTime == nil {
			continue
		}

		next := time.Date(now.Year(), now.Month(), now.Day(), cp.ReportTime.Hour, cp.ReportTime.Minute, 0, 0, now.Location())
		if next.Before(now) || next.Equal(now) {
			next = next.Add(24 * time.Hour)
		}

		if soonest == nil || next.Before(*soonest) {
			soonest = &next
		}
	}

	if soonest == nil {
		return now.Add(24 * time.Hour)
	}

	slog.Info("Sheduling next report generation", "time", soonest)

	return *soonest
}

func (w *ReportWorker) dispatchReports(ctx context.Context) {
	slog.Info("Sending reports...")
	now := time.Now()
	it := w.storage.Chats()
	defer it.Close()

	for it.Next() {
		chatID, cp, err := it.Value()
		if err != nil || cp.ReportTime == nil {
			continue
		}

		reportTime := time.Date(now.Year(), now.Month(), now.Day(), cp.ReportTime.Hour, cp.ReportTime.Minute, 0, 0, now.Location())

		if (reportTime.Before(now) || reportTime.Equal(now)) && cp.LastReportSent.Before(reportTime) {
			select {
			case <-ctx.Done():
				return
			case w.tasks <- reportTask{chatID: chatID, cp: cp}:
				// Task dispatched
			}
		}
	}
}

func (w *ReportWorker) sendReport(ctx context.Context, chatID int64, cp storage.ChatProfile) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	records, err := w.storage.GetRecords(chatID, monthStart)
	if err != nil {
		slog.Error("Failed to get records for report", "chatID", chatID, "error", err)
		return
	}

	stats := report.Generate(cp, records, time.Now())
	msg := report.Format(stats)

	// Send chart image with the report text as caption when available
	if len(stats.Chart) > 0 {
		_, err = w.bot.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:          chatID,
			MessageThreadID: cp.TopicID,
			Photo: &models.InputFileUpload{
				Filename: "spending_chart.png",
				Data:     bytes.NewReader(stats.Chart),
			},
			Caption:   msg,
			ParseMode: models.ParseModeHTML,
		})
	} else {
		_, err = w.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: cp.TopicID,
			Text:            msg,
			ParseMode:       models.ParseModeHTML,
		})
	}

	if err != nil {
		slog.Error("Failed to send daily report", "chatID", chatID, "error", err)
		return
	}

	cp.LastReportSent = now
	if err := w.storage.SetChat(chatID, &cp); err != nil {
		slog.Error("Failed to update LastReportSent", "chatID", chatID, "error", err)
	}
}
