package chart

import (
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

const (
	swatchHeight = 10
	swatchGap    = 5  // between a swatch and its label
	itemGap      = 14 // between items on one row
	legendRowGap = 4
)

// legendItem is one key entry: a swatch drawn into a box, then its label.
type legendItem struct {
	swatchWidth vg.Length
	swatch      func(pic *picture, region draw.Canvas, box vg.Rectangle)
	label       block
}

func (item legendItem) width() vg.Length {
	labelWidth, _ := item.label.size()
	return item.swatchWidth + swatchGap + labelWidth
}

// legendRows fills rows left to right with as many items as fit across a page.
func legendRows(items []legendItem) [][]legendItem {
	var rows [][]legendItem
	var used vg.Length
	for _, item := range items {
		if len(rows) == 0 || used+itemGap+item.width() > pageWidth {
			rows = append(rows, nil)
			used = -itemGap
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], item)
		used += itemGap + item.width()
	}
	return rows
}

func legendRowHeight(row []legendItem) vg.Length {
	height := vg.Length(swatchHeight)
	for _, item := range row {
		_, labelHeight := item.label.size()
		height = max(height, labelHeight)
	}
	return height
}

func legendHeight(rows [][]legendItem) vg.Length {
	var height vg.Length
	for _, row := range rows {
		height += legendRowHeight(row) + legendRowGap
	}
	return height
}

// drawLegend draws rows across the top of region and returns the part of region below them.
func drawLegend(pic *picture, region draw.Canvas, rows [][]legendItem) draw.Canvas {
	top := region.Max.Y
	for _, row := range rows {
		rowHeight := legendRowHeight(row)
		middle := top - rowHeight/2
		left := region.Min.X
		for _, item := range row {
			item.swatch(pic, region, rect(left, middle-swatchHeight/2, left+item.swatchWidth, middle+swatchHeight/2))
			_, labelHeight := item.label.size()
			pic.write(region, vg.Point{X: left + item.swatchWidth + swatchGap, Y: middle + labelHeight/2}, nil, item.label)
			left += item.width() + itemGap
		}
		top -= rowHeight + legendRowGap
	}
	below := region
	below.Max.Y = top
	return below
}

func lineSwatch(style draw.LineStyle) func(*picture, draw.Canvas, vg.Rectangle) {
	return func(pic *picture, region draw.Canvas, box vg.Rectangle) {
		middle := (box.Min.Y + box.Max.Y) / 2
		pic.stroke(region, style, vg.Point{X: box.Min.X, Y: middle}, vg.Point{X: box.Max.X, Y: middle})
	}
}

func dotSwatch(pic *picture, region draw.Canvas, box vg.Rectangle) {
	pic.dot(region, inkMuted, vg.Point{X: (box.Min.X + box.Max.X) / 2, Y: (box.Min.Y + box.Max.Y) / 2})
}

// keyLabel is a legend label: a note, then optionally a value in bold.
func keyLabel(note, value string) block {
	line := []span{{regular(noteSize, inkSecondary), note}}
	if value != "" {
		line = append(line, span{bold(noteSize, inkPrimary), value})
	}
	return block{line}
}
