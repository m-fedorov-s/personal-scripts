// Package chart renders the daily report's chart of a chat's spending this month.
package chart

import (
	"bytes"
	"fmt"
	"log/slog"
	"time"

	"gonum.org/v1/plot/vg/vgimg"

	"financer/internal/storage"
)

// PNG renders the chart of the UTC month of now from records and the chat's daily limit,
// in the chat's design, or as bars when that design cannot be drawn with limit.
func PNG(records []storage.Record, limit int64, design storage.ChartDesign, now time.Time) ([]byte, error) {
	pic := render(newMonth(records, limit, now), design)
	for _, label := range pic.missed {
		slog.Warn("Chart label left out: no free position", "design", design, "label", label)
	}
	for _, placed := range pic.texts {
		if placed.overlapping {
			slog.Warn("Chart label drawn overlapping: no free position", "design", design, "label", placed.text)
		}
	}
	var buf bytes.Buffer
	if _, err := (vgimg.PngCanvas{Canvas: pic.canvas}).WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func render(month spendingMonth, design storage.ChartDesign) *picture {
	if !design.Drawable(month.limit) {
		return drawBars(month)
	}
	switch design {
	case storage.ChartRunningTotal:
		return drawRunningTotal(month)
	case storage.ChartOverUnder:
		return drawOverUnder(month)
	case storage.ChartCalendar:
		return drawCalendar(month)
	default:
		return drawBars(month)
	}
}
