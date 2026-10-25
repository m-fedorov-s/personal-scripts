package storage

// ChartDesign is the chat's daily-chart design. Values are persisted: append, never renumber.
// ChartBars is the zero value on purpose, so profiles saved without the field decode to it.
type ChartDesign int

const (
	ChartBars ChartDesign = iota
	ChartRunningTotal
	ChartOverUnder
	ChartCalendar
	chartDesignCount
)

// Drawable reports whether design can be drawn for a chat with the given daily limit. The running
// total and over/under measure spending against the limit, so they need one above zero.
func (design ChartDesign) Drawable(limit int64) bool {
	switch design {
	case ChartBars, ChartCalendar:
		return true
	case ChartRunningTotal, ChartOverUnder:
		return limit > 0
	default:
		return false
	}
}

// Next is the following design in declaration order, wrapping round, that can be drawn with limit.
// A design that cannot be drawn with limit counts as ChartBars, as it does when rendering.
func (design ChartDesign) Next(limit int64) ChartDesign {
	if !design.Drawable(limit) {
		design = ChartBars
	}
	for {
		design = (design + 1) % chartDesignCount
		if design.Drawable(limit) {
			return design
		}
	}
}
