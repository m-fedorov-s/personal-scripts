package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log/slog"
	"time"
)

type DayTime struct {
	Hour   int
	Minute int
}

type ChatProfile struct {
	TopicID          int
	DayLimit         int64
	CurrentBalance   int64
	LastRecordTime   time.Time
	MaxRecordID      int64
	ReportTime       *DayTime
	ImmediateReports bool
}

func (cp *ChatProfile) Report() string {
	return fmt.Sprintf("Your current balance: %v", cp.CurrentBalance)
}

func (cp *ChatProfile) DecodeFrom(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(cp)
	return err
}

func (cp *ChatProfile) ConsumeRecord(r Record) {
	if cp.LastRecordTime.Before(r.Date) {
		slog.Debug("Pushing up last date")
		if cp.LastRecordTime.Month() != r.Date.Month() {
			cp.CurrentBalance = cp.DayLimit * int64(r.Date.Day())
		} else if cp.LastRecordTime.Day() != r.Date.Day() {
			daysPassed := int64(r.Date.Day() - cp.LastRecordTime.Day())
			slog.Debug(fmt.Sprintf("Increasing balance by %v", cp.DayLimit*daysPassed))
			cp.CurrentBalance += cp.DayLimit * daysPassed
		}
		cp.LastRecordTime = r.Date
	}
	if cp.LastRecordTime.Month() == r.Date.Month() {
		cp.CurrentBalance -= int64(r.Amount)
	}
}
