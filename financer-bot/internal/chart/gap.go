package chart

import (
	"image/color"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func gapWash(value float64) color.NRGBA {
	if value > 0 {
		return withAlpha(overInk, 0.16)
	}
	return withAlpha(withinInk, 0.14)
}

// fillGap washes the space between a series and a reference line from day 0 to day len(gap)-1:
// red where the series is above, blue where below, split where they cross. gap[day] is the series
// minus the reference on that day. It returns the red washes.
func (pic *picture) fillGap(region draw.Canvas, point func(day, value float64) vg.Point, gap []int64, reference func(day float64) float64) (above []*area) {
	wash := func(value float64, points ...vg.Point) {
		filled := pic.fillPolygon(region, gapWash(value), points...)
		if value > 0 {
			above = append(above, filled)
		}
	}
	for day := 1; day < len(gap); day++ {
		from, to := float64(day-1), float64(day)
		before, after := float64(gap[day-1]), float64(gap[day])
		if before*after < 0 {
			cross := from + before/(before-after)
			wash(before, point(from, reference(from)), point(from, reference(from)+before), point(cross, reference(cross)))
			wash(after, point(cross, reference(cross)), point(to, reference(to)+after), point(to, reference(to)))
			continue
		}
		if before == 0 && after == 0 {
			continue
		}
		wash(before+after, point(from, reference(from)), point(from, reference(from)+before),
			point(to, reference(to)+after), point(to, reference(to)))
	}
	return above
}
