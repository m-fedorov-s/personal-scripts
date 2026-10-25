package chart

import (
	"time"

	"financer/internal/storage"
)

// scenario is a month of spending: one record at 09:00 UTC on each day with a non-zero amount,
// plus any extra records.
type scenario struct {
	name  string
	now   time.Time
	limit int64
	daily []int64 // amounts for days 1, 2, ...
	extra []storage.Record
}

func (sc scenario) records() []storage.Record {
	records := append([]storage.Record(nil), sc.extra...)
	for index, value := range sc.daily {
		if value > 0 {
			date := time.Date(sc.now.Year(), sc.now.Month(), index+1, 9, 0, 0, 0, time.UTC)
			records = append(records, storage.Record{Date: date, Amount: uint32(value)})
		}
	}
	return records
}

func (sc scenario) month() spendingMonth { return newMonth(sc.records(), sc.limit, sc.now) }

func october(day int) time.Time { return time.Date(2026, 10, day, 20, 0, 0, 0, time.UTC) }

// repeat returns count copies of value.
func repeat(value int64, count int) []int64 {
	values := make([]int64, count)
	for index := range values {
		values[index] = value
	}
	return values
}

var (
	midMonthUnder = []int64{2100, 3400, 1200, 0, 4600, 2800, 1900, 2500, 6200, 1500, 900, 3100, 2700, 3800, 1100}
	lateOver      = []int64{2400, 3100, 900, 3600, 2800, 4100, 1700, 2600, 3900, 1200, 2300, 6800, 3000, 2100, 800, 3300,
		2900, 1500, 21500, 0, 5400, 2700, 3600, 1900, 8300, 2200, 1400}
	budgetBlown = []int64{3100, 2400, 4200, 1800, 3600, 25000, 2900, 1200, 3800, 4500, 2100, 0, 3300, 5200, 2600, 3900,
		18000, 1600, 4100, 2800, 5100, 3500, 2700, 900}
)
