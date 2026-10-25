package chart

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strings"

	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// picture is a chart being drawn, with every text and mark placed on it, in canvas points.
// The layout tests read it; rendering only needs the canvas.
type picture struct {
	canvas *vgimg.Canvas
	texts  []placedText
	areas  []*area
	lines  []polyline
	missed []string // labels left out because no candidate position was free
}

// placedText is one label as drawn: one or more runs of text treated as a single block.
type placedText struct {
	text    string
	box     vg.Rectangle
	inks    []color.NRGBA
	in      *area   // the rectangular area the text is drawn inside, or nil when it is on the surface
	beneath []*area // washes painted over the text after it is written, which it may lie under
	// overlapping is set on a label written over whatever lay at its spot, since none was free.
	overlapping bool
}

// area is a filled mark: a bar, cell, dot, icon, swatch or shaded region.
type area struct {
	outline []vg.Point
	fill    color.NRGBA
}

// polyline is a stroked line, recorded at its full width even when it is dashed.
type polyline struct {
	points []vg.Point
	width  vg.Length
}

// span is one run of text in one style.
type span struct {
	style text.Style
	text  string
}

// newPicture starts a picture height points tall, at 3x (216 DPI) unless that would make the long
// side wider than 1280 px, in which case the DPI drops to fit.
func newPicture(height vg.Length) *picture {
	dpi := min(216, math.Floor(1280*72/float64(max(canvasWidth, height))))
	return &picture{canvas: vgimg.NewWith(vgimg.UseWH(canvasWidth, height), vgimg.UseDPI(int(dpi)), vgimg.UseBackgroundColor(surface))}
}

func rect(minX, minY, maxX, maxY vg.Length) vg.Rectangle {
	return vg.Rectangle{Min: vg.Point{X: minX, Y: minY}, Max: vg.Point{X: maxX, Y: maxY}}
}

func corners(box vg.Rectangle) []vg.Point {
	return []vg.Point{box.Min, {X: box.Max.X, Y: box.Min.Y}, box.Max, {X: box.Min.X, Y: box.Max.Y}}
}

func (pic *picture) addArea(fill color.NRGBA, outline ...vg.Point) *area {
	added := &area{outline: outline, fill: fill}
	pic.areas = append(pic.areas, added)
	return added
}

// fillPolygon fills and records a polygon.
func (pic *picture) fillPolygon(region draw.Canvas, fill color.NRGBA, points ...vg.Point) *area {
	region.FillPolygon(fill, points)
	return pic.addArea(fill, points...)
}

// fillRect fills and records a rectangle.
func (pic *picture) fillRect(region draw.Canvas, fill color.NRGBA, box vg.Rectangle) *area {
	return pic.fillPolygon(region, fill, corners(box)...)
}

// stroke draws and records a line.
func (pic *picture) stroke(region draw.Canvas, style draw.LineStyle, points ...vg.Point) {
	region.StrokeLines(style, points)
	pic.lines = append(pic.lines, polyline{points: points, width: style.Width})
}

// block is lines of text drawn top to bottom and left-aligned. Each line is runs drawn left to
// right on the baseline of its first run, which must be the line's largest.
type block [][]span

func (label block) size() (width, height vg.Length) {
	for _, line := range label {
		var lineWidth vg.Length
		for _, run := range line {
			lineWidth += run.style.Width(run.text)
		}
		width, height = max(width, lineWidth), height+line[0].style.Height(line[0].text)
	}
	return width, height
}

func (label block) String() string {
	var lines []string
	for _, line := range label {
		var runs []string
		for _, run := range line {
			runs = append(runs, run.text)
		}
		lines = append(lines, strings.Join(runs, ""))
	}
	return strings.Join(lines, " / ")
}

// write draws label with its top-left corner at corner and records it as one text, drawn inside the
// area in, or on the surface when in is nil.
func (pic *picture) write(region draw.Canvas, corner vg.Point, in *area, label block) vg.Rectangle {
	placed := placedText{text: label.String(), box: vg.Rectangle{Min: corner, Max: corner}, in: in}
	top := corner.Y
	for _, line := range label {
		baseline := lineBaseline(top, line[0])
		penX := corner.X
		for _, run := range line {
			style := run.style
			anchor := vg.Point{X: penX, Y: baseline + style.FontExtents().Ascent - style.Font.Size}
			region.FillText(style, anchor, run.text)
			placed.box = union(placed.box, style.Rectangle(run.text).Add(anchor))
			placed.inks = append(placed.inks, color.NRGBAModel.Convert(style.Color).(color.NRGBA))
			penX += style.Width(run.text)
		}
		top -= line[0].style.Height(line[0].text)
	}
	pic.texts = append(pic.texts, placed)
	return placed.box
}

// lineBaseline is where gonum draws the baseline of a line whose top is at top and whose largest
// run, first, comes first: the placement of text/plain.go solved for the baseline.
func lineBaseline(top vg.Length, first span) vg.Length {
	box := first.style.Rectangle(first.text)
	return top - box.Max.Y - first.style.FontExtents().Ascent + first.style.Font.Size
}

