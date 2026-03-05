package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"
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
	LastReportSent   time.Time
}

var NotFoundError = fmt.Errorf("Object not found")

func (cp *ChatProfile) Report() string {
	return fmt.Sprintf("Your current balance: %d", cp.CurrentBalance)
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

type ChatIterator interface {
	Next() bool
	Value() (int64, ChatProfile, error)
	Close() error
}

type Storage interface {
	GetChat(chatID int64) (ChatProfile, error)
	SetChat(chatID int64, cp *ChatProfile) error
	SaveRecord(chatID int64, record Record) error
	GetRecords(chatID int64, since time.Time) ([]Record, error)
	Chats() ChatIterator
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
		if err == badger.ErrKeyNotFound {
			return NotFoundError
		}
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

func (s *BadgerStorage) GetRecords(chatID int64, since time.Time) ([]Record, error) {
	var records []Record
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix, err := encode(RecordKey{ChatID: chatID})
		if err != nil {
			return err
		}

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var r Record
				if err := decode(val, &r); err != nil {
					return err
				}
				if r.Date.After(since) || r.Date.Equal(since) {
					records = append(records, r)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return records, err
}

func (s *BadgerStorage) Chats() ChatIterator {
	txn := s.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	return &BadgerChatIterator{
		txn: txn,
		it:  it,
	}
}

type BadgerChatIterator struct {
	txn *badger.Txn
	it  *badger.Iterator
}

func (it *BadgerChatIterator) Next() bool {
	if !it.it.Valid() {
		it.it.Rewind()
	} else {
		it.it.Next()
	}

	// Skip non-int64 keys (records)
	for it.it.Valid() {
		key := it.it.Item().Key()
		var chatID int64
		if err := decode(key, &chatID); err == nil {
			return true
		}
		it.it.Next()
	}
	return false
}

func (it *BadgerChatIterator) Value() (int64, ChatProfile, error) {
	item := it.it.Item()
	var chatID int64
	if err := decode(item.Key(), &chatID); err != nil {
		return 0, ChatProfile{}, err
	}

	var cp ChatProfile
	err := item.Value(func(val []byte) error {
		return cp.DecodeFrom(val)
	})
	return chatID, cp, err
}

func (it *BadgerChatIterator) Close() error {
	it.it.Close()
	it.txn.Discard()
	return nil
}

func decode(data []byte, o any) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(o)
}

func encode(o any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(o)
	return buf.Bytes(), err
}
