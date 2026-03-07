package storage

import (
	"os"
	"testing"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openTestDB opens a fresh BadgerDB in a temp directory and returns the db
// and a cleanup function.
func openTestDB(t *testing.T) (*badger.DB, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "badger_migration_*")
	require.NoError(t, err)

	opts := badger.DefaultOptions(tmpDir).WithLogger(nil)
	db, err := badger.Open(opts)
	require.NoError(t, err)

	return db, func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
}

// writeLegacyRecord writes a record using the old gob-encoded RecordKey format
// (v0 schema) directly into Badger, bypassing the new storage layer.
func writeLegacyRecord(t *testing.T, db *badger.DB, chatID int64, r Record) {
	t.Helper()
	key, err := encode(RecordKey{ChatID: chatID, RecordID: r.ID})
	require.NoError(t, err)
	val, err := encode(r)
	require.NoError(t, err)
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
	require.NoError(t, err)
}

// writeLegacyChatProfile writes a chat profile using the old gob-encoded int64
// key format (v0 schema) directly into Badger.
func writeLegacyChatProfile(t *testing.T, db *badger.DB, chatID int64, cp ChatProfile) {
	t.Helper()
	key, err := encode(chatID)
	require.NoError(t, err)
	val, err := encode(cp)
	require.NoError(t, err)
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
	require.NoError(t, err)
}

// TestMigration_FreshDB verifies that a fresh database (no metadata key) is
// treated as v0 and migrated to v1 with the metadata key written.
func TestMigration_FreshDB(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	err := RunMigrations(db)
	require.NoError(t, err)

	meta, err := readSchemaMeta(db)
	require.NoError(t, err)
	assert.Equal(t, currentSchemaVersion, meta.Version)
}

// TestMigration_AlreadyAtCurrentVersion verifies that RunMigrations is a no-op
// when the database is already at the current schema version.
func TestMigration_AlreadyAtCurrentVersion(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	// Write metadata at current version.
	err := writeSchemaMeta(db, SchemaMetadata{Version: currentSchemaVersion})
	require.NoError(t, err)

	// Running migrations again should succeed without error.
	err = RunMigrations(db)
	require.NoError(t, err)

	meta, err := readSchemaMeta(db)
	require.NoError(t, err)
	assert.Equal(t, currentSchemaVersion, meta.Version)
}

// TestMigration_LegacyRecordsAreReadable verifies the core migration scenario:
// records written in the v0 gob-encoded key format are readable via GetRecords
// after migration.
func TestMigration_LegacyRecordsAreReadable(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	chatID := int64(12345)
	now := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)

	legacyRecords := []Record{
		{ID: 1, Amount: 100, Date: now.Add(-1 * time.Hour), Name: "coffee"},
		{ID: 2, Amount: 200, Date: now.Add(-2 * time.Hour), Name: "lunch"},
		{ID: 3, Amount: 50, Date: now.Add(-3 * time.Hour), Name: "snack"},
	}

	// Write records in legacy v0 format.
	for _, r := range legacyRecords {
		writeLegacyRecord(t, db, chatID, r)
	}

	// Run migration.
	err := RunMigrations(db)
	require.NoError(t, err)

	// Now open storage with the new layer and verify records are readable.
	s, err := NewBadgerStorage(db)
	require.NoError(t, err)

	retrieved, err := s.GetRecords(chatID, now.Add(-5*time.Hour))
	require.NoError(t, err)
	assert.Len(t, retrieved, 3, "all 3 legacy records should be readable after migration")

	// Verify amounts are preserved.
	totalAmount := uint32(0)
	for _, r := range retrieved {
		totalAmount += r.Amount
	}
	assert.Equal(t, uint32(350), totalAmount)
}