func union(first, second vg.Rectangle) vg.Rectangle {
	return rect(min(first.Min.X, second.Min.X), min(first.Min.Y, second.Min.Y),
		max(first.Max.X, second.Max.X), max(first.Max.Y, second.Max.Y))
}

// candidate is a position offered to a label: the point its box is aligned to, and how.
type candidate struct {
	at     vg.Point
	xAlign draw.XAlignment
	yAlign draw.YAlignment
}

func at(anchorX, anchorY vg.Length, xAlign draw.XAlignment, yAlign draw.YAlignment) candidate {
	return candidate{at: vg.Point{X: anchorX, Y: anchorY}, xAlign: xAlign, yAlign: yAlign}
}

// tryPlace writes label at the first candidate where its box lies inside bounds and keeps textGap
// from everything drawn so far, over a surface halo that hides gridlines. It reports whether a
// candidate was free.
func (pic *picture) tryPlace(region draw.Canvas, bounds vg.Rectangle, label block, candidates ...candidate) bool {
	return pic.tryPlaceBeneath(region, bounds, nil, label, candidates...)
}

// tryPlaceBeneath is tryPlace for a label that the caller paints washes over afterwards, so they do
// not block it.
func (pic *picture) tryPlaceBeneath(region draw.Canvas, bounds vg.Rectangle, washes []*area, label block, candidates ...candidate) bool {
	for _, where := range candidates {
		box := label.boxAt(where)
		if !contains(bounds, box) {
			continue
		}
		if _, blocked := pic.obstacle(box, nil, nil, washes); blocked {
			continue
		}
		pic.writeOnHalo(region, box, washes, label)
		return true
	}
	return false
}

// placeOrOverlap is tryPlaceBeneath that never leaves label out. With no free candidate, it tries
// them all shifted sideways, spreadStep further each time, left before right, until every shifted
// candidate lies outside bounds. With none free there either, it writes label over whatever lies at
// the first of those spots where it fits inside bounds, or at the first candidate if none does.
func (pic *picture) placeOrOverlap(region draw.Canvas, bounds vg.Rectangle, washes []*area, label block, candidates ...candidate) {
	tried := slices.Clone(candidates)
	for shift, inside := vg.Length(spreadStep), true; inside; shift += spreadStep {
		inside = false
		for _, side := range []vg.Length{-shift, shift} {
			for _, where := range candidates {
				where.at.X += side
				inside = inside || contains(bounds, vg.Rectangle{Min: where.at, Max: where.at})
				tried = append(tried, where)
			}
		}
	}
	if pic.tryPlaceBeneath(region, bounds, washes, label, tried...) {
		return
	}
	fallback := candidates[0]
	if index := slices.IndexFunc(tried, func(where candidate) bool { return contains(bounds, label.boxAt(where)) }); index >= 0 {
		fallback = tried[index]
	}
	pic.writeOnHalo(region, label.boxAt(fallback), washes, label)
	pic.texts[len(pic.texts)-1].overlapping = true
}

// boxAt is the box label fills when placed at where.
func (label block) boxAt(where candidate) vg.Rectangle {
	width, height := label.size()
	left := where.at.X + width*vg.Length(where.xAlign)
	top := where.at.Y + height*(1+vg.Length(where.yAlign))
	return rect(left, top-height, left+width, top)
}

// writeOnHalo writes label in box over a surface halo that hides gridlines, beneath washes.
func (pic *picture) writeOnHalo(region draw.Canvas, box vg.Rectangle, washes []*area, label block) {
	_, height := label.size()
	region.FillPolygon(surface, corners(grow(box, textGap*0.75)))
	pic.write(region, box.Min.Add(vg.Point{Y: height}), nil, label)
	pic.texts[len(pic.texts)-1].beneath = washes
}

// place is tryPlace for a label that must appear: one with no free candidate is listed in
// pic.missed, which the layout tests treat as a failure.
func (pic *picture) place(region draw.Canvas, bounds vg.Rectangle, label block, candidates ...candidate) {
	if !pic.tryPlace(region, bounds, label, candidates...) {
		pic.missed = append(pic.missed, label.String())
	}
}

// obstacle names the first thing drawn so far that box, grown by textGap, would touch, other than
// self, the text being checked, in, the area that holds it, and beneath, the washes painted over it.
// self is nil for a label not yet written.
func (pic *picture) obstacle(box vg.Rectangle, self *placedText, in *area, beneath []*area) (string, bool) {
	grown := grow(box, textGap)
	for index := range pic.texts {
		if other := &pic.texts[index]; other != self && overlaps(grown, other.box) {
			return fmt.Sprintf("text %q", other.text), true
		}
	}
	for _, mark := range pic.areas {
		if mark != in && !slices.Contains(beneath, mark) && polygonTouches(mark.outline, grown) {
			return "an area", true
		}
	}
	for _, line := range pic.lines {
		if lineTouches(line, grown) {
			return "a line", true
		}
	}
	return "", false
}
