package report

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"financer/internal/storage"
)

type Stats struct {
	CurrentBalance int64
	Spending24h    int64
	MonthlyAverage float64
	Chart          []byte
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

	chartPNG, err := generateSpendingChart(records, now)
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

// generateSpendingChart builds a bar chart of daily spending for the current month
// and returns the PNG-encoded image as a byte slice.
func generateSpendingChart(records []storage.Record, now time.Time) ([]byte, error) {
	daysInMonth := now.Day()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Aggregate spending per day (1-indexed, day 1 = index 0)
	dailySpending := make([]float64, daysInMonth)
	for _, r := range records {
		if (r.Date.After(monthStart) || r.Date.Equal(monthStart)) && !r.Date.After(now) {
			day := r.Date.Day() // 1-based
			if day >= 1 && day <= daysInMonth {
				dailySpending[day-1] += float64(r.Amount)
			}
		}
	}

	// Build plotter.Values
	values := make(plotter.Values, daysInMonth)
	copy(values, dailySpending)

	p := plot.New()
	p.Title.Text = fmt.Sprintf("Daily Spending — %s %d", now.Month().String(), now.Year())
	p.Y.Label.Text = "Amount"
	p.BackgroundColor = color.White

	bars, err := plotter.NewBarChart(values, vg.Points(12))
	if err != nil {
		return nil, fmt.Errorf("creating bar chart: %w", err)
	}
	bars.LineStyle.Width = vg.Length(0)
	bars.Color = color.RGBA{R: 70, G: 130, B: 180, A: 255} // steel blue

	p.Add(bars)

	// Build X-axis labels: show every 5th day to avoid crowding
	labels := make([]string, daysInMonth)
	for i := 0; i < daysInMonth; i++ {
		day := i + 1
		if day == 1 || day%5 == 0 || day == daysInMonth {
			labels[i] = fmt.Sprintf("%d", day)
		} else {
			labels[i] = ""
		}
	}
	p.NominalX(labels...)

	// Render to in-memory PNG
	const (
		widthPx  = 800
		heightPx = 400
		dpi      = 96
	)
	img := image.NewRGBA(image.Rect(0, 0, widthPx, heightPx))
	canvas := vgimg.NewWith(vgimg.UseImage(img))
	p.Draw(draw.New(canvas))

	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas.Image()); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
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
