package storage

import (
	"bytes"
	"fmt"

	"github.com/abdullin/lex-go/tuple"
)

// Namespace constants are the first element of every key tuple.
// They partition the key space so chat profiles and records never collide,
// and the metadata key lives in its own reserved namespace.
//
// Key layouts:
//
//	Chat profile:  tuple{nsChat,   chatID}             → gob-encoded ChatProfile
//	Record:        tuple{nsRecord, chatID, recordID}   → gob-encoded Record
//	Schema meta:   tuple{nsMeta}                       → gob-encoded SchemaMetadata
const (
	nsMeta   = int64(0) // schema version metadata
	nsChat   = int64(1) // chat profile keys
	nsRecord = int64(2) // record keys
)

// chatProfileKey returns the Badger key for a chat profile.
// Layout: tuple{nsChat, chatID}
func chatProfileKey(chatID int64) []byte {
	return tuple.Tuple{nsChat, chatID}.Pack()
}

// recordKey returns the Badger key for a specific record.
// Layout: tuple{nsRecord, chatID, recordID}
func recordKey(chatID, recordID int64) []byte {
	return tuple.Tuple{nsRecord, chatID, recordID}.Pack()
}

// recordPrefix returns the byte prefix shared by all records belonging to chatID.
// Because the FDB tuple layer encodes tuples such that a shorter tuple is always
// a byte-prefix of any longer tuple with the same leading elements, every key
// produced by recordKey(chatID, *) starts with these bytes.
// Layout: tuple{nsRecord, chatID}
func recordPrefix(chatID int64) []byte {
	return tuple.Tuple{nsRecord, chatID}.Pack()
}

// schemaMetaKey returns the single key used to store SchemaMetadata.
// Layout: tuple{nsMeta}
func schemaMetaKey() []byte {
	return tuple.Tuple{nsMeta}.Pack()
}

// decodeRecordKey unpacks a tuple key and returns (chatID, recordID).
// Returns an error if the key is not a valid 3-element record tuple.
func decodeRecordKey(key []byte) (chatID, recordID int64, err error) {
	t, err := tuple.Unpack(key)
	if err != nil {
		return 0, 0, fmt.Errorf("unpack record key: %w", err)
	}
	if len(t) != 3 {
		return 0, 0, fmt.Errorf("record key: expected 3 elements, got %d", len(t))
	}
	ns, ok := t[0].(int64)
	if !ok || ns != nsRecord {
		return 0, 0, fmt.Errorf("record key: wrong namespace %v", t[0])
	}
	chatID, ok = t[1].(int64)
	if !ok {
		return 0, 0, fmt.Errorf("record key: chatID is not int64: %T", t[1])
	}
	recordID, ok = t[2].(int64)
	if !ok {
		return 0, 0, fmt.Errorf("record key: recordID is not int64: %T", t[2])
	}
	return chatID, recordID, nil
}

// isRecordKey reports whether key is a record key (starts with the nsRecord prefix).
func isRecordKey(key []byte) bool {
	nsPrefix := tuple.Tuple{nsRecord}.Pack()
	return bytes.HasPrefix(key, nsPrefix)
}

// isChatProfileKey reports whether key is a chat profile key.
// A chat profile key unpacks to a 2-element tuple {nsChat, chatID}.
func isChatProfileKey(key []byte) bool {
	t, err := tuple.Unpack(key)
	if err != nil || len(t) != 2 {
		return false
	}
	ns, ok := t[0].(int64)
	return ok && ns == nsChat
}

// decodeChatProfileKey unpacks a chat profile key and returns the chatID.
func decodeChatProfileKey(key []byte) (int64, error) {
	t, err := tuple.Unpack(key)
	if err != nil {
		return 0, fmt.Errorf("unpack chat profile key: %w", err)
	}
	if len(t) != 2 {
		return 0, fmt.Errorf("chat profile key: expected 2 elements, got %d", len(t))
	}
	ns, ok := t[0].(int64)
	if !ok || ns != nsChat {
		return 0, fmt.Errorf("chat profile key: wrong namespace %v", t[0])
	}
	chatID, ok := t[1].(int64)
	if !ok {
		return 0, fmt.Errorf("chat profile key: chatID is not int64: %T", t[1])
	}
	return chatID, nil
}
