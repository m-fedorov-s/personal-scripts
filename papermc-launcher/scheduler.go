package main

import (
	"context"
	"fmt"
	"slices"
	"time"
)

type TimeFrame int

const (
	Hour TimeFrame = iota
	Day
	Week
	Month
)

type ScheduleTime struct {
	Minute  *int
	Hour    *int
	Weekday *time.Weekday
	Day     *int
}

func (st ScheduleTime) String() string {
	hours := 0
	if st.Hour != nil {
		hours = *st.Hour
	}
	minutes := 0
	if st.Minute != nil {
		minutes = *st.Minute
	}
	if st.Day != nil {
		return fmt.Sprintf("%vd %vh %vm", *st.Day, hours, minutes)
	}
	if st.Weekday != nil {
		return fmt.Sprintf("%v %vh %vm", *st.Weekday, hours, minutes)
	}
	return fmt.Sprintf("%vh %vm", hours, minutes)
}

func (st ScheduleTime) Frame() TimeFrame {
	if st.Day != nil {
		return Month
	}
	if st.Weekday != nil {
		return Week
	}
	if st.Hour != nil {
		return Day
	}
	return Hour
}

func (st ScheduleTime) Duration() time.Duration {
	minutes := time.Duration(0)
	if st.Minute != nil {
		minutes = time.Duration(*st.Minute)
	}
	hours := time.Duration(0)
	if st.Hour != nil {
		hours = time.Duration(*st.Hour)
	}
	days := time.Duration(0)
	if st.Weekday != nil {
		if *st.Weekday == time.Sunday {
			days = time.Duration(6)
		} else {
			days = time.Duration(int(*st.Weekday) - 1)
		}
	}
	if st.Day != nil {
		days = time.Duration(*st.Day - 1)
	}
	return time.Minute*minutes + time.Hour*hours + time.Hour*24*days
}

func (st ScheduleTime) NextTime(from time.Time) time.Time {
	frame := st.Frame()
	var periodStart, periodEnd time.Time
	switch frame {
	case Month:
		periodStart = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
		periodEnd = periodStart.AddDate(0, 1, 0)
	case Week:
		periodStart = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		if wd := periodStart.Weekday(); wd == time.Sunday {
			periodStart = periodStart.AddDate(0, 0, -6)
		} else {
			periodStart = periodStart.AddDate(0, 0, -int(wd)+1)
		}
		periodEnd = periodStart.Add(time.Hour * 24 * 7)
	case Day:
		periodStart = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		periodEnd = periodStart.AddDate(0, 0, 1)
	case Hour:
		periodStart = time.Date(from.Year(), from.Month(), from.Day(), from.Hour(), 0, 0, 0, from.Location())
		periodEnd = periodStart.Add(time.Hour)
	}
	if periodStart.Add(st.Duration()).After(from) {
		return periodStart.Add(st.Duration())
	}
	return periodEnd.Add(st.Duration())
}

func (st ScheduleTime) Before(shift time.Duration) ScheduleTime {
	minutes := 0
	if st.Minute != nil {
		minutes = *st.Minute
	}
	hoursOverflow := 0
	if minutes < int(shift.Minutes())%60 {
		hoursOverflow = 1
	}
	minutes = ((minutes-int(shift.Minutes()))%60 + 60) % 60
	if st.Frame() == Hour {
		return ScheduleTime{
			Minute: &minutes,
		}
	}
	hours := 0
	if st.Hour != nil {
		hours = *st.Hour
	}
	daysOverflow := 0
	if hours < int(shift.Hours())+hoursOverflow {
		daysOverflow = 1
	}
	hours = ((hours-int(shift.Hours())-hoursOverflow)%24 + 24) % 24
	if st.Frame() == Day {
		return ScheduleTime{
			Minute: &minutes,
			Hour:   &hours,
		}
	}
	if st.Weekday != nil {
		weekday := *st.Weekday
		weekday = time.Weekday(((int(weekday)-int(shift.Hours()/24)-daysOverflow)%7 + 7) % 7)
		return ScheduleTime{
			Minute:  &minutes,
			Hour:    &hours,
			Weekday: &weekday,
		}
	}
	days := *st.Day
	days = ((days-int(shift.Hours()/24)-daysOverflow)%30 + 30) % 30
	return ScheduleTime{
		Minute: &minutes,
		Hour:   &hours,
		Day:    &days,
	}
}

type RepeatedEvent[M any] struct {
	Time    ScheduleTime
	Message M
}

func (re RepeatedEvent[M]) NextTime(from time.Time) time.Time {
	return re.Time.NextTime(from)
}

type Scheduler[M any] struct {
	Output    chan<- M
	callbacks []func()
	Schedule  []RepeatedEvent[M]
	Timezone  time.Location
}

func (s *Scheduler[M]) OnCancel(f func()) {
	s.callbacks = append(s.callbacks, f)
}

func (s *Scheduler[M]) Start(ctx context.Context) {
	fmt.Println("Starting scheduler")
	defer func() {
		for _, c := range s.callbacks {
			c()
		}
	}()
	timer := time.NewTimer(time.Hour)
	for {
		message, nextTime := getNextEvent(s.Schedule, s.Timezone)
		if nextTime != nil {
			timer.Reset(time.Until(*nextTime))
		} else {
			timer.Reset(time.Hour)
		}
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if message != nil {
				s.Output <- *message
			}
		}
	}
}

func getNextEvent[M any](s []RepeatedEvent[M], tz time.Location) (*M, *time.Time) {
	return getNextEventImpl(s, time.Now().In(&tz))
}

func getNextEventImpl[M any](s []RepeatedEvent[M], now time.Time) (*M, *time.Time) {
	if len(s) == 0 {
		return nil, nil
	}
	nextEvent := slices.MinFunc(s, func(a, b RepeatedEvent[M]) int {
		return a.NextTime(now).Compare(b.NextTime(now))
	})
	nextTime := nextEvent.NextTime(now)
	return &nextEvent.Message, &nextTime
}
