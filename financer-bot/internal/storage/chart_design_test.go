package storage

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestChartDesign_Next(t *testing.T) {
	tests := []struct {
		name  string
		from  ChartDesign
		limit int64
		want  ChartDesign
	}{
		{"bars to running total", ChartBars, 3300, ChartRunningTotal},
		{"running total to over/under", ChartRunningTotal, 3300, ChartOverUnder},
		{"over/under to calendar", ChartOverUnder, 3300, ChartCalendar},
		{"calendar wraps to bars", ChartCalendar, 3300, ChartBars},
		{"bars to calendar without a limit", ChartBars, 0, ChartCalendar},
		{"calendar to bars without a limit", ChartCalendar, 0, ChartBars},
		{"stored running total to calendar without a limit", ChartRunningTotal, -100, ChartCalendar},
		{"stored over/under to calendar without a limit", ChartOverUnder, 0, ChartCalendar},
		{"unknown value counts as bars", ChartDesign(9), 3300, ChartRunningTotal},
		{"negative value counts as bars", ChartDesign(-1), 0, ChartCalendar},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.Next(tt.limit); got != tt.want {
				t.Errorf("ChartDesign(%d).Next(%d) = %d, want %d", tt.from, tt.limit, got, tt.want)
			}
		})
	}
}

func TestChatProfile_DecodeFrom_ProfileSavedBeforeChartDesignDecodesAsBars(t *testing.T) {
	type savedProfile struct {
		TopicID  int
		DayLimit int64
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(savedProfile{TopicID: 7, DayLimit: 1500}); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	var decoded ChatProfile
	if err := decoded.DecodeFrom(buf.Bytes()); err != nil {
		t.Fatalf("DecodeFrom() error = %v", err)
	}

	if decoded.DayLimit != 1500 {
		t.Errorf("DayLimit = %d, want 1500", decoded.DayLimit)
	}
	if decoded.ChartDesign != ChartBars {
		t.Errorf("ChartDesign = %d, want %d", decoded.ChartDesign, ChartBars)
	}
}

func TestChatProfile_DecodeFrom_RoundTripsEveryChartDesign(t *testing.T) {
	for design := ChartBars; design < chartDesignCount; design++ {
		original := ChatProfile{DayLimit: 1500, ChartDesign: design}
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(original); err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}

		var decoded ChatProfile
		if err := decoded.DecodeFrom(buf.Bytes()); err != nil {
			t.Fatalf("DecodeFrom() error = %v", err)
		}

		if decoded.DayLimit != original.DayLimit || decoded.ChartDesign != design {
			t.Errorf("DecodeFrom() = %+v, want %+v", decoded, original)
		}
	}
}
