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

// NewBadgerStorage creates a BadgerStorage and runs any pending schema migrations.
// It returns an error if migrations fail, which should be treated as fatal.
func NewBadgerStorage(db *badger.DB) (*BadgerStorage, error) {
	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("storage migrations: %w", err)
	}
	return &BadgerStorage{db: db}, nil
}

func (s *BadgerStorage) GetChat(chatID int64) (ChatProfile, error) {
	var result ChatProfile
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(chatProfileKey(chatID))
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
		value, err := encode(cp)
		if err != nil {
			return err
		}
		return txn.Set(chatProfileKey(chatID), value)
	})
}

func (s *BadgerStorage) SaveRecord(chatID int64, record Record) error {
	return s.db.Update(func(txn *badger.Txn) error {
		encodedRecord, err := encode(record)
		if err != nil {
			return err
		}
		return txn.Set(recordKey(chatID, record.ID), encodedRecord)
	})
}

// GetRecords returns all records for chatID whose Date is >= since.
// It uses recordPrefix(chatID) for efficient prefix iteration — this works
// correctly because the FDB tuple layer guarantees that recordKey(chatID, *)
// always starts with recordPrefix(chatID).
func (s *BadgerStorage) GetRecords(chatID int64, since time.Time) ([]Record, error) {
	var records []Record
	prefix := recordPrefix(chatID)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var r Record
				if err := decode(val, &r); err != nil {
					return err
				}
				if !r.Date.Before(since) {
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

	// Skip keys that are not chat profile keys (records, metadata).
	for it.it.Valid() {
		key := it.it.Item().Key()
		// A chat profile key unpacks to a 2-element tuple {nsChat, chatID}.
		// We detect it by attempting to decode and checking the namespace.
		if isChatProfileKey(key) {
			return true
		}
		it.it.Next()
	}
	return false
}

func (it *BadgerChatIterator) Value() (int64, ChatProfile, error) {
	item := it.it.Item()
	chatID, err := decodeChatProfileKey(item.Key())
	if err != nil {
		return 0, ChatProfile{}, err
	}

	var cp ChatProfile
	err = item.Value(func(val []byte) error {
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
