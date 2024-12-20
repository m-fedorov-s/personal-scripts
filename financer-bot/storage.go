package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log/slog"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

type Record struct {
	Date   time.Time
	Name   string
	Amount uint32
	ID     int64
}

type RecordKey struct {
	ChatID   int64
	RecordID int64
}

func Encode(o any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(o)
	return buf.Bytes(), err
}

type ChatProfile struct {
	TopicID        int
	DayLimit       int64
	CurrentBalance int64
	LastRecordTime time.Time
	MaxRecordID    int64
}

func (cp *ChatProfile) Describe() string {
	return fmt.Sprintf("Your current balance: %v", cp.CurrentBalance)
}

func (cp *ChatProfile) DecodeFrom(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(cp)
	return err
}

func GetChat(txn *badger.Txn, id int64) (ChatProfile, error) {
	var result ChatProfile
	key, err := Encode(id)
	if err != nil {
		return result, err
	}
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		value, err := Encode(result)
		if err != nil {
			return result, err
		}
		txn.Set(key, value)
		return result, nil
	}
	err = item.Value(func(val []byte) error {
		err := result.DecodeFrom(val)
		return err
	})
	if err != nil {
		slog.Info("Failed to get chat settings", "error", err)
	}
	return result, err
}

func SetChat(txn *badger.Txn, id int64, cp *ChatProfile) error {
	key, err := Encode(id)
	if err != nil {
		return err
	}
	value, err := Encode(cp)
	if err != nil {
		return err
	}
	txn.Set(key, value)
	return nil
}

func (cp *ChatProfile) ConsumeRecord(r Record) {
	if cp.LastRecordTime.Before(r.Date) {
		slog.Debug("Pushing up last date")
		if cp.LastRecordTime.Month() != r.Date.Month() {
			cp.CurrentBalance = 0
		} else if cp.LastRecordTime.Day() != r.Date.Day() {
			daysPassed := int64(r.Date.Day() - cp.LastRecordTime.Day())
			slog.Debug(fmt.Sprintf("Upping up balance by %v"))
			cp.CurrentBalance += cp.DayLimit * daysPassed
		}
		cp.LastRecordTime = r.Date
	}
	if cp.LastRecordTime.Month() == r.Date.Month() {
		cp.CurrentBalance -= int64(r.Amount)
	}
}
