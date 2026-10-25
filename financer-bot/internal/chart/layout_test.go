package chart

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/plot/vg/vgimg"

	"financer/internal/storage"
)

var scenarios = []scenario{
	{name: "early under", now: october(3), limit: 3300, daily: []int64{800, 3100, 0}},
	{name: "mid-month under", now: october(15), limit: 3300, daily: midMonthUnder},
	{name: "late over with budget left", now: october(27), limit: 3300, daily: lateOver},
	{name: "budget blown", now: october(24), limit: 3300, daily: budgetBlown},
	{name: "day 1 with no records", now: october(1), limit: 3300},
	{name: "last day under", now: october(31), limit: 3300, daily: repeat(3000, 31)},
	{name: "last day over", now: october(31), limit: 3300, daily: append(repeat(3000, 30), 20000)},
	{name: "today's records alone use up the budget", now: october(5), limit: 3300, daily: []int64{0, 0, 0, 0, 110000}},
	{name: "spent equals budget", now: october(20), limit: 3300, daily: append(repeat(5000, 19), 7300)},
	{name: "spent equals allowance", now: october(10), limit: 3300, daily: repeat(3300, 10)},
	{name: "today over the limit with budget left", now: october(12), limit: 3300, daily: append(repeat(2000, 11), 6000)},
	{name: "heavily under late", now: october(28), limit: 3300, daily: repeat(300, 28)},
	{name: "one day above three limits", now: october(15), limit: 3300, daily: append(repeat(2500, 14), 21500)},
	{name: "two adjacent days above three limits", now: october(15), limit: 3300, daily: append(repeat(1000, 13), 21500, 18000)},
	{name: "three adjacent days above three limits", now: october(15), limit: 3300, daily: append(repeat(1000, 12), 21500, 18000, 25000)},
	{name: "three days above three limits ending the month", now: october(31), limit: 3300, daily: append(repeat(1000, 28), 21500, 18000, 25000)},
	{name: "four days above three limits", now: october(15), limit: 3300, daily: append(repeat(1000, 11), 21500, 18000, 25000, 30000)},
	{name: "seven-digit amounts", now: october(14), limit: 400000,
		daily: []int64{350000, 1200000, 90000, 2500000, 410000, 0, 380000, 1000000, 220000, 300000, 450000, 390000, 1100000, 70000}},
	{name: "28-day month, 4 calendar rows", now: time.Date(2027, 2, 14, 20, 0, 0, 0, time.UTC), limit: 3300, daily: repeat(2900, 14)},
	{name: "29-day month, last day", now: time.Date(2028, 2, 29, 20, 0, 0, 0, time.UTC), limit: 3300, daily: repeat(3400, 29)},
	{name: "30-day month, 6 calendar rows", now: time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC), limit: 3300, daily: repeat(3500, 20)},
	{name: "31-day month, 6 calendar rows", now: time.Date(2026, 8, 12, 20, 0, 0, 0, time.UTC), limit: 3300, daily: repeat(3100, 12)},
	{name: "records from last month and after now", now: october(15), limit: 3300, daily: midMonthUnder, extra: []storage.Record{
		{Date: time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC), Amount: 50000},
		{Date: october(15).Add(time.Second), Amount: 50000},
	}},
	{name: "a ten-digit day", now: october(10), limit: 3300, daily: append(repeat(1000, 9), 4294967295)},
	{name: "limit far below spending", now: october(15), limit: 300, daily: midMonthUnder},
	{name: "no limit", now: october(15), limit: 0, daily: midMonthUnder},
	{name: "negative limit", now: october(15), limit: -100, daily: midMonthUnder},
	{name: "no limit and nothing spent", now: october(15), limit: 0},
	{name: "no limit with three days above three median days", now: october(15), limit: 0, daily: append(repeat(1000, 12), 21500, 18000, 25000)},
}

// designs are the renderers under test, by the name used for written images.
var designs = []struct {
	name       string
	draw       func(spendingMonth) *picture
	needsLimit bool
}{
	{"bars", drawBars, false},
	{"running-total", drawRunningTotal, true},
	{"over-under", drawOverUnder, true},
	{"calendar", drawCalendar, false},
}

func TestEveryDesignKeepsTheLayoutRulesInEveryScenario(t *testing.T) {
	out := os.Getenv("CHART_PNG_DIR")
	for _, design := range designs {
		for _, sc := range scenarios {
			if design.needsLimit && sc.limit <= 0 {
				continue
			}
			t.Run(design.name+"/"+sc.name, func(t *testing.T) {
				pic := design.draw(sc.month())
				assert.Empty(t, problems(pic))
				if out != "" {
					writePNG(t, pic, filepath.Join(out, design.name+"-"+strings.ReplaceAll(sc.name, " ", "-")+".png"))
				}
			})
		}
	}
}

// Reports render in parallel workers, so a render must not depend on another one in flight.
// Different months render at once, so state shared between renders shows as a render that
// differs from its serial twin.
func TestConcurrentRendersMatchSerialRenders(t *testing.T) {
	type job struct {
		name  string
		draw  func(spendingMonth) *picture
		month spendingMonth
	}
	var jobs []job
	for _, design := range designs {
		for _, sc := range scenarios {
			if design.needsLimit && sc.limit <= 0 {
				continue
			}
			jobs = append(jobs, job{design.name + "/" + sc.name, design.draw, sc.month()})
		}
	}
	encode := func(work job) []byte {
		var buf bytes.Buffer
		_, err := vgimg.PngCanvas{Canvas: work.draw(work.month).canvas}.WriteTo(&buf)
		assert.NoError(t, err)
		return buf.Bytes()
	}
	want := make([][]byte, len(jobs))
	for index, work := range jobs {
		want[index] = encode(work)
	}
	got := make([][]byte, len(jobs))
	var wg sync.WaitGroup
	for index, work := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[index] = encode(work)
		}()
	}
	wg.Wait()
	for index, work := range jobs {
		assert.True(t, bytes.Equal(want[index], got[index]), "%s differs from its serial render", work.name)
	}
}

func texts(pic *picture) []string {
	var all []string
	for _, placed := range pic.texts {
		all = append(all, placed.text)
	}
	return all
}

func countFill(pic *picture, fill color.NRGBA) int {
	count := 0
	for _, drawn := range pic.areas {
		if drawn.fill == fill {
			count++
		}
	}
	return count
}

// writePNG saves a render for a person to look at; set CHART_PNG_DIR to get them.
func writePNG(t *testing.T, pic *picture, path string) {
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()
	_, err = vgimg.PngCanvas{Canvas: pic.canvas}.WriteTo(file)
	require.NoError(t, err)
}
