package chart

import (
	"image/color"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

const (
	// minMarkLength keeps a short mark visible when its two ends nearly meet.
	minMarkLength = 4
	// gapBarWidth is the stroke of the bars that show a gap between two amounts.
	gapBarWidth = 3.5
	// runningGutter leaves room right of the plot for the dots drawn on the last day.
	runningGutter = 8
	// spreadStep is the distance between the offsets a label is tried at.
	spreadStep = 4
)

var (
	spendLine     = draw.LineStyle{Color: spendInk, Width: vg.Points(2.25)}
	paceLine      = draw.LineStyle{Color: inkSecondary, Width: vg.Points(1.5), Dashes: []vg.Length{vg.Points(5), vg.Points(3)}}
	allowanceLine = draw.LineStyle{Color: inkMuted, Width: vg.Points(1.25)}
)

// gapMark returns the ends of the bar from today's dot, centred at dotY, towards the allowance at
// allowanceY, which lies below the dot when the chat is over. The bar starts at the edge of the dot's
// ring, so the dot never covers it, and runs past the allowance when that is nearer than minMarkLength.
func gapMark(dotY, allowanceY vg.Length, over bool) (start, end vg.Length) {
	if over {
		start = dotY - dotRing
		return start, min(allowanceY, start-minMarkLength)
	}
	start = dotY + dotRing
	return start, max(allowanceY, start+minMarkLength)
}

// runningKey lists the key's entries; with gapInKey it also carries the label of the gap at today.
func (month spendingMonth) runningKey(gapInKey bool) []legendItem {
	items := []legendItem{{swatchWidth: 16, swatch: lineSwatch(spendLine), label: keyLabel("spent so far", "")}}
	if month.daysAfter() > 0 {
		items = append(items, legendItem{swatchWidth: 16, swatch: lineSwatch(paceLine), label: keyLabel("this pace: ", amount(month.pace()))})
	}
	items = append(items,
		legendItem{swatchWidth: 16, swatch: lineSwatch(allowanceLine), label: keyLabel("allowance ", amount(month.limit)+"/day")},
		legendItem{swatchWidth: 12, swatch: dotSwatch, label: keyLabel("budget ", amount(month.budget()))})
	if month.spent() > month.budget() && month.daysAfter() > 0 {
		items = append(items, legendItem{swatchWidth: 6, swatch: barSwatch(overInk), label: keyLabel("still ", amount(month.spent()-month.budget())+" over")})
	}
	if gapInKey {
		ink, gapAmount, word := month.gapStyle()
		items = append(items, legendItem{swatchWidth: 6, swatch: barSwatch(ink), label: keyLabel("", gapAmount+word)})
	}
	return items
}

// gapStyle is the colour, formatted amount and wording of the gap between spent and the allowance.
func (month spendingMonth) gapStyle() (color.NRGBA, string, string) {
	if month.overBy() > 0 {
		return overInk, amount(month.overBy()), " over allowance"
	}
	return withinInk, amount(-month.overBy()), " under allowance"
}

func barSwatch(fill color.NRGBA) func(*picture, draw.Canvas, vg.Rectangle) {
	return func(pic *picture, region draw.Canvas, box vg.Rectangle) {
		center := (box.Min.X + box.Max.X) / 2
		pic.fillRect(region, fill, rect(center-gapBarWidth/2, box.Min.Y-2, center+gapBarWidth/2, box.Max.Y+2))
	}
}

// drawRunningTotal draws the month's cumulative spend against the allowance diagonal, the gap
// between them, and the current pace projected to the month's end. It needs a daily limit. The
// gap is labelled beside it when there is room there, else in the key.
func drawRunningTotal(month spendingMonth) *picture {
	if pic := drawRunningTotalWith(month, false); len(pic.missed) == 0 {
		return pic
	}
	return drawRunningTotalWith(month, true)
}

func drawRunningTotalWith(month spendingMonth, gapInKey bool) *picture {
	key := legendRows(month.runningKey(gapInKey))
	pic, page := newPage(month, legendHeight(key)+sectionGap+plotHeight)
	body := drawLegend(pic, page, key)
	body.Max.Y -= sectionGap

	days, limit := float64(month.daysInMonth), float64(month.limit)
	spent, allowance, budget, pace := float64(month.spent()), float64(month.allowance()), float64(month.budget()), float64(month.pace())
	yMax := 1.08 * max(pace, allowance)

	graph := newPlot(month.daysInMonth)
	// Lifting the data a dot's ring above the axis keeps a dot at zero off the tick labels.
	graph.X.Padding = dotRing
	graph.X.Min, graph.X.Max = 0, days
	graph.Y.Min, graph.Y.Max = 0, yMax
	graph.Y.Tick.Marker = amountTicks(niceStep(yMax, 5), 0, yMax)
	graph.Add(plotFunc(func(region draw.Canvas, graph *plot.Plot) {
		toX, toY := graph.Transforms(&region)
		point := func(day, value float64) vg.Point { return vg.Point{X: toX(day), Y: toY(value)} }
		pic.fillGap(region, point, month.overUnder(), func(day float64) float64 { return limit * day })

		lastDay := min(days, yMax/limit)
		pic.stroke(region, allowanceLine, point(0, 0), point(lastDay, limit*lastDay))
		now := point(float64(month.today), spent)
		if month.daysAfter() > 0 {
			pic.stroke(region, paceLine, now, point(days, pace))
		}
		gapInk, gapAmount, gapWord := month.gapStyle()
		gapStart, gapEnd := gapMark(now.Y, toY(allowance), month.overBy() > 0)
		if month.overBy() != 0 {
			pic.stroke(region, draw.LineStyle{Color: gapInk, Width: gapBarWidth}, vg.Point{X: now.X, Y: gapStart}, vg.Point{X: now.X, Y: gapEnd})
		}
		// The gap bar can run along the spend line's last segment, and the line stays on top.
		totals := month.cumulative()
		spend := make([]vg.Point, len(totals))
		for day, total := range totals {
			spend[day] = point(float64(day), float64(total))
		}
		pic.stroke(region, spendLine, spend...)
		if budget <= yMax && month.daysAfter() > 0 {
			pic.dot(region, inkMuted, point(days, budget))
		}
		if month.spent() > month.budget() && month.daysAfter() > 0 {
			// The still-over mark starts textGap clear of the budget dot's ring, so it never covers the dot.
			markBottom := toY(budget) + dotRing + textGap
			pic.stroke(region, draw.LineStyle{Color: overInk, Width: gapBarWidth},
				vg.Point{X: toX(days), Y: markBottom}, vg.Point{X: toX(days), Y: max(toY(spent), markBottom+minMarkLength)})
		}
		pic.dot(region, spendInk, now)

		bounds := rect(region.Min.X, region.Min.Y, body.Max.X, region.Max.Y)
		if month.overBy() != 0 && !gapInKey {
			valueStyle := bold(labelSize, inkPrimary)
			oneLine := block{{{valueStyle, gapAmount}, {regular(labelSize, inkPrimary), gapWord}}}
			twoLines := block{{{valueStyle, gapAmount}}, {{regular(minTextSize, inkSecondary), gapWord[1:]}}}
			middle := (gapStart + gapEnd) / 2
			var candidates []candidate
			for _, offset := range spread(48) {
				candidates = append(candidates, at(now.X+8, middle+offset, draw.XLeft, draw.YCenter),
					at(now.X-8, middle+offset, draw.XRight, draw.YCenter))
			}
			if !pic.tryPlace(region, bounds, oneLine, candidates...) {
				pic.place(region, bounds, twoLines, candidates...)
			}
		}
	}))
	graph.Draw(draw.Crop(body, 0, -runningGutter, 0, 0))
	return pic
}

// spread returns 0, then ±spreadStep, ±2·spreadStep and so on up to ±limit, nearest first.
func spread(limit vg.Length) []vg.Length {
	offsets := []vg.Length{0}
	for offset := vg.Length(spreadStep); offset <= limit; offset += spreadStep {
		offsets = append(offsets, offset, -offset)
	}
	return offsets
}
