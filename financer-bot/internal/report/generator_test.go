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

func TestGenerateChart_ReturnsPNG(t *testing.T) {
	now := time.Now()
	records := []storage.Record{
		{Amount: 100, Date: now.Add(-1 * time.Hour)},
		{Amount: 250, Date: now.Add(-25 * time.Hour)},
		{Amount: 50, Date: time.Date(now.Year(), now.Month(), 1, 10, 0, 0, 0, now.Location())},
	}

	pngBytes, err := generateSpendingChart(records, now)
	if err != nil {
		t.Fatalf("generateSpendingChart returned error: %v", err)
	}
	if len(pngBytes) == 0 {
		t.Fatal("generateSpendingChart returned empty bytes")
	}

	// Verify PNG magic bytes: 0x89 0x50 0x4E 0x47
	if pngBytes[0] != 0x89 || pngBytes[1] != 0x50 || pngBytes[2] != 0x4E || pngBytes[3] != 0x47 {
		t.Errorf("Output does not start with PNG magic bytes, got: %x %x %x %x",
			pngBytes[0], pngBytes[1], pngBytes[2], pngBytes[3])
	}
}

func TestGenerateChart_EmptyRecords(t *testing.T) {
	now := time.Now()
	pngBytes, err := generateSpendingChart([]storage.Record{}, now)
	if err != nil {
		t.Fatalf("generateSpendingChart with empty records returned error: %v", err)
	}
	if len(pngBytes) == 0 {
		t.Fatal("generateSpendingChart with empty records returned empty bytes")
	}
}

func TestGenerate_ChartPopulated(t *testing.T) {
	now := time.Now()
	cp := storage.ChatProfile{CurrentBalance: 500}
	records := []storage.Record{
		{Amount: 75, Date: now.Add(-2 * time.Hour)},
	}

	stats := Generate(cp, records)
	if len(stats.Chart) == 0 {
		t.Error("Expected Stats.Chart to be populated, got empty slice")
	}
}
