package handler

import (
	"financer/internal/storage"
	"github.com/go-telegram/bot"
)

func RegisterHandlers(b *bot.Bot, s storage.Storage) []bot.Option {
	msgHandler := &MessageHandler{BaseHandler{Storage: s}}
	startHandler := &StartHandler{BaseHandler{Storage: s}}
	setLimitHandler := &SetLimitHandler{BaseHandler{Storage: s}}
	setBalanceHandler := &SetBalanceHandler{BaseHandler{Storage: s}}
	reportHandler := &ReportHandler{BaseHandler{Storage: s}}
	toggleReportHandler := &ToggleReportHandler{BaseHandler{Storage: s}}

	return []bot.Option{
		bot.WithDefaultHandler(msgHandler.Handle),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, startHandler.Handle),
		bot.WithMessageTextHandler("/setLimit", bot.MatchTypePrefix, setLimitHandler.Handle),
		bot.WithMessageTextHandler("/setBalance", bot.MatchTypePrefix, setBalanceHandler.Handle),
		bot.WithMessageTextHandler("/report", bot.MatchTypeExact, reportHandler.Handle),
		bot.WithMessageTextHandler("/toggleReport", bot.MatchTypeExact, toggleReportHandler.Handle),
	}
}
