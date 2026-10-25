package chart

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"financer/internal/storage"
)

func TestMonthQuantities(t *testing.T) {
	tests := []struct {
		name                                 string
		sc                                   scenario
		spent, allowance, overBy, left, pace int64
		daysAfter                            int
		target                               int64
		hasTarget                            bool
	}{
		{name: "day 1 counts as passed before any record", sc: scenario{now: october(1), limit: 3300},
			allowance: 3300, overBy: -3300, left: 102300, daysAfter: 30, target: 3410, hasTarget: true},
		{name: "target is rounded down at the start of the month", sc: scenario{now: october(2), limit: 3300},
			allowance: 6600, overBy: -6600, left: 102300, daysAfter: 29, target: 3527, hasTarget: true},
		{name: "target is rounded down", sc: scenario{now: october(15), limit: 3300, daily: midMonthUnder},
			spent: 37800, allowance: 49500, overBy: -11700, left: 64500, pace: 78120, daysAfter: 16, target: 4031, hasTarget: true},
		{name: "last day has no target", sc: scenario{now: october(31), limit: 3300, daily: repeat(3000, 31)},
			spent: 93000, allowance: 102300, overBy: -9300, left: 9300, pace: 93000},
		{name: "one under the budget keeps a zero target", sc: scenario{now: october(20), limit: 3300, daily: append(repeat(5000, 19), 7299)},
			spent: 102299, allowance: 66000, overBy: 36299, left: 1, pace: 158563, daysAfter: 11, hasTarget: true},
		{name: "spent equal to the budget leaves no target", sc: scenario{now: october(20), limit: 3300, daily: append(repeat(5000, 19), 7300)},
			spent: 102300, allowance: 66000, overBy: 36300, left: 0, pace: 158565, daysAfter: 11},
		{name: "records outside the month are ignored", sc: scenario{now: october(15), limit: 3300, daily: midMonthUnder, extra: []storage.Record{
			{Date: time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC), Amount: 50000},
			{Date: october(15).Add(time.Second), Amount: 50000},
		}}, spent: 37800, allowance: 49500, overBy: -11700, left: 64500, pace: 78120, daysAfter: 16, target: 4031, hasTarget: true},
		{name: "a record above the int32 range is summed whole", sc: scenario{now: october(10), limit: 3300, daily: append(repeat(1000, 9), 4294967295)},
			spent: 4294976295, allowance: 33000, overBy: 4294943295, left: -4294873995, pace: 13314426514, daysAfter: 21},
		{name: "no limit", sc: scenario{now: october(15), limit: 0, daily: midMonthUnder},
			spent: 37800, overBy: 37800, left: -37800, pace: 78120, daysAfter: 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			month := tt.sc.month()
			assert.Equal(t, tt.spent, month.spent(), "spent")
			assert.Equal(t, tt.allowance, month.allowance(), "allowance")
			assert.Equal(t, tt.overBy, month.overBy(), "overBy")
			assert.Equal(t, tt.left, month.left(), "left")
			assert.Equal(t, tt.pace, month.pace(), "pace")
			assert.Equal(t, tt.daysAfter, month.daysAfter(), "daysAfter")
			target, ok := month.target()
			assert.Equal(t, tt.hasTarget, ok, "has target")
			assert.Equal(t, tt.target, target, "target")
		})
	}
}

func TestMondayOffsetAndCalendarRows(t *testing.T) {
	tests := []struct {
		name         string
		now          time.Time
		offset, rows int
	}{
		{"a February that starts on a Monday fits four rows", time.Date(2027, 2, 14, 20, 0, 0, 0, time.UTC), 0, 4},
		{"October 2026 starts on a Thursday", october(1), 3, 5},
		{"a 30-day month starting on a Sunday needs six rows", time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC), 6, 6},
		{"a 31-day month starting on a Saturday needs six rows", time.Date(2026, 8, 1, 20, 0, 0, 0, time.UTC), 5, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			month := newMonth(nil, 3300, tt.now)
			assert.Equal(t, tt.offset, month.mondayOffset(), "offset")
			assert.Equal(t, tt.rows, month.calendarRows(), "rows")
		})
	}
}

func TestNewMonthFilesRecordsUnderTheirUTCDay(t *testing.T) {
	local := time.Date(2026, 10, 15, 1, 0, 0, 0, time.FixedZone("UTC+3", 3*3600)) // 14 October, 22:00 UTC
	month := newMonth([]storage.Record{{Date: local, Amount: 700}}, 3300, october(20))
	assert.Equal(t, int64(700), month.daily[13])
	assert.Equal(t, int64(0), month.daily[14])
}

func TestNewMonthReadsNowInUTC(t *testing.T) {
	now := time.Date(2026, 11, 1, 1, 0, 0, 0, time.FixedZone("UTC+3", 3*3600))
	month := newMonth([]storage.Record{{Date: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC), Amount: 700}}, 3300, now)
	assert.Equal(t, time.October, month.start.Month())
	assert.Equal(t, 31, month.daysInMonth)
	assert.Equal(t, 31, month.today)
	assert.Equal(t, int64(700), month.spent())
}
