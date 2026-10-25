package chart

import (
	"image/color"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/plot/vg"
)

func TestOverUnderMarksTheMonthEndOnlyWhileTheBudgetIsGoneWithDaysLeft(t *testing.T) {
	tests := []struct {
		sc       scenario
		wantEnds []string
	}{
		{scenario{name: "budget gone", now: october(24), limit: 3300, daily: budgetBlown}, []string{"at best / 6,000 over"}},
		{scenario{name: "budget used up", now: october(20), limit: 3300, daily: append(repeat(5000, 19), 7300)}, []string{"at best / on budget"}},
		{scenario{name: "budget left", now: october(27), limit: 3300, daily: lateOver}, nil},
		{scenario{name: "last day over the budget", now: october(31), limit: 3300, daily: append(repeat(3000, 30), 20000)}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.sc.name, func(t *testing.T) {
			ends := slices.DeleteFunc(texts(drawOverUnder(tt.sc.month())), func(label string) bool { return !strings.HasPrefix(label, "at best") })
			assert.ElementsMatch(t, tt.wantEnds, ends)
		})
	}
}

func TestUnderLabelsWriteTheAmountWithoutASign(t *testing.T) {
	month := scenario{now: october(15), limit: 3300, daily: midMonthUnder}.month()
	assert.Contains(t, texts(drawOverUnder(month)), "11,700 under")
	assert.Contains(t, texts(drawRunningTotal(month)), "11,700 under allowance")

	keyed := scenario{now: october(3), limit: 3300, daily: []int64{800, 3100, 0}}.month()
	assert.NotEmpty(t, drawRunningTotalWith(keyed, false).missed, "the gap label fits beside the bar, so the key copy is not drawn")
	assert.Contains(t, texts(drawRunningTotal(keyed)), "6,000 under allowance")
}

func TestOverUnderEndLabelSitsBesideItsMarkUnderTheTint(t *testing.T) {
	pic := drawOverUnder(scenario{now: october(24), limit: 3300, daily: budgetBlown}.month())
	var label, mark vg.Rectangle
	for _, placed := range pic.texts {
		if strings.HasPrefix(placed.text, "at best") {
			label = placed.box
		}
	}
	tintRight := vg.Length(math.Inf(-1))
	for _, drawn := range pic.areas {
		box := boundsOf(drawn.outline)
		switch {
		case drawn.fill == overInk && isSquare(box, 2*dotRing):
			mark = box
		case drawn.fill == gapWash(1):
			tintRight = max(tintRight, box.Max.X)
		}
	}
	assert.InDelta(t, float64(mark.Min.Y+mark.Max.Y)/2, float64(label.Min.Y+label.Max.Y)/2, 1e-6, "centred on the end mark")
	assert.InDelta(t, float64(mark.Min.X+mark.Max.X)/2-10, float64(label.Max.X), 1e-6, "10 pt left of the end mark")

	// Left of the tint's right edge the halo lies on the tint, which is painted over it, so no pixel
	// there is bare surface.
	img := pic.canvas.Image()
	scale := pic.canvas.DPI() / 72
	_, height := pic.canvas.Size()
	halo := grow(label, textGap*0.75)
	var checked, bare int
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			pointX, pointY := vg.Length(float64(x)/scale), height-vg.Length(float64(y)/scale)
			if pointX <= halo.Min.X || pointX >= min(halo.Max.X, tintRight) || pointY <= halo.Min.Y || pointY >= halo.Max.Y {
				continue
			}
			checked++
			if color.NRGBAModel.Convert(img.At(x, y)) == surface {
				bare++
			}
		}
	}
	require.NotZero(t, checked, "the end label overlaps the tint")
	assert.Zero(t, bare, "of %d pixels where the end label's halo lies on the tint", checked)
}

