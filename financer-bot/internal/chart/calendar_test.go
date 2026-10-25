package chart

import (
	"image/color"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/plot/vg"
)

func TestCellInkReachesMinimumContrastOnEveryFill(t *testing.T) {
	for _, fill := range append(blueSteps, overInk) {
		assert.GreaterOrEqual(t, contrast(inkOn(fill), fill), minContrast, "fill %v", fill)
	}
}

// fillOf is cellFill's colour for value, failing the test on an error.
func fillOf(t *testing.T, month spendingMonth, value int64) color.NRGBA {
	t.Helper()
	fill, err := month.cellFill(value)
	require.NoError(t, err, "value %d", value)
	return fill
}

func TestCellFillFollowsQuartersOfTheLimit(t *testing.T) {
	month := scenario{now: october(15), limit: 3300}.month()
	for value, want := range map[int64]int{0: 0, 1: 1, 825: 1, 826: 2, 1650: 2, 2475: 3, 2476: 4, 3300: 4} {
		assert.Equal(t, blueSteps[want], fillOf(t, month, value), "value %d", value)
	}
	assert.Equal(t, overInk, fillOf(t, month, 3301))
}

func TestCellFillWithoutALimitFollowsTheLargestDay(t *testing.T) {
	month := scenario{now: october(3), daily: []int64{400, 0, 1000}}.month()
	assert.Equal(t, blueSteps[2], fillOf(t, month, 400))
	assert.Equal(t, blueSteps[4], fillOf(t, month, 1000))
	assert.Equal(t, blueSteps[0], fillOf(t, scenario{now: october(3)}.month(), 0))
}

func TestCellFillRefusesAPositiveValueWithoutAScale(t *testing.T) {
	_, err := scenario{now: october(3)}.month().cellFill(500)
	assert.Error(t, err)
}

func TestCalendarDrawsAGhostCellForEachDayAfterTodayAndTheTargetInTomorrowsOnly(t *testing.T) {
	month := scenario{now: october(15), limit: 3300, daily: midMonthUnder}.month()
	pic := drawCalendar(month)
	// the key's swatch is a ghost cell too
	assert.Equal(t, month.daysAfter()+1, countFill(pic, withAlpha(withinInk, 0.05)))
	assert.Len(t, slices.DeleteFunc(texts(pic), func(label string) bool { return label != "4,031" }), 1)
}

func TestCellAmountShrinksThenAbbreviates(t *testing.T) {
	assert.Equal(t, vg.Length(labelSize), cellAmount(21500, true, inkPrimary).style.Font.Size)
	assert.Equal(t, "333,000", cellAmount(333000, false, inkPrimary).text)
	assert.Equal(t, "1.2M", cellAmount(1234567, true, white).text)
}
