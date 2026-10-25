package chart

import (
	"image/color"
	"math"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// bar fills a column with slightly rounded top corners and records it as its rectangle.
func (pic *picture) bar(region draw.Canvas, fill color.NRGBA, box vg.Rectangle) {
	radius := min(vg.Points(1.6), box.Size().X/2, box.Size().Y)
	var path vg.Path
	path.Move(box.Min)
	path.Line(vg.Point{X: box.Min.X, Y: box.Max.Y - radius})
	path.Arc(vg.Point{X: box.Min.X + radius, Y: box.Max.Y - radius}, radius, math.Pi, -math.Pi/2)
	path.Line(vg.Point{X: box.Max.X - radius, Y: box.Max.Y})
	path.Arc(vg.Point{X: box.Max.X - radius, Y: box.Max.Y - radius}, radius, math.Pi/2, -math.Pi/2)
	path.Line(vg.Point{X: box.Max.X, Y: box.Min.Y})
	path.Close()
	region.SetColor(fill)
	region.Fill(path)
	pic.addArea(fill, corners(box)...)
}

// ghost outlines a column with dotted sides and top over a faint wash, for a day still to come.
func (pic *picture) ghost(region draw.Canvas, box vg.Rectangle) {
	fill := withAlpha(withinInk, 0.07)
	region.FillPolygon(fill, corners(box))
	sides := draw.LineStyle{Color: withAlpha(withinInk, 0.45), Width: vg.Points(0.8), Dashes: []vg.Length{vg.Points(1.2), vg.Points(2)}}
	region.StrokeLines(sides, []vg.Point{box.Min, {X: box.Min.X, Y: box.Max.Y}}, []vg.Point{{X: box.Max.X, Y: box.Min.Y}, box.Max})
	top := draw.LineStyle{Color: withinInk, Width: vg.Points(1.5), Dashes: []vg.Length{vg.Points(1.5), vg.Points(1.5)}}
	region.StrokeLines(top, []vg.Point{{X: box.Min.X, Y: box.Max.Y}, box.Max})
	pic.addArea(fill, corners(grow(box, vg.Points(0.75)))...)
}

// breakGap is the width of the cut across a broken column.
const breakGap = 2

// drawBreak cuts a column with a surface-coloured diagonal, to show it runs past the scale.
func drawBreak(region draw.Canvas, left, right, cutY vg.Length) {
	from, to := breakEnds(left, right, cutY)
	region.StrokeLine2(draw.LineStyle{Color: surface, Width: breakGap}, from.X, from.Y, to.X, to.Y)
}

// drawGhostBreak is drawBreak for a ghost, whose faint wash would hide a plain cut: it also edges
// the cut on both sides in the within ink.
func drawGhostBreak(region draw.Canvas, left, right, cutY vg.Length) {
	drawBreak(region, left, right, cutY)
	from, to := breakEnds(left, right, cutY)
	edge := draw.LineStyle{Color: withinInk, Width: vg.Points(1)}
	run := to.Sub(from)
	length := vg.Length(math.Hypot(float64(run.X), float64(run.Y)))
	normal := vg.Point{X: -run.Y, Y: run.X}.Scale((breakGap + edge.Width) / 2 / length)
	for _, side := range []vg.Point{normal, normal.Scale(-1)} {
		region.StrokeLine2(edge, from.X+side.X, from.Y+side.Y, to.X+side.X, to.Y+side.Y)
	}
}

// breakEnds are the ends of the cut across a column, a little past both its sides.
func breakEnds(left, right, cutY vg.Length) (from, to vg.Point) {
	return vg.Point{X: left - 1, Y: cutY - 3}, vg.Point{X: right + 1, Y: cutY + 3}
}

// plotFunc adapts a closure to plot.Plotter, so custom marks get the plot's data transforms.
type plotFunc func(region draw.Canvas, graph *plot.Plot)

func (paint plotFunc) Plot(region draw.Canvas, graph *plot.Plot) { paint(region, graph) }

// newPlot returns a plot of a month's days with recessive axes: hairline grid, no y-axis line, and
// tick labels in secondary ink, which survives Telegram's JPEG re-encoding better than muted ink.
func newPlot(daysInMonth int) *plot.Plot {
	graph := plot.New()
	graph.BackgroundColor = surface
	graph.X.Color = axisInk
	graph.X.Tick.Color = axisInk
	graph.X.Padding = 0
	tickLabel := regular(labelSize, inkSecondary)
	graph.X.Tick.Label.Font, graph.X.Tick.Label.Color = tickLabel.Font, tickLabel.Color
	graph.Y.Tick.Label.Font, graph.Y.Tick.Label.Color = tickLabel.Font, tickLabel.Color
	graph.X.Width = vg.Points(1)
	graph.X.Tick.Length = vg.Points(3)
	graph.X.Tick.Marker = dayTicks(daysInMonth)
	graph.Y.Width = 0
	graph.Y.Tick.Length = 0
	graph.Y.Padding = vg.Points(4)
	grid := plotter.NewGrid()
	grid.Vertical.Width = 0
	grid.Horizontal.Color = gridInk
	grid.Horizontal.Width = vg.Points(0.8)
	graph.Add(grid)
	return graph
}

// dayTicks labels day 1, every fifth day not within three days of the end, and the last day.
func dayTicks(daysInMonth int) plot.ConstantTicks {
	var ticks plot.ConstantTicks
	for day := 1; day <= daysInMonth; day++ {
		if day == 1 || day%5 == 0 && daysInMonth-day >= 3 || day == daysInMonth {
			ticks = append(ticks, plot.Tick{Value: float64(day), Label: strconv.Itoa(day)})
		}
	}
	return ticks
}

// niceStep is the smallest whole 1, 2, 2.5 or 5 times a power of ten that splits extent into at
// most maxSteps steps.
func niceStep(extent float64, maxSteps int) float64 {
	for magnitude := 1.0; ; magnitude *= 10 {
		for _, factor := range []float64{1, 2, 2.5, 5} {
			step := factor * magnitude
			if step == math.Trunc(step) && extent/step <= float64(maxSteps) {
				return step
			}
		}
	}
}

// amountTicks puts labelled gridlines on the multiples of step from low to high.
func amountTicks(step, low, high float64) plot.ConstantTicks {
	var ticks plot.ConstantTicks
	for value := math.Ceil(low/step) * step; value <= high; value += step {
		ticks = append(ticks, plot.Tick{Value: value, Label: amount(int64(math.Round(value)))})
	}
	return ticks
}

// dotRing is the radius of the surface-coloured ring every dot is drawn in.
const dotRing = 6

// dot draws a filled circle in a surface-coloured ring, recorded as the ring's bounding square.
func (pic *picture) dot(region draw.Canvas, fill color.NRGBA, center vg.Point) {
	const radius = 4
	region.DrawGlyph(draw.GlyphStyle{Color: surface, Radius: dotRing, Shape: draw.CircleGlyph{}}, center)
	region.DrawGlyph(draw.GlyphStyle{Color: fill, Radius: radius, Shape: draw.CircleGlyph{}}, center)
	pic.addArea(fill, corners(rect(center.X-dotRing, center.Y-dotRing, center.X+dotRing, center.Y+dotRing))...)
}
