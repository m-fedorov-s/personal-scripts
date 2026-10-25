package chart

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBarsDrawsAGhostForEachDayAfterToday(t *testing.T) {
	tests := []struct {
		sc         scenario
		wantGhosts int
	}{
		{scenario{name: "mid-month", now: october(15), limit: 3300, daily: midMonthUnder}, 16},
		{scenario{name: "day 30", now: october(30), limit: 3300, daily: repeat(3300, 30)}, 1},
		{scenario{name: "last day", now: october(31), limit: 3300, daily: repeat(3000, 31)}, 0},
		{scenario{name: "budget gone", now: october(24), limit: 3300, daily: budgetBlown}, 0},
		{scenario{name: "no limit", now: october(15), daily: midMonthUnder}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.sc.name, func(t *testing.T) {
			month := tt.sc.month()
			want := tt.wantGhosts
			if _, ok := month.target(); ok {
				want++ // the header's target icon is drawn as a ghost too
			}
			assert.Equal(t, want, countFill(drawBars(month), withAlpha(withinInk, 0.07)))
		})
	}
}

func TestBarsBreaksOnlyTheThreeTallestDays(t *testing.T) {
	month := scenario{now: october(15), limit: 3300, daily: append(repeat(1000, 11), 21500, 18000, 25000, 30000)}.month()
	labels := texts(drawBars(month))
	assert.Subset(t, labels, []string{"21,500", "25,000", "30,000"})
	assert.NotContains(t, labels, "18,000")
}

func TestBarsLeavesATallTargetOutOfTheScaleAndTheThreeBreaks(t *testing.T) {
	month := scenario{now: october(28), limit: 3300, daily: append(repeat(300, 25), 10000, 11000, 12000)}.month()
	target, _ := month.target()
	assert.True(t, month.tall(target), "target %d is not tall", target)
	assert.Subset(t, texts(drawBars(month)), []string{"10,000", "11,000", "12,000"})
}

func TestBarsDrawsTheTargetWhenNoSpotInTheGutterIsFree(t *testing.T) {
	// Four tall days set the scale, so no spot in the gutter for the small target clears both the axis and the limit.
	pic := drawBars(scenario{now: october(25), limit: 1000, daily: append(repeat(6000, 4), repeat(300, 21)...)}.month())
	target := slices.IndexFunc(pic.texts, func(placed placedText) bool { return placed.text == "target / 116" })
	require.True(t, target >= 0, "the target is drawn")
	assert.False(t, pic.texts[target].overlapping)
	assert.Empty(t, problems(pic))
}

func TestBarsWithoutALimitDrawsOnlyBlueBarsAndNoLimitOrTarget(t *testing.T) {
	for _, limit := range []int64{0, -100} {
		month := scenario{now: october(15), limit: limit, daily: midMonthUnder}.month()
		pic := drawBars(month)
		assert.Zero(t, countFill(pic, overInk), "limit %d", limit)
		assert.Equal(t, 14, countFill(pic, withinInk), "limit %d", limit)           // one per day with spending
		assert.Zero(t, countFill(pic, withAlpha(overInk, 0.07)), "limit %d", limit) // the wash behind "budget gone"
		assert.Empty(t, pic.lines, "limit %d", limit)
		assert.Contains(t, texts(pic), "No daily limit set", "limit %d", limit)
		for _, label := range texts(pic) {
			for _, limitOnly := range []string{"limit", "target", "budget"} {
				assert.False(t, strings.HasPrefix(label, limitOnly), label)
			}
		}
	}
}

func TestBarsWithoutALimitBreaksDaysAboveThreeTimesTheMedianSpendingDay(t *testing.T) {
	tests := []struct {
		name       string
		daily      []int64
		value      string
		wantBroken bool
	}{
		{"odd count, above", []int64{1000, 2000, 6001}, "6,001", true},
		{"odd count, equal", []int64{1000, 2000, 6000}, "6,000", false},
		{"even count, above the unrounded median", []int64{1000, 2000, 3001, 7502}, "7,502", true},
		{"even count, below the unrounded median", []int64{1000, 2000, 3001, 7501}, "7,501", false},
		{"days without spending are not in the median", []int64{0, 0, 0, 1000, 3100}, "3,100", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			month := scenario{now: october(len(tt.daily)), daily: tt.daily}.month()
			assert.Equal(t, tt.wantBroken, slices.Contains(texts(drawBars(month)), tt.value))
		})
	}
}

func TestBarsWithoutALimitOrSpendingBreaksNothing(t *testing.T) {
	assert.False(t, scenario{now: october(15)}.month().tall(1))
}
