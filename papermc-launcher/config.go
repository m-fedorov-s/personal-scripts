package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

// Represents time in HH:MM format
type DayTime struct {
	hours   int
	minutes int
}

func (d DayTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%02d:%02d", d.hours, d.minutes))
}

func (d *DayTime) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		{
			_, err := fmt.Sscanf(value, "%02d:%02d", &(d.hours), &(d.minutes))
			return err
		}
	default:
		return fmt.Errorf("Daytime should be in HH:MM format")
	}
}

func (d DayTime) Duration() time.Duration {
	return time.Hour*time.Duration(d.hours) + time.Minute*time.Duration(d.minutes)
}

type TimeInterval struct {
	Start DayTime `json:"start"`
	End   DayTime `json:"end"`
}

type Location time.Location

func (l Location) MarshalJSON() ([]byte, error) {
	tmp := time.Location(l)
	return json.Marshal((&tmp).String())
}

func (l *Location) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		tmp, err := time.LoadLocation(value)
		if err != nil {
			return err
		}
		*l = Location(*tmp)
		return nil
	default:
		return fmt.Errorf("invalid location")
	}
}

type Weekday time.Weekday

func (d Weekday) MarshalText() ([]byte, error) {
	return json.Marshal(time.Weekday(d).String())
}

func (d *Weekday) UnmarshalText(b []byte) error {
	switch string(b) {
	case "Sunday":
		*d = Weekday(time.Sunday)
		return nil
	case "Monday":
		*d = Weekday(time.Monday)
		return nil
	case "Tuesday":
		*d = Weekday(time.Tuesday)
		return nil
	case "Wednesday":
		*d = Weekday(time.Wednesday)
		return nil
	case "Thursday":
		*d = Weekday(time.Thursday)
		return nil
	case "Friday":
		*d = Weekday(time.Friday)
		return nil
	case "Saturday":
		*d = Weekday(time.Saturday)
		return nil
	default:
		return fmt.Errorf("invalid weekday")
	}
}

type Schedule struct {
	Timezone     Location                 `json:"timezone"`
	DaysSchedule map[Weekday]TimeInterval `json:"days_schedule"`
}

type PlayerType int

const (
	Java PlayerType = iota
	Bedrock
)

func (pt PlayerType) MarshalJSON() ([]byte, error) {
	var typeStr string
	switch pt {
	case Java:
		typeStr = "Java"
	case Bedrock:
		typeStr = "Bedrock"
	default:
		return nil, fmt.Errorf("Invalid player type: %v", pt)
	}
	return json.Marshal(typeStr)
}

func (pt *PlayerType) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		switch string(value) {
		case "Java":
			*pt = Java
			return nil
		case "Bedrock":
			*pt = Bedrock
			return nil
		default:
			return fmt.Errorf("invalid player type")
		}
	default:
		return fmt.Errorf("Player type should be a string")
	}
}

type Player struct {
	Type     PlayerType `json:"type"`
	Nickname string     `json:"nickname"`
}

// Config represents the configuration with a schedule to restart the process
type Config struct {
	WorkDir        string     `json:"work_dir"`
	WarnBefore     []Duration `json:"warn_before"`
	AccessSchedule Schedule   `json:"schedule"`
	Memory         string     `json:"memory"`
	Players        []Player   `json:"players"`
}
