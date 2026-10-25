package chart

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAmountGroupsThousandsAndUsesATrueMinus(t *testing.T) {
	for value, want := range map[int64]string{0: "0", 999: "999", 1000: "1,000", 102300: "102,300", -1234567: "−1,234,567"} {
		assert.Equal(t, want, amount(value))
	}
}

func TestShortAmountRoundsToATenthOfTheUnitWithTiesDownAndCapsAt999Point9(t *testing.T) {
	for value, want := range map[int64]string{999999: "999,999", 1000000: "1.0M", 1250000: "1.2M", 2500000000: "2.5B",
		999999000: "999.9M", 1290000: "1.3M", 1234000999: "1.2B", 999999999999: "999.9B",
		1050000: "1.0M", 1150000: "1.1M", 1150001: "1.2M", 1000000000000: "1.0T", 1250000000000: "1.2T",
		999999999999999: "999.9T", math.MaxInt64: "999.9T"} {
		assert.Equal(t, want, shortAmount(value))
	}
}

func TestHeaderLines(t *testing.T) {
	type status struct{ headline, detail string }
	tests := []struct {
		name                          string
		sc                            scenario
		title                         string
		allowanceStatus, targetStatus status
	}{
		{"under with a target", scenario{now: october(15), limit: 3300, daily: midMonthUnder}, "October 2026 · day 15 of 31",
			status{"Under by 11,700", " · spent 37,800 of 49,500 so far"}, status{"Target 4,031/day", " for the 16 days after today"}},
		{"one day after today", scenario{now: october(30), limit: 3300, daily: repeat(3300, 30)}, "October 2026 · day 30 of 31",
			status{"On the allowance", " · spent 99,000 of 99,000 so far"}, status{"Target 3,300/day", " for the 1 day after today"}},
		{"budget gone", scenario{now: october(20), limit: 3300, daily: append(repeat(5000, 19), 8300)}, "October 2026 · day 20 of 31",
			status{"Over by 37,300", " · spent 103,300 of 66,000 so far"}, status{"Month budget gone", " · over by 1,000"}},
		{"budget used up", scenario{now: october(20), limit: 3300, daily: append(repeat(5000, 19), 7300)}, "October 2026 · day 20 of 31",
			status{"Over by 36,300", " · spent 102,300 of 66,000 so far"}, status{"Month budget used up", ""}},
		{"last day over the budget", scenario{now: october(31), limit: 3300, daily: append(repeat(3000, 30), 20000)}, "October 2026 · day 31 of 31",
			status{"Over by 7,700", " · spent 110,000 of 102,300 so far"}, status{"Last day of the month", " · no target"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			month := tt.sc.month()
			assert.Equal(t, tt.title, month.title())
			_, headline, detail := month.allowanceStatus()
			assert.Equal(t, tt.allowanceStatus, status{headline, detail})
			_, headline, detail = month.targetStatus()
			assert.Equal(t, tt.targetStatus, status{headline, detail})
		})
	}
}
