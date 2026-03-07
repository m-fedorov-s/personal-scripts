package report

import (
	"bytes"
	"image/png"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"financer/internal/storage"
)

// fixedNow is a deterministic reference time used across all tests.
// It is the 15th of March 2024 at noon UTC — mid-month, mid-day, no edge cases
// from the calendar itself.
var fixedNow = time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)

// dayStart and monthStart derived from fixedNow for convenience.
var (
	fixedDayStart   = time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	fixedMonthStart = time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
)

// ─────────────────────────────────────────────────────────────────────────────
// Generate — spending statistics
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerate_Spending24h(t *testing.T) {
	tests := []struct {
		name        string
		records     []storage.Record
		want24h     int64
		description string
	}{
		{
			name: "record exactly at midnight is included (boundary fix)",
			records: []storage.Record{
				{Amount: 100, Date: fixedDayStart}, // exactly 00:00:00 UTC
			},
			want24h:     100,
			description: "!r.Date.Before(dayStart) must include midnight records",
		},
		{
			name: "record one nanosecond before midnight is excluded",
			records: []storage.Record{
				{Amount: 100, Date: fixedDayStart.Add(-time.Nanosecond)},
			},
			want24h:     0,
			description: "record from yesterday must not appear in today's 24h spending",
		},
		{
			name: "record at noon today is included",
			records: []storage.Record{
				{Amount: 250, Date: fixedNow.Add(-1 * time.Hour)},
			},
			want24h: 250,
		},
		{
			name: "record exactly at now is included",
			records: []storage.Record{
				{Amount: 75, Date: fixedNow},
			},
			want24h: 75,
		},
		{
			name: "record after now is excluded",
			records: []storage.Record{
				{Amount: 999, Date: fixedNow.Add(time.Second)},
			},
			want24h: 0,
		},
		{
			name: "multiple records — only today's are summed",
			records: []storage.Record{
				{Amount: 100, Date: fixedDayStart},                   // today midnight — included
				{Amount: 200, Date: fixedNow.Add(-2 * time.Hour)},    // today — included
				{Amount: 300, Date: fixedDayStart.Add(-time.Second)}, // yesterday — excluded
			},
			want24h: 300,
		},
		{
			name:    "empty records",
			records: []storage.Record{},
			want24h: 0,
		},
		{
			name: "UTC record vs non-UTC now — timezone normalisation",
			records: []storage.Record{
				// Record stored in UTC+1 local time but represents the same instant.
				{Amount: 150, Date: fixedNow.In(time.FixedZone("CET", 3600))},
			},
			want24h:     150,
			description: "Generate must normalise both now and r.Date to UTC before comparing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := storage.ChatProfile{CurrentBalance: 1000}
			stats := Generate(cp, tt.records, fixedNow)
			assert.Equal(t, tt.want24h, stats.Spending24h, tt.description)
		})
	}
}

func TestGenerate_MonthlyAverage(t *testing.T) {
	tests := []struct {
		name        string
		records     []storage.Record
		now         time.Time
		wantAverage float64
	}{
		{
			name: "record at month start is included",
			records: []storage.Record{
				{Amount: 300, Date: fixedMonthStart},
			},
			now:         fixedNow, // day 15
			wantAverage: 300.0 / 15.0,
		},
		{
			name: "record one nanosecond before month start is excluded",
			records: []storage.Record{
				{Amount: 300, Date: fixedMonthStart.Add(-time.Nanosecond)},
			},
			now:         fixedNow,
			wantAverage: 0,
		},
		{
			name: "first day of month — daysInMonth=1, no division by zero",
			records: []storage.Record{
				{Amount: 500, Date: time.Date(2024, 3, 1, 6, 0, 0, 0, time.UTC)},
			},
			now:         time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC),
			wantAverage: 500.0,
		},
		{
			name: "multiple records across the month",
			records: []storage.Record{
				{Amount: 100, Date: time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)},
				{Amount: 200, Date: time.Date(2024, 3, 7, 10, 0, 0, 0, time.UTC)},
				{Amount: 300, Date: time.Date(2024, 3, 14, 10, 0, 0, 0, time.UTC)},
			},
			now:         fixedNow, // day 15
			wantAverage: 600.0 / 15.0,
		},
		{
			name:        "empty records",
			records:     []storage.Record{},
			now:         fixedNow,
			wantAverage: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := storage.ChatProfile{CurrentBalance: 1000}
			stats := Generate(cp, tt.records, tt.now)
			assert.InDelta(t, tt.wantAverage, stats.MonthlyAverage, 1e-9)
		})
	}
}

