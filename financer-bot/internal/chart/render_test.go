package chart

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"financer/internal/storage"
)

func TestRenderDrawsTheStoredDesign(t *testing.T) {
	month := scenario{now: october(15), limit: 3300, daily: midMonthUnder}.month()
	tests := []struct {
		design storage.ChartDesign
		want   *picture
	}{
		{storage.ChartBars, drawBars(month)},
		{storage.ChartRunningTotal, drawRunningTotal(month)},
		{storage.ChartOverUnder, drawOverUnder(month)},
		{storage.ChartCalendar, drawCalendar(month)},
	}
	for _, tt := range tests {
		assert.Equal(t, texts(tt.want), texts(render(month, tt.design)), "design %d", tt.design)
	}
}

func TestRenderDrawsBarsForADesignThatNeedsAMissingLimit(t *testing.T) {
	for _, limit := range []int64{0, -100} {
		month := scenario{now: october(15), limit: limit, daily: midMonthUnder}.month()
		bars := texts(drawBars(month))
		assert.Equal(t, bars, texts(render(month, storage.ChartRunningTotal)), "running total, limit %d", limit)
		assert.Equal(t, bars, texts(render(month, storage.ChartOverUnder)), "over/under, limit %d", limit)
		assert.Equal(t, texts(drawCalendar(month)), texts(render(month, storage.ChartCalendar)), "calendar, limit %d", limit)
	}
}

func TestRenderDrawsBarsForAnUnknownDesign(t *testing.T) {
	month := scenario{now: october(15), limit: 3300, daily: midMonthUnder}.month()
	assert.Equal(t, texts(drawBars(month)), texts(render(month, storage.ChartDesign(9))))
}
