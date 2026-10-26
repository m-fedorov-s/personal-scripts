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

func TestBadgerStorage_LastReportID(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	s, err := NewBadgerStorage(db)
	if err != nil {
		t.Fatalf("NewBadgerStorage() error = %v", err)
	}

	// nil means no previous report and must stay nil.
	if err := s.SetChat(1, &ChatProfile{}); err != nil {
		t.Fatalf("SetChat() error = %v", err)
	}
	got, err := s.GetChat(1)
	if err != nil {
		t.Fatalf("GetChat() error = %v", err)
	}
	if got.LastReportID != nil {
		t.Errorf("GetChat() LastReportID = %d, want nil", *got.LastReportID)
	}

	id := 4217
	if err := s.SetChat(2, &ChatProfile{LastReportID: &id}); err != nil {
		t.Fatalf("SetChat() error = %v", err)
	}
	got, err = s.GetChat(2)
	if err != nil {
		t.Fatalf("GetChat() error = %v", err)
	}
	if got.LastReportID == nil || *got.LastReportID != id {
		t.Errorf("GetChat() LastReportID = %v, want %d", got.LastReportID, id)
	}
}

func TestBadgerStorage_LastReportBuiltAt(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	s, err := NewBadgerStorage(db)
	if err != nil {
		t.Fatalf("NewBadgerStorage() error = %v", err)
	}

	if err := s.SetChat(1, &ChatProfile{}); err != nil {
		t.Fatalf("SetChat() error = %v", err)
	}
	got, err := s.GetChat(1)
	if err != nil {
		t.Fatalf("GetChat() error = %v", err)
	}
	if got.LastReportBuiltAt != nil {
		t.Errorf("GetChat() LastReportBuiltAt = %v, want nil", *got.LastReportBuiltAt)
	}

	builtAt := time.Now()
	if err := s.SetChat(2, &ChatProfile{LastReportBuiltAt: &builtAt}); err != nil {
		t.Fatalf("SetChat() error = %v", err)
	}
	got, err = s.GetChat(2)
	if err != nil {
		t.Fatalf("GetChat() error = %v", err)
	}
	// Equal, not ==: gob drops the monotonic reading and may change the Location.
	if got.LastReportBuiltAt == nil || !got.LastReportBuiltAt.Equal(builtAt) {
		t.Errorf("GetChat() LastReportBuiltAt = %v, want %v", got.LastReportBuiltAt, builtAt)
	}
}

// A report time of 00:00 is all zero fields, which gob omits; the pointer
// must still come back non-nil or the chat silently stops getting reports.
func TestBadgerStorage_MidnightReportTime(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	s, err := NewBadgerStorage(db)
	if err != nil {
		t.Fatalf("NewBadgerStorage() error = %v", err)
	}

	if err := s.SetChat(1, &ChatProfile{ReportTime: &DayTime{Hour: 0, Minute: 0}}); err != nil {
		t.Fatalf("SetChat() error = %v", err)
	}
	got, err := s.GetChat(1)
	if err != nil {
		t.Fatalf("GetChat() error = %v", err)
	}
	if got.ReportTime == nil {
		t.Fatal("GetChat() ReportTime = nil, want 00:00")
	}
	if got.ReportTime.Hour != 0 || got.ReportTime.Minute != 0 {
		t.Errorf("GetChat() ReportTime = %+v, want 00:00", *got.ReportTime)
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
