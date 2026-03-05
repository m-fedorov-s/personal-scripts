package handler

import (
	"financer/internal/storage"
	"github.com/go-telegram/bot"
)

func RegisterHandlers(b *bot.Bot, s storage.Storage, reschedule chan *storage.DayTime) []bot.Option {
	baseHandler := BaseHandler{Storage: s, Reschedule: reschedule}
	msgHandler := &MessageHandler{baseHandler}
	startHandler := &StartHandler{baseHandler}
	setLimitHandler := &SetLimitHandler{baseHandler}
	setBalanceHandler := &SetBalanceHandler{baseHandler}
	setReportTimeHandler := &SetReportTimeHandler{baseHandler}
	reportHandler := &ReportHandler{baseHandler}
	toggleReportHandler := &ToggleReportHandler{baseHandler}

	return []bot.Option{
		bot.WithDefaultHandler(msgHandler.Handle),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, startHandler.Handle),
		bot.WithMessageTextHandler("/setLimit", bot.MatchTypePrefix, setLimitHandler.Handle),
		bot.WithMessageTextHandler("/setBalance", bot.MatchTypePrefix, setBalanceHandler.Handle),
		bot.WithMessageTextHandler("/setReportTime", bot.MatchTypePrefix, setReportTimeHandler.Handle),
		bot.WithMessageTextHandler("/report", bot.MatchTypeExact, reportHandler.Handle),
		bot.WithMessageTextHandler("/toggleReport", bot.MatchTypeExact, toggleReportHandler.Handle),
	}
}
