package report

import (
	"fmt"
	"log/slog"
	"time"

	"financer/internal/chart"
	"financer/internal/storage"
)

type Stats struct {
	CurrentBalance int64
	Spending24h    int64
	MonthlyAverage float64
	Chart          []byte
}

// Generate computes spending statistics from records relative to now.
//
// now is passed explicitly so callers can control the reference time, making
// the function deterministic and testable without mocking time.Now().
//
// All boundary calculations are performed in UTC to avoid timezone mismatches
// between the server locale and the UTC timestamps stored in the database.
func Generate(cp storage.ChatProfile, records []storage.Record, now time.Time) Stats {
	// Normalise to UTC so comparisons are consistent regardless of the
	// timezone of the server running the bot.
	now = now.UTC()

	// dayStart is the beginning of today in UTC (00:00:00.000000000).
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// monthStart is the first instant of the current month in UTC.
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var spending24h int64
	var monthlyTotal int64

	for _, r := range records {
		rUTC := r.Date.UTC()

		// 24h spending: records from today's midnight (inclusive) up to now.
		// Bug fix: use !rUTC.Before(dayStart) instead of rUTC.After(dayStart)
		// so that records timestamped exactly at midnight are included.
		if !rUTC.Before(dayStart) && !rUTC.After(now) {
			spending24h += int64(r.Amount)
		}

		// Monthly total: records from the first of the month (inclusive) up to now.
		if !rUTC.Before(monthStart) && !rUTC.After(now) {
			monthlyTotal += int64(r.Amount)
		}
	}

	daysInMonth := now.Day()
	monthlyAverage := float64(monthlyTotal) / float64(daysInMonth)

	chartPNG, err := chart.PNG(records, cp.DayLimit, cp.ChartDesign, now)
	if err != nil {
		slog.Error("Failed to generate spending chart", "error", err)
	}

	return Stats{
		CurrentBalance: cp.CurrentBalance,
		Spending24h:    spending24h,
		MonthlyAverage: monthlyAverage,
		Chart:          chartPNG,
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