// TestMigration_LegacyChatProfileIsReadable verifies that chat profiles written
// in the v0 gob-encoded int64 key format are readable after migration.
func TestMigration_LegacyChatProfileIsReadable(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	chatID := int64(42)
	original := ChatProfile{
		TopicID:        1,
		DayLimit:       1000,
		CurrentBalance: 5000,
	}

	writeLegacyChatProfile(t, db, chatID, original)

	err := RunMigrations(db)
	require.NoError(t, err)

	s, err := NewBadgerStorage(db)
	require.NoError(t, err)

	got, err := s.GetChat(chatID)
	require.NoError(t, err)
	assert.Equal(t, original.DayLimit, got.DayLimit)
	assert.Equal(t, original.CurrentBalance, got.CurrentBalance)
}

// TestMigration_MixedLegacyData verifies that a database with both legacy chat
// profiles and legacy records is fully migrated without data loss.
func TestMigration_MixedLegacyData(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	chatID1 := int64(100)
	chatID2 := int64(200)
	now := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)

	writeLegacyChatProfile(t, db, chatID1, ChatProfile{DayLimit: 500, CurrentBalance: 1000})
	writeLegacyChatProfile(t, db, chatID2, ChatProfile{DayLimit: 750, CurrentBalance: 2000})
	writeLegacyRecord(t, db, chatID1, Record{ID: 1, Amount: 100, Date: now})
	writeLegacyRecord(t, db, chatID1, Record{ID: 2, Amount: 200, Date: now.Add(-time.Hour)})
	writeLegacyRecord(t, db, chatID2, Record{ID: 1, Amount: 300, Date: now})

	err := RunMigrations(db)
	require.NoError(t, err)

	s, err := NewBadgerStorage(db)
	require.NoError(t, err)

	// Verify chat profiles.
	cp1, err := s.GetChat(chatID1)
	require.NoError(t, err)
	assert.Equal(t, int64(500), cp1.DayLimit)

	cp2, err := s.GetChat(chatID2)
	require.NoError(t, err)
	assert.Equal(t, int64(750), cp2.DayLimit)

	// Verify records per chat.
	recs1, err := s.GetRecords(chatID1, now.Add(-5*time.Hour))
	require.NoError(t, err)
	assert.Len(t, recs1, 2)

	recs2, err := s.GetRecords(chatID2, now.Add(-5*time.Hour))
	require.NoError(t, err)
	assert.Len(t, recs2, 1)
}

// TestMigration_EmptyDB verifies that an empty database migrates cleanly.
func TestMigration_EmptyDB(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	err := RunMigrations(db)
	require.NoError(t, err)

	s, err := NewBadgerStorage(db)
	require.NoError(t, err)

	// No records should exist.
	recs, err := s.GetRecords(999, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Empty(t, recs)
}

// TestGetRecords_NewFormat verifies that records written with the new storage
// layer are correctly retrieved (regression test for the original prefix bug).
func TestGetRecords_NewFormat(t *testing.T) {
	chatID := int64(12345)
	now := time.Now().UTC()

	tests := []struct {
		name    string
		records []Record
		since   time.Time
		wantLen int
	}{
		{
			name: "two records retrieved",
			records: []Record{
				{ID: 1, Amount: 100, Date: now.Add(-1 * time.Hour)},
				{ID: 2, Amount: 200, Date: now.Add(-2 * time.Hour)},
			},
			since:   now.Add(-5 * time.Hour),
			wantLen: 2,
		},
		{
			name: "since filter excludes old records",
			records: []Record{
				{ID: 3, Amount: 50, Date: now.Add(-48 * time.Hour)},
			},
			since:   now.Add(-24 * time.Hour),
			wantLen: 0,
		},
		{
			name: "record exactly at since boundary is included",
			records: []Record{
				{ID: 4, Amount: 75, Date: now.Add(-24 * time.Hour)},
			},
			since:   now.Add(-24 * time.Hour),
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a fresh DB for each sub-test to avoid cross-contamination.
			subDB, subCleanup := openTestDB(t)
			defer subCleanup()

			subS, err := NewBadgerStorage(subDB)
			require.NoError(t, err)

			for _, r := range tt.records {
				require.NoError(t, subS.SaveRecord(chatID, r))
			}

			retrieved, err := subS.GetRecords(chatID, tt.since)
			require.NoError(t, err)
			assert.Len(t, retrieved, tt.wantLen)
		})
	}
}
