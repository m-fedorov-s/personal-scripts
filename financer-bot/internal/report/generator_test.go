package report

import (
	"testing"
	"time"

	"financer/internal/storage"
)

func TestGenerate(t *testing.T) {
	now := time.Now()
	cp := storage.ChatProfile{
		CurrentBalance: 1000,
	}

	records := []storage.Record{
		{Amount: 100, Date: now.Add(-1 * time.Hour)},                                            // Today
		{Amount: 200, Date: now.Add(-25 * time.Hour)},                                           // Yesterday
		{Amount: 300, Date: time.Date(now.Year(), now.Month(), 1, 10, 0, 0, 0, now.Location())}, // Start of month
	}

	stats := Generate(cp, records)

	if stats.CurrentBalance != 1000 {
		t.Errorf("Expected balance 1000, got %d", stats.CurrentBalance)
	}

	// 24h spending should only include the 100 record (since Generate uses dayStart)
	if stats.Spending24h != 100 {
		t.Errorf("Expected 24h spending 100, got %d", stats.Spending24h)
	}

	expectedMonthlyTotal := int64(100 + 200 + 300)
	expectedAverage := float64(expectedMonthlyTotal) / float64(now.Day())

	if stats.MonthlyAverage != expectedAverage {
		t.Errorf("Expected monthly average %f, got %f", expectedAverage, stats.MonthlyAverage)
	}
}