func TestOverUnderMovesTheEndLabelSidewaysWhenNoSpotBesideItsMarkIsFree(t *testing.T) {
	pic := drawOverUnder(scenario{now: october(12), limit: 3300, daily: append(repeat(1000, 11), 4294967295)}.month())
	end := slices.IndexFunc(pic.texts, func(placed placedText) bool { return strings.HasPrefix(placed.text, "at best") })
	mark := slices.IndexFunc(pic.areas, func(drawn *area) bool {
		return drawn.fill == overInk && isSquare(boundsOf(drawn.outline), 2*dotRing)
	})
	require.True(t, end >= 0 && mark >= 0, "the end label and its mark are drawn")
	markBox := boundsOf(pic.areas[mark].outline)
	assert.False(t, pic.texts[end].overlapping)
	assert.InDelta(t, float64(markBox.Min.X+markBox.Max.X)/2-10+spreadStep, float64(pic.texts[end].box.Max.X), 1e-6,
		"one step right of its spots, which end 10 pt left of the mark")
	assert.Empty(t, problems(pic))
}

func TestOverUnderDrawsTheTodayLabelAtItsFirstSpotThatFitsWhenNoSpotIsFree(t *testing.T) {
	// The under wash fills the plot below zero, and no row of the label's spots fits between the zero
	// line and the header.
	pic := drawOverUnder(scenario{now: october(30), limit: 1000, daily: append(slices.Repeat([]int64{0, 0, 0, 0, 4000}, 5), 0, 0, 0, 0, 9000)}.month())
	label := slices.IndexFunc(pic.texts, func(placed placedText) bool { return placed.text == "1,000 under" })
	dot := slices.IndexFunc(pic.areas, func(drawn *area) bool { return drawn.fill == spendInk })
	require.True(t, label >= 0 && dot >= 0, "the today label and dot are drawn")
	box, dotBox := pic.texts[label].box, boundsOf(pic.areas[dot].outline)
	assert.True(t, pic.texts[label].overlapping)
	assert.InDelta(t, float64(dotBox.Min.X+dotBox.Max.X)/2-10, float64(box.Max.X), 1e-6,
		"10 pt left of today's dot, since its first spot, right of the dot, runs into the page margin")
	assert.InDelta(t, float64(dotBox.Min.Y+dotBox.Max.Y)/2+12, float64(box.Min.Y+box.Max.Y)/2, 1e-6, "12 pt above it")
	assert.Empty(t, problems(pic))
}

func TestOverUnderDrawsTheTodayLabelWholeWhenItsFirstSpotRunsOffTheImage(t *testing.T) {
	pic := drawOverUnder(scenario{now: october(27), limit: 1000, daily: slices.Repeat([]int64{0, 3000, 0}, 9)}.month())
	label := slices.IndexFunc(pic.texts, func(placed placedText) bool { return placed.text == "on the allowance" })
	require.True(t, label >= 0, "the today label is drawn")
	assert.True(t, pic.texts[label].overlapping)
	assert.Empty(t, problems(pic))
}

// The layout checks cover text only, so this checks the key's icons against the drawn dots.
func TestOverUnderKeyIconsKeepClearOfEveryDot(t *testing.T) {
	for _, sc := range scenarios {
		month := sc.month()
		if !month.hasLimit() {
			continue
		}
		t.Run(sc.name, func(t *testing.T) {
			pic := drawOverUnder(month)
			var icons, dots []vg.Rectangle
			for _, drawn := range pic.areas {
				switch box := boundsOf(drawn.outline); {
				case isSquare(box, iconSize):
					icons = append(icons, box)
				case isSquare(box, 2*dotRing):
					dots = append(dots, box)
				}
			}
			require.NotEmpty(t, dots)
			for _, keyIcon := range icons {
				for _, dot := range dots {
					assert.False(t, overlaps(grow(keyIcon, textGap), dot), "icon %v is within %v of the dot %v", keyIcon, textGap, dot)
				}
			}
		})
	}
}

func isSquare(box vg.Rectangle, side vg.Length) bool {
	return math.Abs(float64(box.Max.X-box.Min.X-side)) < 1e-6 && math.Abs(float64(box.Max.Y-box.Min.Y-side)) < 1e-6
}
