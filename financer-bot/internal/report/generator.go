package report

import (
	"fmt"
	"time"

	"financer/internal/storage"
)

type Stats struct {
	CurrentBalance int64
	Spending24h    int64
	MonthlyAverage float64
}

func Generate(cp storage.ChatProfile, records []storage.Record) Stats {
	now := time.Now()
	var spending24h int64
	var monthlyTotal int64

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	for _, r := range records {
		// 24h spending (from today's start to now)
		if r.Date.After(dayStart) {
			spending24h += int64(r.Amount)
		}
		// Monthly total
		if r.Date.After(monthStart) || r.Date.Equal(monthStart) {
			monthlyTotal += int64(r.Amount)
		}
	}

	daysInMonth := now.Day()
	monthlyAverage := float64(monthlyTotal) / float64(daysInMonth)

	return Stats{
		CurrentBalance: cp.CurrentBalance,
		Spending24h:    spending24h,
		MonthlyAverage: monthlyAverage,
	}
}

func Format(stats Stats) string {
	return fmt.Sprintf(
		"📊 <b>Daily Report</b>\n\n"+
			"💰 <b>Current Balance</b>: %d\n"+
			"💸 <b>Last 24h Spending</b>: %d\n"+
			"📈 <b>Monthly Daily Average</b>: %.2f",
		stats.CurrentBalance, stats.Spending24h, stats.MonthlyAverage,
	)
}