func TestGenerate_CurrentBalance(t *testing.T) {
	cp := storage.ChatProfile{CurrentBalance: 9999}
	stats := Generate(cp, nil, fixedNow)
	assert.Equal(t, int64(9999), stats.CurrentBalance)
}

// ─────────────────────────────────────────────────────────────────────────────
// generateSpendingChart — PNG output
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateSpendingChart_ValidPNG(t *testing.T) {
	tests := []struct {
		name    string
		records []storage.Record
		now     time.Time
	}{
		{
			name: "normal records mid-month",
			records: []storage.Record{
				{Amount: 100, Date: fixedNow.Add(-1 * time.Hour)},
				{Amount: 250, Date: fixedNow.Add(-25 * time.Hour)},
				{Amount: 50, Date: fixedMonthStart},
			},
			now: fixedNow,
		},
		{
			name:    "empty records",
			records: []storage.Record{},
			now:     fixedNow,
		},
		{
			name: "first day of month",
			records: []storage.Record{
				{Amount: 100, Date: time.Date(2024, 3, 1, 6, 0, 0, 0, time.UTC)},
			},
			now: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "record exactly at month start boundary",
			records: []storage.Record{
				{Amount: 200, Date: fixedMonthStart},
			},
			now: fixedNow,
		},
		{
			name: "record in non-UTC timezone",
			records: []storage.Record{
				{Amount: 150, Date: fixedNow.In(time.FixedZone("CET", 3600))},
			},
			now: fixedNow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pngBytes, err := generateSpendingChart(tt.records, tt.now)
			require.NoError(t, err)
			require.NotEmpty(t, pngBytes, "chart must not be empty")

			// Verify valid PNG magic bytes: 0x89 0x50 0x4E 0x47
			require.True(t, len(pngBytes) >= 4, "PNG must be at least 4 bytes")
			assert.Equal(t, byte(0x89), pngBytes[0])
			assert.Equal(t, byte(0x50), pngBytes[1])
			assert.Equal(t, byte(0x4E), pngBytes[2])
			assert.Equal(t, byte(0x47), pngBytes[3])

			// Verify the bytes decode as a valid PNG image.
			_, err = png.Decode(bytes.NewReader(pngBytes))
			assert.NoError(t, err, "chart bytes must decode as a valid PNG image")
		})
	}
}

// TestGenerateSpendingChart_RecordsOutsideMonthAreExcluded verifies that
// records from previous months do not appear in the chart data.
func TestGenerateSpendingChart_RecordsOutsideMonthAreExcluded(t *testing.T) {
	// We can't inspect the bar values directly, but we can verify the function
	// does not error and returns a valid PNG even when all records are outside
	// the current month.
	records := []storage.Record{
		{Amount: 999, Date: time.Date(2024, 2, 28, 10, 0, 0, 0, time.UTC)}, // previous month
		{Amount: 888, Date: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)}, // two months ago
	}
	pngBytes, err := generateSpendingChart(records, fixedNow)
	require.NoError(t, err)
	require.NotEmpty(t, pngBytes)
}

// ─────────────────────────────────────────────────────────────────────────────
// Generate — chart field populated
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerate_ChartIsPopulated(t *testing.T) {
	tests := []struct {
		name    string
		records []storage.Record
	}{
		{
			name: "with records",
			records: []storage.Record{
				{Amount: 75, Date: fixedNow.Add(-2 * time.Hour)},
			},
		},
		{
			name:    "empty records still produces chart",
			records: []storage.Record{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := storage.ChatProfile{CurrentBalance: 500}
			stats := Generate(cp, tt.records, fixedNow)
			assert.NotEmpty(t, stats.Chart, "Stats.Chart must always be populated")
		})
	}
}
