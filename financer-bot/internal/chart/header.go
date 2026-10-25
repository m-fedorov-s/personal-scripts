package chart

import (
	"strings"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

const (
	iconSize = 10
	iconGap  = 6
	// sectionGap separates the header, a key and the plot or grid stacked down a page.
	sectionGap = 10
)

// icon draws a status icon centred on center.
type icon func(pic *picture, region draw.Canvas, center vg.Point)

type headerRow struct {
	icon     icon // nil for none
	indent   vg.Length
	gapAbove vg.Length
	text     block
}

// headerRows lays out the title and status lines for a page. A status line too wide for the page
// puts its detail on a second line.
func headerRows(month spendingMonth) []headerRow {
	rows := []headerRow{{text: block{{{bold(titleSize, inkPrimary), month.title()}}}}}
	status := func(mark icon, headline, detail string) headerRow {
		head := span{bold(statusSize, inkPrimary), headline}
		row := headerRow{icon: mark, indent: iconSize + iconGap, gapAbove: 3, text: block{{head}}}
		if detail == "" {
			return row
		}
		rest := span{regular(labelSize, inkSecondary), detail}
		if row.indent+head.style.Width(head.text)+rest.style.Width(rest.text) <= pageWidth {
			row.text = block{{head, rest}}
		} else {
			rest.text = strings.TrimLeft(detail, " ·")
			row.text = block{{head}, {rest}}
		}
		return row
	}
	if !month.hasLimit() {
		return append(rows, status(nil, "No daily limit set", ""))
	}
	return append(rows, status(month.allowanceStatus()), status(month.targetStatus()))
}

// newPage starts a picture with the header drawn, and returns the part of the page below the
// header, which is bodyHeight tall.
func newPage(month spendingMonth, bodyHeight vg.Length) (*picture, draw.Canvas) {
	rows := headerRows(month)
	headerHeight := vg.Length(sectionGap)
	for _, row := range rows {
		_, rowHeight := row.text.size()
		headerHeight += row.gapAbove + rowHeight
	}
	pic := newPicture(2*pageMargin + headerHeight + bodyHeight)
	page := draw.Crop(draw.New(pic.canvas), pageMargin, -pageMargin, pageMargin, -pageMargin)
	return pic, drawHeader(pic, page, rows)
}

// drawHeader draws rows across the top of page and returns the part of page below them.
func drawHeader(pic *picture, page draw.Canvas, rows []headerRow) draw.Canvas {
	top := page.Max.Y
	for _, row := range rows {
		top -= row.gapAbove
		first := row.text[0][0]
		if row.icon != nil {
			baseline := lineBaseline(top, first)
			row.icon(pic, page, vg.Point{X: page.Min.X + iconSize/2, Y: baseline + first.style.Font.Size*0.36})
		}
		box := pic.write(page, vg.Point{X: page.Min.X + row.indent, Y: top}, nil, row.text)
		top = box.Min.Y
	}
	below := page
	below.Max.Y = top - sectionGap
	return below
}

func iconOver(pic *picture, region draw.Canvas, center vg.Point) {
	half := vg.Length(iconSize) / 2
	pic.fillPolygon(region, overInk, vg.Point{X: center.X, Y: center.Y + half},
		vg.Point{X: center.X - half, Y: center.Y - half}, vg.Point{X: center.X + half, Y: center.Y - half})
}

func iconUnder(pic *picture, region draw.Canvas, center vg.Point) {
	half := vg.Length(iconSize) / 2
	pic.fillPolygon(region, withinInk, vg.Point{X: center.X, Y: center.Y - half},
		vg.Point{X: center.X - half, Y: center.Y + half}, vg.Point{X: center.X + half, Y: center.Y + half})
}

func iconTarget(pic *picture, region draw.Canvas, center vg.Point) {
	half := vg.Length(iconSize) / 2
	pic.ghost(region, rect(center.X-half*0.7, center.Y-half, center.X+half*0.7, center.Y+half))
}

func iconGone(pic *picture, region draw.Canvas, center vg.Point) {
	half := vg.Length(iconSize) / 2
	pic.fillRect(region, overInk, rect(center.X-half, center.Y-half, center.X+half, center.Y+half))
	cross := draw.LineStyle{Color: white, Width: vg.Points(1.6)}
	inset := half / 2
	region.StrokeLine2(cross, center.X-inset, center.Y-inset, center.X+inset, center.Y+inset)
	region.StrokeLine2(cross, center.X-inset, center.Y+inset, center.X+inset, center.Y-inset)
}
