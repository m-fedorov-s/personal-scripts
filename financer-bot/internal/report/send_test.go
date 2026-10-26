package report

import (
	"testing"
	"time"
)

func TestInDifferentUTCMonth(t *testing.T) {
	octoberStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	octoberMid := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	decemberEnd := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	octoberLastYear := time.Date(2025, 10, 15, 12, 0, 0, 0, time.UTC)
	utcPlus3 := time.FixedZone("UTC+3", 3*3600)

	tests := []struct {
		name        string
		prevBuiltAt *time.Time
		builtAt     time.Time
		want        bool
	}{
		{"same UTC month", &octoberStart, time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC), false},
		{"next month", &octoberMid, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), true},
		{"December to January", &decemberEnd, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"same month a year apart", &octoberLastYear, octoberMid, true},
		{"nil previous", nil, octoberMid, false},
		{"next local month, same UTC month", &octoberMid, time.Date(2026, 11, 1, 2, 30, 0, 0, utcPlus3), false}, // 31 October, 23:30 UTC
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inDifferentUTCMonth(tt.prevBuiltAt, tt.builtAt); got != tt.want {
				t.Errorf("inDifferentUTCMonth(%v, %v) = %v, want %v", tt.prevBuiltAt, tt.builtAt, got, tt.want)
			}
		})
	}
}
