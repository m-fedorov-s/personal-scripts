package main

import (
	"testing"
	"time"
)

func getIntPointer(val int) *int {
	return &val
}
func getWeekdayPointer(val time.Weekday) *time.Weekday {
	return &val
}

func TestGetNextTime(t *testing.T) {
	type TestCase struct {
		CurrentTime time.Time
		Schedule    ScheduleTime
		Expected    time.Time
	}

	tests := []TestCase{
		TestCase{
			CurrentTime: time.Date(2012, 5, 2, 12, 11, 0, 0, time.UTC),
			Schedule: ScheduleTime{
				Minute: getIntPointer(25),
			},
			Expected: time.Date(2012, 5, 2, 12, 25, 0, 0, time.UTC),
		},
		TestCase{
			CurrentTime: time.Date(2012, 5, 2, 12, 41, 0, 0, time.UTC),
			Schedule: ScheduleTime{
				Minute: getIntPointer(25),
			},
			Expected: time.Date(2012, 5, 2, 13, 25, 0, 0, time.UTC),
		},
		TestCase{
			CurrentTime: time.Date(2012, 5, 2, 12, 11, 0, 0, time.UTC),
			Schedule: ScheduleTime{
				Minute: getIntPointer(25),
				Hour:   getIntPointer(14),
			},
			Expected: time.Date(2012, 5, 2, 14, 25, 0, 0, time.UTC),
		},
		TestCase{
			CurrentTime: time.Date(2012, 5, 2, 12, 11, 0, 0, time.UTC),
			Schedule: ScheduleTime{
				Minute: getIntPointer(25),
				Hour:   getIntPointer(11),
			},
			Expected: time.Date(2012, 5, 3, 11, 25, 0, 0, time.UTC),
		},
		TestCase{
			CurrentTime: time.Date(2012, 5, 2, 12, 11, 0, 0, time.UTC),
			Schedule: ScheduleTime{
				Minute:  getIntPointer(25),
				Hour:    getIntPointer(15),
				Weekday: getWeekdayPointer(time.Thursday),
			},
			Expected: time.Date(2012, 5, 3, 15, 25, 0, 0, time.UTC),
		},
		TestCase{
			CurrentTime: time.Date(2012, 5, 3, 16, 11, 0, 0, time.UTC),
			Schedule: ScheduleTime{
				Minute:  getIntPointer(25),
				Hour:    getIntPointer(15),
				Weekday: getWeekdayPointer(time.Thursday),
			},
			Expected: time.Date(2012, 5, 10, 15, 25, 0, 0, time.UTC),
		},
		TestCase{
			CurrentTime: time.Date(2012, 5, 3, 16, 11, 0, 0, time.UTC),
			Schedule: ScheduleTime{
				Minute: getIntPointer(0),
				Hour:   getIntPointer(15),
				Day:    getIntPointer(2),
			},
			Expected: time.Date(2012, 6, 2, 15, 0, 0, 0, time.UTC),
		},
	}

	for _, test := range tests {
		if test.Schedule.NextTime(test.CurrentTime) != test.Expected {
			t.Errorf("Expected next time %v, got %v", test.Expected, test.Schedule.NextTime(test.CurrentTime))
		}
	}
}

func TestGetNextEvent(t *testing.T) {
	events := []RepeatedEvent[string]{
		RepeatedEvent[string]{
			Time: ScheduleTime{
				Minute: getIntPointer(30),
				Hour:   getIntPointer(18),
			},
			Message: "The first",
		},
		RepeatedEvent[string]{
			Time: ScheduleTime{
				Hour: getIntPointer(11),
				Day:  getIntPointer(7),
			},
			Message: "The second",
		},
	}
	now := time.Date(2077, 2, 4, 19, 32, 0, 0, time.UTC)
	message, timeNext := getNextEventImpl(events, now)
	if *message != "The first" {
		t.Errorf("Expected message `The first`, got: `%v`", *message)
	}
	expectedTime := time.Date(2077, 2, 5, 18, 30, 0, 0, time.UTC)
	if *timeNext != expectedTime {
		t.Errorf("Expected time %v, got %v", expectedTime, *timeNext)
	}
}

func TestShiftTime(t *testing.T) {
	type TestCase struct {
		Period   ScheduleTime
		ShiftBy  time.Duration
		Expected ScheduleTime
	}

	tests := []TestCase{
		TestCase{
			Period: ScheduleTime{
				Minute: getIntPointer(45),
			},
			ShiftBy: time.Minute * 20,
			Expected: ScheduleTime{
				Minute: getIntPointer(25),
			},
		},
		TestCase{
			Period: ScheduleTime{
				Minute: getIntPointer(5),
			},
			ShiftBy: time.Minute * 20,
			Expected: ScheduleTime{
				Minute: getIntPointer(45),
			},
		},
		TestCase{
			Period: ScheduleTime{
				Minute: getIntPointer(5),
				Hour:   getIntPointer(4),
			},
			ShiftBy: time.Minute * 20,
			Expected: ScheduleTime{
				Minute: getIntPointer(45),
				Hour:   getIntPointer(3),
			},
		},
		TestCase{
			Period: ScheduleTime{
				Minute: getIntPointer(5),
				Hour:   getIntPointer(2),
			},
			ShiftBy: time.Minute * 179,
			Expected: ScheduleTime{
				Minute: getIntPointer(6),
				Hour:   getIntPointer(23),
			},
		},
	}

	for _, test := range tests {
		result := test.Period.Before(test.ShiftBy)
		ok := true
		if test.Expected.Minute == nil {
			ok = ok && result.Minute == nil
		} else {
			ok = ok && *result.Minute == *test.Expected.Minute
		}
		if test.Expected.Hour == nil {
			ok = ok && result.Hour == nil
		} else {
			ok = ok && *result.Hour == *test.Expected.Hour
		}
		if test.Expected.Day == nil {
			ok = ok && result.Day == nil
		} else {
			ok = ok && *result.Day == *test.Expected.Day
		}
		if test.Expected.Weekday == nil {
			ok = ok && result.Weekday == nil
		} else {
			ok = ok && *result.Weekday == *test.Expected.Weekday
		}
		if !ok {
			t.Errorf("Expected %v, got %v", test.Expected, result)
		}
	}
}
