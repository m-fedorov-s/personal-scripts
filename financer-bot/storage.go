package main

import (
	"bytes"
	"encoding/gob"
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
