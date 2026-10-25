package chart

import (
	"cmp"
	"image/color"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/plot/vg"
)

// verticalBars are the vertical gap-bar strokes of pic, left to right.
func verticalBars(pic *picture) []polyline {
	var bars []polyline
	for _, line := range pic.lines {
		if line.width == gapBarWidth && len(line.points) == 2 && line.points[0].X == line.points[1].X {
			bars = append(bars, line)
		}
	}
	slices.SortFunc(bars, func(first, second polyline) int { return cmp.Compare(first.points[0].X, second.points[0].X) })
	return bars
}

// dotAt is the box of the dot of the given fill centred on x.
func dotAt(t *testing.T, pic *picture, x vg.Length, fill color.NRGBA) vg.Rectangle {
	t.Helper()
	for _, drawn := range pic.areas {
		if box := boundsOf(drawn.outline); drawn.fill == fill && (box.Min.X+box.Max.X)/2 == x {
			return box
		}
	}
	require.Fail(t, "no dot", "fill %v at x %v", fill, x)
	return vg.Rectangle{}
}

// The layout checks cover text only, so this checks the drawn mark against the drawn dot.
func TestStillOverMarkStartsClearOfTheBudgetDot(t *testing.T) {
	for _, sc := range []scenario{
		{name: "budget blown", now: october(24), limit: 3300, daily: budgetBlown},
		{name: "one over the budget", now: october(20), limit: 3300, daily: append(repeat(5000, 19), 7301)},
	} {
		t.Run(sc.name, func(t *testing.T) {
			pic := drawRunningTotal(sc.month())
			bars := verticalBars(pic)
			require.NotEmpty(t, bars, "no still-over mark drawn")
			mark := bars[len(bars)-1]
			dot := dotAt(t, pic, mark.points[0].X, inkMuted)
			assert.GreaterOrEqual(t, float64(min(mark.points[0].Y, mark.points[1].Y)), float64(dot.Max.Y+textGap))
		})
	}
}

// As above: the dot is drawn over the bar, so the bar must start where the dot's ring ends.
func TestGapBarStartsAtTheEdgeOfTodaysDot(t *testing.T) {
	for _, sc := range scenarios {
		month := sc.month()
		if !month.hasLimit() || month.overBy() == 0 {
			continue
		}
		t.Run(sc.name, func(t *testing.T) {
			pic := drawRunningTotal(month)
			bars := verticalBars(pic)
			require.NotEmpty(t, bars, "no gap bar drawn")
			bar := bars[0]
			dot := dotAt(t, pic, bar.points[0].X, spendInk)
			low, high := min(bar.points[0].Y, bar.points[1].Y), max(bar.points[0].Y, bar.points[1].Y)
			assert.True(t, high <= dot.Min.Y || low >= dot.Max.Y, "gap bar %v..%v enters today's dot %v..%v", low, high, dot.Min.Y, dot.Max.Y)
			assert.GreaterOrEqual(t, float64(high-low), float64(minMarkLength))
		})
	}
}

func TestSpendLineIsDrawnOverTheGapBar(t *testing.T) {
	for _, sc := range scenarios {
		month := sc.month()
		if !month.hasLimit() || month.overBy() == 0 {
			continue
		}
		t.Run(sc.name, func(t *testing.T) {
			var barAt, spendAt []int
			for index, line := range drawRunningTotal(month).lines {
				switch line.width {
				case gapBarWidth:
					barAt = append(barAt, index)
				case spendLine.Width:
					spendAt = append(spendAt, index)
				}
			}
			require.NotEmpty(t, barAt, "no gap bar drawn")
			// The first spend-line stroke is the key's swatch.
			assert.Less(t, barAt[0], spendAt[len(spendAt)-1], "the gap bar is drawn over the spend line")
		})
	}
}

func TestRunningTotalDrawsTheBudgetDotOnlyBeforeTheLastDay(t *testing.T) {
	tests := []struct {
		sc   scenario
		want int
	}{
		{scenario{name: "day before the last", now: october(30), limit: 3300, daily: repeat(3400, 30)}, 2},
		{scenario{name: "last day", now: october(31), limit: 3300, daily: repeat(3400, 31)}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.sc.name, func(t *testing.T) {
			// The key's budget swatch is a dot of the same ink.
			assert.Equal(t, tt.want, countFill(drawRunningTotal(tt.sc.month()), inkMuted))
		})
	}
}

// The layout checks do not see the plot's own tick labels, so this reads the axis line off the image.
func TestRunningTotalLiftsTheDataARingAboveTheAxis(t *testing.T) {
	pic := drawRunningTotal(scenario{now: october(15), limit: 3300, daily: midMonthUnder}.month())
	var zeroY vg.Length
	for _, line := range pic.lines {
		if line.width == allowanceLine.Width {
			zeroY = line.points[0].Y // the key's swatch comes first, so the last one is the plot's
		}
	}
	width, height := pic.canvas.Size()
	pixels := pic.canvas.Image().Bounds()
	scale := float64(pixels.Dy()) / float64(height)
	var axisRows []int
	for row := pixels.Min.Y; row < pixels.Max.Y; row++ {
		inked := 0
		for column := pixels.Min.X; column < pixels.Max.X; column++ {
			if color.NRGBAModel.Convert(pic.canvas.Image().At(column, row)) == axisInk {
				inked++
			}
		}
		if float64(inked) > float64(width)*scale/2 {
			axisRows = append(axisRows, row)
		}
	}
	require.NotEmpty(t, axisRows, "no axis line found")
	axisY := height - vg.Length((float64(axisRows[0])+float64(axisRows[len(axisRows)-1])+1)/2/scale)
	assert.InDelta(t, float64(dotRing), float64(zeroY-axisY), 1)
}
