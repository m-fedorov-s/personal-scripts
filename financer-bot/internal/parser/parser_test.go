package parser

import (
	"financer/internal/storage"
	"testing"
	"time"
)

func TestParseMessage(t *testing.T) {
	now := time.Now()
	// The parser uses time.Now() for the initial date, which has local timezone and monotonic clock.
	// We need to match that for comparison if we want to be exact, or just compare Unix timestamps.
	initialDate := now
	today := initialDate

	tests := []struct {
		name     string
		text     string
		expected []storage.Record
	}{
		{
			name: "simple record",
			text: "coffee 300",
			expected: []storage.Record{
				{Date: today, Name: "coffee", Amount: 300},
			},
		},
		{
			name: "multiple records",
			text: "coffee 300\nlunch 1200",
			expected: []storage.Record{
				{Date: today, Name: "coffee", Amount: 300},
				{Date: today, Name: "lunch", Amount: 1200},
			},
		},
		{
			name: "record with date change",
			text: "01.01\ngift 5000",
			expected: []storage.Record{
				{Date: time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Now().UTC().Location()), Name: "gift", Amount: 5000},
			},
		},
		{
			name: "mixed date and records",
			text: "coffee 300\n02.01\nlunch 1200",
			expected: []storage.Record{
				{Date: today, Name: "coffee", Amount: 300},
				{Date: time.Date(now.Year(), 1, 2, 0, 0, 0, 0, time.Now().UTC().Location()), Name: "lunch", Amount: 1200},
			},
		},
		{
			name: "invalid lines",
			text: "just text\n100\n01.01.2023\n",
			expected: []storage.Record{
				{Date: today, Name: "", Amount: 100},
			},
		},
		{
			name: "multiple words in name",
			text: "grocery store 1500",
			expected: []storage.Record{
				{Date: today, Name: "grocery store", Amount: 1500},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMessage(tt.text)
			if len(got) != len(tt.expected) {
				t.Errorf("ParseMessage() length = %v, want %v", len(got), len(tt.expected))
				return
			}
			for i := range got {
				// Compare dates by year, month, day since the parser might have different sub-second precision
				y1, m1, d1 := got[i].Date.Date()
				y2, m2, d2 := tt.expected[i].Date.Date()
				if y1 != y2 || m1 != m2 || d1 != d2 {
					t.Errorf("ParseMessage()[%d].Date = %v, want %v", i, got[i].Date, tt.expected[i].Date)
				}
				if got[i].Name != tt.expected[i].Name {
					t.Errorf("ParseMessage()[%d].Name = %v, want %v", i, got[i].Name, tt.expected[i].Name)
				}
				if got[i].Amount != tt.expected[i].Amount {
					t.Errorf("ParseMessage()[%d].Amount = %v, want %v", i, got[i].Amount, tt.expected[i].Amount)
				}
			}
		})
	}
}

func TestParseMessage_Empty(t *testing.T) {
	got := ParseMessage("")
	if len(got) != 0 {
		t.Errorf("ParseMessage(\"\") = %v, want empty slice", got)
	}
}

func TestParseMessage_NoAmount(t *testing.T) {
	got := ParseMessage("no amount here")
	if len(got) != 0 {
		t.Errorf("ParseMessage(\"no amount here\") = %v, want empty slice", got)
	}
}
