package chart

import "gonum.org/v1/plot/vg"

func grow(box vg.Rectangle, by vg.Length) vg.Rectangle {
	return rect(box.Min.X-by, box.Min.Y-by, box.Max.X+by, box.Max.Y+by)
}

// overlaps reports whether the interiors of two rectangles meet.
func overlaps(first, second vg.Rectangle) bool {
	return first.Min.X < second.Max.X && second.Min.X < first.Max.X &&
		first.Min.Y < second.Max.Y && second.Min.Y < first.Max.Y
}

func contains(outer, inner vg.Rectangle) bool {
	return inner.Min.X >= outer.Min.X && inner.Max.X <= outer.Max.X &&
		inner.Min.Y >= outer.Min.Y && inner.Max.Y <= outer.Max.Y
}

// segmentTouches reports whether the segment from-to has a point in box, by Liang-Barsky clipping.
func segmentTouches(from, to vg.Point, box vg.Rectangle) bool {
	low, high := 0.0, 1.0
	dx, dy := float64(to.X-from.X), float64(to.Y-from.Y)
	for _, edge := range [4]struct{ delta, room float64 }{
		{-dx, float64(from.X - box.Min.X)},
		{dx, float64(box.Max.X - from.X)},
		{-dy, float64(from.Y - box.Min.Y)},
		{dy, float64(box.Max.Y - from.Y)},
	} {
		switch {
		case edge.delta == 0 && edge.room < 0:
			return false
		case edge.delta < 0:
			low = max(low, edge.room/edge.delta)
		case edge.delta > 0:
			high = min(high, edge.room/edge.delta)
		}
	}
	return low <= high
}

// insidePolygon reports whether point lies inside the closed polygon outline, by ray casting.
func insidePolygon(outline []vg.Point, point vg.Point) bool {
	inside := false
	for i, j := 0, len(outline)-1; i < len(outline); j, i = i, i+1 {
		current, previous := outline[i], outline[j]
		if (current.Y > point.Y) != (previous.Y > point.Y) &&
			point.X < current.X+(point.Y-current.Y)*(previous.X-current.X)/(previous.Y-current.Y) {
			inside = !inside
		}
	}
	return inside
}

// polygonTouches reports whether a closed polygon and a rectangle share a point.
func polygonTouches(outline []vg.Point, box vg.Rectangle) bool {
	for i := range outline {
		if segmentTouches(outline[i], outline[(i+1)%len(outline)], box) {
			return true
		}
	}
	return insidePolygon(outline, box.Min)
}

// lineTouches reports whether a stroked line reaches into box. Growing box by half the stroke width
// over-reaches at box's corners by at most 0.21 of the width, which only errs towards a clash.
func lineTouches(line polyline, box vg.Rectangle) bool {
	grown := grow(box, line.width/2)
	for i := 1; i < len(line.points); i++ {
		if segmentTouches(line.points[i-1], line.points[i], grown) {
			return true
		}
	}
	return false
}
