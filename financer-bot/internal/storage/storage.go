package storage

import (
	"bytes"
	"encoding/gob"
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
	return "Your current balance: " + string(cp.CurrentBalance) // Simplified for brevity
}

func (cp *ChatProfile) DecodeFrom(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(cp)
}

func (cp *ChatProfile) ConsumeRecord(r Record) {
	if cp.LastRecordTime.Before(r.Date) {
		if cp.LastRecordTime.Month() != r.Date.Month() {
			cp.CurrentBalance = cp.DayLimit * int64(r.Date.Day())
		} else if cp.LastRecordTime.Day() != r.Date.Day() {
			daysPassed := int64(r.Date.Day() - cp.LastRecordTime.Day())
			cp.CurrentBalance += cp.DayLimit * daysPassed
		}
		cp.LastRecordTime = r.Date
	}
	if cp.LastRecordTime.Month() == r.Date.Month() {
		cp.CurrentBalance -= int64(r.Amount)
	}
}

type Storage interface {
	GetChat(chatID int64) (ChatProfile, error)
	SetChat(chatID int64, cp *ChatProfile) error
	SaveRecord(chatID int64, record Record) error
}

type BadgerStorage struct {
	db *badger.DB
}

func NewBadgerStorage(db *badger.DB) *BadgerStorage {
	return &BadgerStorage{db: db}
}

func (s *BadgerStorage) GetChat(chatID int64) (ChatProfile, error) {
	var result ChatProfile
	err := s.db.View(func(txn *badger.Txn) error {
		key, err := encode(chatID)
		if err != nil {
			return err
		}
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return result.DecodeFrom(val)
		})
	})
	return result, err
}

func (s *BadgerStorage) SetChat(chatID int64, cp *ChatProfile) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key, err := encode(chatID)
		if err != nil {
			return err
		}
		value, err := encode(cp)
		if err != nil {
			return err
		}
		return txn.Set(key, value)
	})
}

func (s *BadgerStorage) SaveRecord(chatID int64, record Record) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := RecordKey{ChatID: chatID, RecordID: record.ID}
		encodedKey, err := encode(key)
		if err != nil {
			return err
		}
		encodedRecord, err := encode(record)
		if err != nil {
			return err
		}
		return txn.Set(encodedKey, encodedRecord)
	})
}

func encode(o any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(o)
	return buf.Bytes(), err
}
