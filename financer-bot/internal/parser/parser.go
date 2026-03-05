package parser

import (
	"financer/internal/storage"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ParseMessage(text string) []storage.Record {
	dateRE := regexp.MustCompile("^\\d{1,2}\\.\\d{1,2}$")
	date := time.Now()
	lines := strings.Split(text, "\n")
	result := make([]storage.Record, 0, len(lines))
	for _, line := range lines {
		if dateRE.Match([]byte(line)) {
			ints := strings.Split(line, ".")
			day, _ := strconv.ParseInt(ints[0], 10, 64)
			month, _ := strconv.ParseInt(ints[1], 10, 64)
			date = time.Date(date.Year(), time.Month(month), int(day), 0, 0, 0, 0, time.Now().UTC().Location())
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}
		nameBits := make([]string, 0, len(parts)-1)
		var amount int32 = 0
		for _, part := range parts {
			parsed, err := strconv.ParseInt(part, 10, 32)
			if err != nil {
				nameBits = append(nameBits, part)
			} else {
				amount = int32(parsed)
			}
		}
		if amount > 0 {
			result = append(result, storage.Record{
				Date:   date,
				Name:   strings.Join(nameBits, " "),
				Amount: uint32(amount),
			})
		}
	}
	return result
}
