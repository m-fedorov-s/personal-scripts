package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log/slog"

	badger "github.com/dgraph-io/badger/v4"
)

// currentSchemaVersion is the latest known schema version.
// Increment this constant whenever a new migration is added.
const currentSchemaVersion = uint32(1)

// SchemaMetadata is stored under schemaMetaKey() and tracks the DB schema version.
type SchemaMetadata struct {
	Version uint32
}

// RunMigrations reads the stored schema version and applies any pending
// migrations in order. It is called once during NewBadgerStorage initialization.
// If no metadata key exists the database is assumed to be at version 0 (legacy
// gob-encoded record keys) and all migrations are applied.
func RunMigrations(db *badger.DB) error {
	meta, err := readSchemaMeta(db)
	if err != nil {
		return fmt.Errorf("read schema metadata: %w", err)
	}

	if meta.Version >= currentSchemaVersion {
		slog.Debug("Storage schema is up to date", "version", meta.Version)
		return nil
	}

	slog.Info("Running storage migrations",
		"from_version", meta.Version,
		"to_version", currentSchemaVersion,
	)

	// Apply migrations sequentially.
	for v := meta.Version; v < currentSchemaVersion; v++ {
		switch v {
		case 0:
			slog.Info("Applying migration v0 → v1: re-encoding record keys with FDB tuple layer")
			if err := migrateV0toV1(db); err != nil {
				return fmt.Errorf("migration v0→v1: %w", err)
			}
		default:
			return fmt.Errorf("unknown migration step from version %d", v)
		}
	}

	// Persist the new schema version.
	if err := writeSchemaMeta(db, SchemaMetadata{Version: currentSchemaVersion}); err != nil {
		return fmt.Errorf("write schema metadata after migration: %w", err)
	}

	slog.Info("Storage migrations complete", "version", currentSchemaVersion)
	return nil
}

// migrateV0toV1 scans all keys in the database and re-encodes any legacy
// gob-encoded RecordKey keys into the new FDB tuple format.
//
// Legacy format (v0):
//
//	key   = gob(RecordKey{ChatID, RecordID})
//	value = gob(Record{...})
//
// New format (v1):
//
//	key   = tuple{nsRecord, chatID, recordID}.Pack()
//	value = gob(Record{...})  ← unchanged
//
// Chat profile keys (gob(int64)) are left untouched in this migration because
// they are also re-encoded by the storage layer on first write. However, to
// avoid a chicken-and-egg problem we also re-encode chat profile keys here.
//
// Legacy chat profile key: gob(chatID int64)
// New chat profile key:    tuple{nsChat, chatID}.Pack()
func migrateV0toV1(db *badger.DB) error {
	// Collect all legacy keys in a read-only scan first to avoid
	// modifying the iterator while iterating.
	type legacyEntry struct {
		oldKey []byte
		newKey []byte
		value  []byte
	}
	var entries []legacyEntry

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			rawKey := item.KeyCopy(nil)

			// Skip the schema metadata key itself (it uses the new format already
			// if written, or simply doesn't exist yet).
			if bytes.Equal(rawKey, schemaMetaKey()) {
				continue
			}

			var val []byte
			if err := item.Value(func(v []byte) error {
				val = append([]byte(nil), v...)
				return nil
			}); err != nil {
				return fmt.Errorf("read value for key %x: %w", rawKey, err)
			}

			// Try to decode as legacy RecordKey (gob-encoded struct).
			var rk RecordKey
			if err := decode(rawKey, &rk); err == nil && rk.ChatID != 0 {
				newKey := recordKey(rk.ChatID, rk.RecordID)
				entries = append(entries, legacyEntry{
					oldKey: rawKey,
					newKey: newKey,
					value:  val,
				})
				continue
			}

			// Try to decode as legacy chat profile key (gob-encoded int64).
			var chatID int64
			if err := decode(rawKey, &chatID); err == nil && chatID != 0 {
				newKey := chatProfileKey(chatID)
				entries = append(entries, legacyEntry{
					oldKey: rawKey,
					newKey: newKey,
					value:  val,
				})
				continue
			}

			// Unknown key format — log and skip to avoid data loss.
			slog.Warn("Migration v0→v1: skipping unrecognised key", "key_hex", fmt.Sprintf("%x", rawKey))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scan legacy keys: %w", err)
	}

	if len(entries) == 0 {
		slog.Info("Migration v0→v1: no legacy keys found, nothing to migrate")
		return nil
	}

	slog.Info("Migration v0→v1: re-encoding keys", "count", len(entries))

	// Write new keys and delete old keys in batches to stay within Badger's
	// transaction size limits.
	const batchSize = 100
	for i := 0; i < len(entries); i += batchSize {
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[i:end]

		if err := db.Update(func(txn *badger.Txn) error {
			for _, e := range batch {
				if err := txn.Set(e.newKey, e.value); err != nil {
					return fmt.Errorf("set new key %x: %w", e.newKey, err)
				}
				if err := txn.Delete(e.oldKey); err != nil {
					return fmt.Errorf("delete old key %x: %w", e.oldKey, err)
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("write migration batch [%d:%d]: %w", i, end, err)
		}
	}

	return nil
}

// readSchemaMeta reads the SchemaMetadata from the database.
// If the key does not exist it returns a zero-value metadata (Version=0).
func readSchemaMeta(db *badger.DB) (SchemaMetadata, error) {
	var meta SchemaMetadata
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(schemaMetaKey())
		if err == badger.ErrKeyNotFound {
			// No metadata → legacy database at version 0.
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			buf := bytes.NewBuffer(val)
			return gob.NewDecoder(buf).Decode(&meta)
		})
	})
	return meta, err
}

// writeSchemaMeta persists SchemaMetadata to the database.
func writeSchemaMeta(db *badger.DB, meta SchemaMetadata) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(meta); err != nil {
		return err
	}
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set(schemaMetaKey(), buf.Bytes())
	})
}
