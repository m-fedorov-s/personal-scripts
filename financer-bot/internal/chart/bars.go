package chart

import (
	"math"
	"slices"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

const (
	// plotHeight is the height below the header of the three plotted designs.
	plotHeight = 300
	// barShare is the part of a day's slot a bar fills.
	barShare = 0.62
	// gutterPad is the space between the plot and the labels in the gutter to its right.
	gutterPad = 4
	// breakAbove times the daily limit, or without one the median spending day: a day above it is tall, and may be drawn broken.
	breakAbove = 3
	// maxBroken is the most days drawn broken: the tallest; the axis covers any other tall day.
	maxBroken = 3
	// brokenRise is how far a broken bar reaches above the top gridline.
	brokenRise = 6
)

// gutterLabels are the stacks that sit right of the bars: the limit, the target, budget gone.
func (month spendingMonth) gutterLabels() (limit, target, gone block) {
	small := regular(minTextSize, inkSecondary)
	valueStyle := bold(labelSize, inkPrimary)
	if !month.hasLimit() {
		return nil, nil, nil
	}
	limit = block{{{small, "limit"}}, {{valueStyle, amount(month.limit)}}}
	if amountPerDay, ok := month.target(); ok {
		target = block{{{small, "target"}}, {{valueStyle, amount(amountPerDay)}}}
	} else if month.daysAfter() > 0 {
		word := "gone"
		if month.left() == 0 {
			word = "used up"
		}
		gone = block{{{small, "budget"}}, {{valueStyle, word}}}
	}
	return limit, target, gone
}

// drawBars draws one bar per day so far, coloured by the daily limit, the limit line, and dotted
// ghost bars at the target for the days after today.
func drawBars(month spendingMonth) *picture {
	pic, body := newPage(month, plotHeight)

	limitLabel, targetLabel, goneLabel := month.gutterLabels()
	var gutter vg.Length
	for _, label := range []block{limitLabel, targetLabel, goneLabel} {
		if label != nil {
			width, _ := label.size()
			gutter = max(gutter, gutterPad+width+textGap)
		}
	}

	target, hasTarget := month.target()
	byHeight := slices.Sorted(slices.Values(month.daily[:month.today]))
	slices.Reverse(byHeight)
	leftOut := 0
	for leftOut < min(maxBroken, len(byHeight)) && month.tall(byHeight[leftOut]) {
		leftOut++
	}
	top := float64(month.limit)
	if leftOut < len(byHeight) {
		top = max(top, float64(byHeight[leftOut]))
	}
	if !month.tall(target) {
		top = max(top, float64(target))
	}
	step := niceStep(top, 4)
	topTick := max(step, math.Ceil(top/step)*step)
	broken := func(value int64) bool { return float64(value) > topTick }
	headroom := 1.08
	for _, value := range append(month.daily[:month.today:month.today], target) {
		if broken(value) {
			headroom = 1.3
		}
	}

	graph := newPlot(month.daysInMonth)
	graph.X.Min, graph.X.Max = 0.5, float64(month.daysInMonth)+0.5
	graph.Y.Min, graph.Y.Max = 0, topTick*headroom
	graph.Y.Tick.Marker = amountTicks(step, 0, topTick)
	graph.Add(plotFunc(func(region draw.Canvas, graph *plot.Plot) {
		toX, toY := graph.Transforms(&region)
		half := (toX(2) - toX(1)) * barShare / 2
		zero := toY(0)
		columnTop := func(value int64) vg.Length {
			if broken(value) {
				return toY(topTick) + brokenRise
			}
			return toY(float64(value))
		}

		var brokenDays []int
		for day := 1; day <= month.today; day++ {
			value := month.daily[day-1]
			if value <= 0 {
				continue
			}
			fill := withinInk
			if month.hasLimit() && value > month.limit {
				fill = overInk
			}
			center := toX(float64(day))
			pic.bar(region, fill, rect(center-half, zero, center+half, columnTop(value)))
			if broken(value) {
				drawBreak(region, center-half, center+half, toY(topTick))
				brokenDays = append(brokenDays, day)
			}
		}
		if hasTarget {
			for day := month.today + 1; day <= month.daysInMonth; day++ {
				center := toX(float64(day))
				pic.ghost(region, rect(center-half, zero, center+half, columnTop(target)))
				if broken(target) {
					drawGhostBreak(region, center-half, center+half, toY(topTick))
				}
			}
		}
		if goneLabel != nil {
			pic.fillRect(region, withAlpha(overInk, 0.07),
				rect(toX(float64(month.today)+0.5), zero, toX(float64(month.daysInMonth)+0.5), toY(topTick)))
		}
		limitY := toY(float64(month.limit))
		if month.hasLimit() {
			pic.stroke(region, draw.LineStyle{Color: inkSecondary, Width: vg.Points(1.25)},
				vg.Point{X: region.Min.X, Y: limitY}, vg.Point{X: region.Max.X, Y: limitY})
		}

		bounds := rect(region.Min.X, region.Min.Y, body.Max.X, region.Max.Y)
		for _, day := range brokenDays {
			center := toX(float64(day))
			label := block{{{bold(noteSize, inkPrimary), amount(month.daily[day-1])}}}
			_, rowHeight := label.size()
			var candidates []candidate
			for row := 0; row < maxBroken; row++ {
				labelY := columnTop(month.daily[day-1]) + gutterPad + vg.Length(row)*(rowHeight+textGap)
				candidates = append(candidates, at(center, labelY, draw.XCenter, draw.YBottom),
					at(center+half, labelY, draw.XRight, draw.YBottom), at(center-half, labelY, draw.XLeft, draw.YBottom),
					at(center, labelY, draw.XRight, draw.YBottom), at(center, labelY, draw.XLeft, draw.YBottom))
			}
			pic.place(region, bounds, label, candidates...)
		}

		gutterX := region.Max.X + gutterPad
		if limitLabel != nil {
			limitAt, targetAt := beside(gutterX, limitY), []candidate(nil)
			if hasTarget {
				targetY := columnTop(target)
				targetAt = beside(gutterX, targetY)
				_, stackHeight := limitLabel.size()
				if vg.Length(math.Abs(float64(targetY-limitY))) < stackHeight+textGap {
					upper, lower := &limitAt, &targetAt
					if targetY > limitY {
						upper, lower = lower, upper
					}
					*upper = append([]candidate{at(gutterX, max(limitY, targetY)+textGap, draw.XLeft, draw.YBottom)}, *upper...)
					*lower = append([]candidate{at(gutterX, min(limitY, targetY)-textGap, draw.XLeft, draw.YTop)}, *lower...)
				}
			}
			pic.place(region, bounds, limitLabel, limitAt...)
			if targetLabel != nil {
				pic.placeOrOverlap(region, bounds, nil, targetLabel, targetAt...)
			}
		}
		if goneLabel != nil {
			var fromTop []candidate
			for labelY := toY(topTick); labelY > zero; labelY -= spreadStep {
				fromTop = append(fromTop, at(gutterX, labelY, draw.XLeft, draw.YTop))
			}
			pic.place(region, bounds, goneLabel, fromTop...)
		}
	}))
	graph.Draw(draw.Crop(body, 0, -gutter, 0, 0))
	return pic
}

// tall reports whether value is high enough for its bar to be drawn broken.
func (month spendingMonth) tall(value int64) bool {
	doubledBase, ok := month.doubledBreakBase()
	return ok && 2*value > breakAbove*doubledBase
}

// doubledBreakBase is twice the amount that tall multiplies by breakAbove: the daily limit or,
// without one, the median of the days so far that have spending. Doubling keeps the median of an
// even count, the mean of the middle two, whole. There is none with no limit and no spending yet.
func (month spendingMonth) doubledBreakBase() (int64, bool) {
	if month.hasLimit() {
		return 2 * month.limit, true
	}
	var spending []int64
	for _, value := range month.daily[:month.today] {
		if value > 0 {
			spending = append(spending, value)
		}
	}
	if len(spending) == 0 {
		return 0, false
	}
	slices.Sort(spending)
	middle := len(spending) / 2
	if len(spending)%2 == 0 {
		return spending[middle-1] + spending[middle], true
	}
	return 2 * spending[middle], true
}

// beside offers positions for a gutter stack centred on centerY, then moving away from it in 6 pt steps.
func beside(gutterX, centerY vg.Length) []candidate {
	candidates := []candidate{at(gutterX, centerY, draw.XLeft, draw.YCenter)}
	for offset := vg.Length(6); offset <= 48; offset += 6 {
		candidates = append(candidates, at(gutterX, centerY+offset, draw.XLeft, draw.YCenter), at(gutterX, centerY-offset, draw.XLeft, draw.YCenter))
	}
	return candidates
}
