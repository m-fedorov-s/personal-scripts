package chart

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"

	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

const (
	cellGap     = 3
	cellHeight  = 40
	cellPadding = 3
	// weekdayHeight is the height of the row of weekday names above the grid.
	weekdayHeight = 18
	cellWidth     = vg.Length(pageWidth-6*cellGap) / 7
)

// blueSteps shade amounts up to the scale, lightest for nothing spent.
var blueSteps = []color.NRGBA{rgb(0xcde2fb), rgb(0x9ec5f4), rgb(0x6da7ec), rgb(0x3987e5), rgb(0x256abf)}

var weekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// scale is the amount the blue steps run up to: the daily limit, or without one the month's
// largest day.
func (month spendingMonth) scale() int64 {
	if month.hasLimit() {
		return month.limit
	}
	return slices.Max(month.daily)
}

// cellFill is red for a day over the daily limit and otherwise the blue step for value's quarter
// of the scale. A positive value with no positive scale to place it on is an error.
func (month spendingMonth) cellFill(value int64) (color.NRGBA, error) {
	scale := month.scale()
	switch {
	case month.hasLimit() && value > month.limit:
		return overInk, nil
	case value <= 0:
		return blueSteps[0], nil
	case scale <= 0:
		return color.NRGBA{}, fmt.Errorf("no blue step for %d on a scale of %d", value, scale)
	}
	return blueSteps[min((4*value+scale-1)/scale, 4)], nil
}

// inkOn is whichever of primary ink and white contrasts more with fill.
func inkOn(fill color.NRGBA) color.NRGBA {
	if contrast(inkPrimary, fill) >= contrast(white, fill) {
		return inkPrimary
	}
	return white
}

// cellAmount writes value as large as fits inside a cell: in full at 13, 12 or 11 pt, else abbreviated.
func cellAmount(value int64, heavy bool, ink color.Color) span {
	room := cellWidth - 2*cellPadding
	style := regular
	if heavy {
		style = bold
	}
	sizes := []font.Length{labelSize, noteSize, minTextSize}
	for _, write := range []func(int64) string{amount, shortAmount} {
		for _, size := range sizes {
			if run := (span{style(size, ink), write(value)}); run.style.Width(run.text) <= room {
				return run
			}
		}
	}
	return span{style(minTextSize, ink), shortAmount(value)}
}

func (month spendingMonth) calendarKey() []legendItem {
	strip := legendItem{swatchWidth: 40, swatch: func(pic *picture, region draw.Canvas, box vg.Rectangle) {
		width := box.Size().X / vg.Length(len(blueSteps))
		for index, fill := range blueSteps {
			left := box.Min.X + width*vg.Length(index)
			pic.fillRect(region, fill, rect(left, box.Min.Y, left+width, box.Max.Y))
		}
	}, label: keyLabel("0 to ", amount(month.scale()))}
	ghost := legendItem{swatchWidth: 14, swatch: ghostCell, label: keyLabel("dotted: days after today", "")}
	if !month.hasLimit() {
		return []legendItem{strip, ghost}
	}
	over := legendItem{swatchWidth: 14, swatch: func(pic *picture, region draw.Canvas, box vg.Rectangle) {
		pic.fillRect(region, overInk, box)
	}, label: keyLabel("over ", amount(month.limit))}
	return []legendItem{strip, over, ghost}
}

// ghostCell draws a day still to come: a near-surface fill inside a dotted border.
func ghostCell(pic *picture, region draw.Canvas, box vg.Rectangle) {
	pic.fillRect(region, withAlpha(withinInk, 0.05), box)
	border := draw.LineStyle{Color: withAlpha(withinInk, 0.6), Width: vg.Points(1), Dashes: []vg.Length{vg.Points(1.5), vg.Points(1.5)}}
	region.StrokeLines(border, append(corners(box), box.Min))
}

// drawCalendar draws the month as a Monday-first grid of days, each shaded by what was spent.
func drawCalendar(month spendingMonth) *picture {
	rows := month.calendarRows()
	gridHeight := vg.Length(rows)*cellHeight + vg.Length(rows-1)*cellGap
	key := legendRows(month.calendarKey())
	pic, body := newPage(month, weekdayHeight+gridHeight+sectionGap+legendHeight(key))

	weekdayStyle := regular(noteSize, inkSecondary)
	for column, name := range weekdays {
		center := body.Min.X + vg.Length(column)*(cellWidth+cellGap) + cellWidth/2
		width := weekdayStyle.Width(name)
		pic.write(body, vg.Point{X: center - width/2, Y: body.Max.Y}, nil, block{{{weekdayStyle, name}}})
	}
	gridTop := body.Max.Y - weekdayHeight
	target, hasTarget := month.target()
	for day := 1; day <= month.daysInMonth; day++ {
		index := month.mondayOffset() + day - 1
		left := body.Min.X + vg.Length(index%7)*(cellWidth+cellGap)
		top := gridTop - vg.Length(index/7)*(cellHeight+cellGap)
		box := rect(left, top-cellHeight, left+cellWidth, top)
		corner := vg.Point{X: left + cellPadding, Y: top - cellPadding}
		if day > month.today {
			ghostCell(pic, body, box)
			cell := pic.areas[len(pic.areas)-1]
			pic.write(body, corner, cell, block{{{regular(noteSize, inkSecondary), strconv.Itoa(day)}}})
			if day == month.today+1 && hasTarget {
				run := cellAmount(target, false, inkPrimary)
				pic.write(body, cellBottom(box, run), cell, block{{run}})
			}
			continue
		}
		value := month.daily[day-1]
		// A day the scale cannot shade is drawn without a fill rather than in a wrong colour.
		var cell *area
		ink := inkOn(surface)
		if fill, err := month.cellFill(value); err == nil {
			cell = pic.fillRect(body, fill, box)
			ink = inkOn(fill)
		}
		pic.write(body, corner, cell, block{{{regular(noteSize, ink), strconv.Itoa(day)}}})
		run := cellAmount(value, month.hasLimit() && value > month.limit, ink)
		pic.write(body, cellBottom(box, run), cell, block{{run}})
		if day == month.today {
			outline := grow(box, 1)
			pic.stroke(body, draw.LineStyle{Color: inkPrimary, Width: vg.Points(2)}, append(corners(outline), outline.Min)...)
		}
	}
	below := body
	below.Max.Y = gridTop - gridHeight - sectionGap
	drawLegend(pic, below, key)
	return pic
}

// cellBottom is the top-left corner that puts run on the bottom line of a cell.
func cellBottom(box vg.Rectangle, run span) vg.Point {
	_, height := block{{run}}.size()
	return vg.Point{X: box.Min.X + cellPadding, Y: box.Min.Y + cellPadding + height}
}
