package chart

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
)

// amount formats a whole amount with comma thousands separators and a true minus sign.
func amount(value int64) string {
	return strings.Replace(humanize.Comma(value), "-", "\u2212", 1)
}

// shortAmount writes amounts from a million as 1.2M, 1.2B or 1.2T: a tenth of the unit, ties rounding down, capped at 999.9.
func shortAmount(value int64) string {
	var unit int64
	var suffix string
	switch {
	case value >= 1_000_000_000_000:
		unit, suffix = 1_000_000_000_000, "T"
	case value >= 1_000_000_000:
		unit, suffix = 1_000_000_000, "B"
	case value >= 1_000_000:
		unit, suffix = 1_000_000, "M"
	default:
		return amount(value)
	}
	tenth := unit / 10
	tenths := value / tenth
	if value%tenth > tenth/2 {
		tenths++
	}
	tenths = min(tenths, 9999)
	return fmt.Sprintf("%d.%d%s", tenths/10, tenths%10, suffix)
}

func (month spendingMonth) title() string {
	return fmt.Sprintf("%s %d · day %d of %d", month.start.Month(), month.start.Year(), month.today, month.daysInMonth)
}

// allowanceStatus says whether the chat is over its allowance so far, and by how much.
func (month spendingMonth) allowanceStatus() (mark icon, headline, detail string) {
	detail = fmt.Sprintf(" · spent %s of %s so far", amount(month.spent()), amount(month.allowance()))
	switch over := month.overBy(); {
	case over > 0:
		return iconOver, "Over by " + amount(over), detail
	case over < 0:
		return iconUnder, "Under by " + amount(-over), detail
	default:
		return nil, "On the allowance", detail
	}
}

// targetStatus says what daily spend keeps the month within its budget.
func (month spendingMonth) targetStatus() (mark icon, headline, detail string) {
	if target, ok := month.target(); ok {
		days := "days"
		if month.daysAfter() == 1 {
			days = "day"
		}
		return iconTarget, fmt.Sprintf("Target %s/day", amount(target)), fmt.Sprintf(" for the %d %s after today", month.daysAfter(), days)
	}
	switch {
	case month.daysAfter() == 0:
		return nil, "Last day of the month", " · no target"
	case month.left() == 0:
		return iconGone, "Month budget used up", ""
	default:
		return iconGone, "Month budget gone", " · over by " + amount(-month.left())
	}
}
