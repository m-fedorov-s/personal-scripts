package storage

import (
	"os"
	"testing"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

func TestBadgerStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("Failed to open badger: %v", err)
	}
	defer db.Close()

	s, err := NewBadgerStorage(db)
	if err != nil {
		t.Fatalf("NewBadgerStorage() error = %v", err)
	}
	chatID := int64(12345)

	// Test SetChat and GetChat
	profile := ChatProfile{
		TopicID:        1,
		DayLimit:       1000,
		CurrentBalance: 5000,
		LastRecordTime: time.Now().Truncate(time.Second),
	}

	err = s.SetChat(chatID, &profile)
	if err != nil {
		t.Fatalf("SetChat() error = %v", err)
	}

	gotProfile, err := s.GetChat(chatID)
	if err != nil {
		t.Fatalf("GetChat() error = %v", err)
	}

	if gotProfile.TopicID != profile.TopicID || gotProfile.DayLimit != profile.DayLimit || gotProfile.CurrentBalance != profile.CurrentBalance {
		t.Errorf("GetChat() = %+v, want %+v", gotProfile, profile)
	}

	// Test SaveRecord
	record := Record{
		ID:     1,
		Date:   time.Now().Truncate(time.Second),
		Name:   "test record",
		Amount: 100,
	}

	err = s.SaveRecord(chatID, record)
	if err != nil {
		t.Fatalf("SaveRecord() error = %v", err)
	}

	// Verify record exists in DB using the new tuple-encoded key.
	err = db.View(func(txn *badger.Txn) error {
		key := recordKey(chatID, record.ID)
		_, err := txn.Get(key)
		return err
	})
	if err != nil {
		t.Errorf("Record not found in database: %v", err)
	}
}

func TestBadgerStorage_GetNonExistentChat(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "badger-test-empty")
	defer os.RemoveAll(tmpDir)

	db, _ := badger.Open(badger.DefaultOptions(tmpDir).WithLogger(nil))
	defer db.Close()

	s, err := NewBadgerStorage(db)
	if err != nil {
		t.Fatalf("NewBadgerStorage() error = %v", err)
	}
	_, err = s.GetChat(999)
	if err == nil {
		t.Error("GetChat() for non-existent chat should return error")
	}
}
