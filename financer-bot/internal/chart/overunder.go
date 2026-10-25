package chart

import (
	"slices"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/recorder"
)

// overUnderGutter holds the key: an icon clear of the plot's dots, then a word up to 32 pt wide.
const overUnderGutter = dotRing + textGap + iconSize + iconGap + 32

// keySide is which side of the zero line a key entry sits on.
type keySide int

const (
	aboveZero keySide = iota
	belowZero
)

// overUnderText is "1,200 over", "1,200 under", or zeroText when value is zero.
func overUnderText(value int64, zeroText string) block {
	word := " over"
	switch {
	case value == 0:
		return block{{{bold(labelSize, inkPrimary), zeroText}}}
	case value < 0:
		word, value = " under", -value
	}
	return block{{{bold(labelSize, inkPrimary), amount(value)}, {regular(labelSize, inkPrimary), word}}}
}

// drawOverUnder draws how far the month's spend has run over or under the allowance each day, red
// above zero and blue below. It needs a daily limit.
func drawOverUnder(month spendingMonth) *picture {
	pic, body := newPage(month, plotHeight)

	series := month.overUnder()
	end, hasEnd := month.spent()-month.budget(), month.budgetGone() && month.daysAfter() > 0
	low, high := min(0, slices.Min(series)), max(0, slices.Max(series))
	step := niceStep(1.2*float64(max(high-low, 3*month.limit)), 5)
	yMin, yMax := float64(low)-0.3*step, float64(high)+0.3*step

	graph := newPlot(month.daysInMonth)
	graph.X.Min, graph.X.Max = 0, float64(month.daysInMonth)
	graph.Y.Min, graph.Y.Max = yMin, yMax
	graph.Y.Tick.Marker = amountTicks(step, yMin, yMax)
	graph.Add(plotFunc(func(region draw.Canvas, graph *plot.Plot) {
		// Everything but the end label is recorded and replayed after it, so the over tint is painted
		// over the label and its halo instead of being cut by the halo.
		endLabelRegion := region
		var afterEndLabel recorder.Canvas
		region.Canvas = &afterEndLabel
		toX, toY := graph.Transforms(&region)
		point := func(day, value float64) vg.Point { return vg.Point{X: toX(day), Y: toY(value)} }
		overTint := pic.fillGap(region, point, series, func(float64) float64 { return 0 })
		zero := toY(0)
		pic.stroke(region, draw.LineStyle{Color: inkSecondary, Width: vg.Points(1)}, vg.Point{X: region.Min.X, Y: zero}, vg.Point{X: region.Max.X, Y: zero})
		line := make([]vg.Point, len(series))
		for day, value := range series {
			line[day] = point(float64(day), float64(value))
		}
		pic.stroke(region, spendLine, line...)
		now := line[month.today]
		var endPoint vg.Point
		if hasEnd {
			endPoint = point(float64(month.daysInMonth), float64(end))
			pic.dot(region, overInk, endPoint)
		}
		pic.dot(region, spendInk, now)

		bounds := rect(region.Min.X, body.Min.Y, body.Max.X, body.Max.Y)
		// The key goes first, so the labels placed after it keep clear of its icons.
		keyX := region.Max.X + dotRing + textGap
		keyEntry := func(word string, mark icon, side keySide) {
			yAlign, direction := draw.YBottom, vg.Length(1)
			if side == belowZero {
				yAlign, direction = draw.YTop, -1
			}
			var candidates []candidate
			for offset := vg.Length(0); offset <= 60; offset += spreadStep {
				candidates = append(candidates, at(keyX+iconSize+iconGap, zero+direction*(textGap+offset), draw.XLeft, yAlign))
			}
			label := block{{{regular(noteSize, inkSecondary), word}}}
			if !pic.tryPlace(region, bounds, label, candidates...) {
				pic.missed = append(pic.missed, label.String())
				return
			}
			box := pic.texts[len(pic.texts)-1].box
			mark(pic, region, vg.Point{X: keyX + iconSize/2, Y: (box.Min.Y + box.Max.Y) / 2})
		}
		keyEntry("over", iconOver, aboveZero)
		keyEntry("under", iconUnder, belowZero)

		var nearToday []candidate
		for _, offset := range spread(60) {
			nearToday = append(nearToday, at(now.X+10, now.Y+12+offset, draw.XLeft, draw.YCenter),
				at(now.X-10, now.Y+12+offset, draw.XRight, draw.YCenter))
		}
		pic.placeOrOverlap(region, bounds, nil, overUnderText(month.overBy(), "on the allowance"), nearToday...)
		if hasEnd {
			label := append(block{{{regular(minTextSize, inkSecondary), "at best"}}}, overUnderText(end, "on budget")...)
			var leftOfEnd []candidate
			for _, offset := range spread(60) {
				leftOfEnd = append(leftOfEnd, at(endPoint.X-10, endPoint.Y+offset, draw.XRight, draw.YCenter))
			}
			pic.placeOrOverlap(endLabelRegion, bounds, overTint, label, leftOfEnd...)
		}
		// ReplayOn fails only for a font outside gonum's Liberation collection; textStyle uses Liberation.
		_ = afterEndLabel.ReplayOn(endLabelRegion.Canvas)
	}))
	graph.Draw(draw.Crop(body, 0, -overUnderGutter, 0, 0))
	return pic
}
