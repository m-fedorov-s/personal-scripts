package chart

import (
	"time"

	"financer/internal/storage"
)

// spendingMonth is a chat's spending in the current UTC month, as of now. Today counts as a day passed.
type spendingMonth struct {
	start       time.Time
	daysInMonth int
	today       int
	limit       int64   // the chat's daily limit; 0 or less means none is set
	daily       []int64 // daily[day-1] is the amount recorded on that day; 0 after today
}

func newMonth(records []storage.Record, limit int64, now time.Time) spendingMonth {
	now = now.UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	month := spendingMonth{
		start:       start,
		daysInMonth: start.AddDate(0, 1, -1).Day(),
		today:       now.Day(),
		limit:       limit,
	}
	month.daily = make([]int64, month.daysInMonth)
	for _, record := range records {
		date := record.Date.UTC()
		if date.Before(start) || date.After(now) {
			continue
		}
		month.daily[date.Day()-1] += int64(record.Amount)
	}
	return month
}

func (month spendingMonth) hasLimit() bool { return month.limit > 0 }

func (month spendingMonth) spent() int64 {
	var total int64
	for _, dayAmount := range month.daily {
		total += dayAmount
	}
	return total
}

func (month spendingMonth) allowance() int64 { return month.limit * int64(month.today) }

// overBy is positive when over the allowance so far and negative when under it.
func (month spendingMonth) overBy() int64 { return month.spent() - month.allowance() }

func (month spendingMonth) budget() int64 { return month.limit * int64(month.daysInMonth) }

func (month spendingMonth) left() int64 { return month.budget() - month.spent() }

func (month spendingMonth) budgetGone() bool { return month.left() <= 0 }

func (month spendingMonth) daysAfter() int { return month.daysInMonth - month.today }

// mondayOffset is the number of weekdays before day 1 in a Monday-first week.
func (month spendingMonth) mondayOffset() int { return (int(month.start.Weekday()) + 6) % 7 }

// calendarRows is the number of Monday-first weeks the month spans.
func (month spendingMonth) calendarRows() int {
	return (month.mondayOffset() + month.daysInMonth + 6) / 7
}

// pace is the month's total if every day matched the average so far.
func (month spendingMonth) pace() int64 {
	return month.spent() * int64(month.daysInMonth) / int64(month.today)
}

// target is the daily spend over the days after today that brings the month's mean to the limit,
// rounded down so that following it never exceeds the budget. There is none on the last day or
// once the budget is gone.
func (month spendingMonth) target() (int64, bool) {
	if month.budgetGone() || month.daysAfter() == 0 {
		return 0, false
	}
	return month.left() / int64(month.daysAfter()), true
}

// cumulative returns the total spent by the end of each day, from day 0 (nothing) to today.
func (month spendingMonth) cumulative() []int64 {
	totals := make([]int64, month.today+1)
	for day := 1; day <= month.today; day++ {
		totals[day] = totals[day-1] + month.daily[day-1]
	}
	return totals
}

// overUnder is the amount spent beyond the allowance at the end of each day from 0 to today:
// positive when over.
func (month spendingMonth) overUnder() []int64 {
	series := month.cumulative()
	for day := range series {
		series[day] -= month.limit * int64(day)
	}
	return series
}
