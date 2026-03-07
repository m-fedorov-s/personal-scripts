package storage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRecordPrefix_IsPrefixOfRecordKey verifies the core invariant that the
// FDB tuple layer guarantees: recordPrefix(chatID) is always a byte-prefix of
// recordKey(chatID, recordID) for any recordID.
func TestRecordPrefix_IsPrefixOfRecordKey(t *testing.T) {
	tests := []struct {
		name     string
		chatID   int64
		recordID int64
	}{
		{"zero recordID", 12345, 0},
		{"positive recordID", 12345, 1},
		{"large recordID", 12345, 999999},
		{"max int64 recordID", 12345, 1<<62 - 1},
		{"different chatID", 99999, 42},
		{"chatID=1 recordID=1", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := recordPrefix(tt.chatID)
			key := recordKey(tt.chatID, tt.recordID)

			assert.True(t,
				bytes.HasPrefix(key, prefix),
				"recordKey(%d,%d) = %x does not start with recordPrefix(%d) = %x",
				tt.chatID, tt.recordID, key, tt.chatID, prefix,
			)
		})
	}
}

// TestRecordPrefix_DoesNotMatchOtherChat verifies that a prefix for chatID A
// does not match keys for chatID B.
func TestRecordPrefix_DoesNotMatchOtherChat(t *testing.T) {
	prefixA := recordPrefix(111)
	keyB := recordKey(222, 1)

	assert.False(t,
		bytes.HasPrefix(keyB, prefixA),
		"recordKey for chatID 222 should not match prefix for chatID 111",
	)
}

// TestKeyNamespaceIsolation verifies that chat profile keys, record keys, and
// the schema metadata key never share a common prefix with each other.
func TestKeyNamespaceIsolation(t *testing.T) {
	chatKey := chatProfileKey(42)
	recKey := recordKey(42, 1)
	metaKey := schemaMetaKey()

	assert.False(t, bytes.HasPrefix(recKey, chatKey),
		"record key should not start with chat profile key")
	assert.False(t, bytes.HasPrefix(chatKey, recKey),
		"chat profile key should not start with record key")
	assert.False(t, bytes.HasPrefix(metaKey, chatKey),
		"metadata key should not start with chat profile key")
	assert.False(t, bytes.HasPrefix(metaKey, recKey[:min(len(metaKey), len(recKey))]),
		"metadata key should not start with record key prefix")
}

// TestDecodeRecordKey_RoundTrip verifies that decodeRecordKey correctly
// unpacks a key produced by recordKey.
func TestDecodeRecordKey_RoundTrip(t *testing.T) {
	tests := []struct {
		chatID   int64
		recordID int64
	}{
		{1, 1},
		{12345, 99},
		{-1, 0},            // negative chatID (edge case)
		{0, 0},             // zero values
		{1 << 40, 1 << 40}, // large values
	}

	for _, tt := range tests {
		key := recordKey(tt.chatID, tt.recordID)
		gotChat, gotRecord, err := decodeRecordKey(key)
		require.NoError(t, err)
		assert.Equal(t, tt.chatID, gotChat)
		assert.Equal(t, tt.recordID, gotRecord)
	}
}

// TestDecodeChatProfileKey_RoundTrip verifies that decodeChatProfileKey
// correctly unpacks a key produced by chatProfileKey.
func TestDecodeChatProfileKey_RoundTrip(t *testing.T) {
	tests := []int64{1, 12345, -1, 0, 1 << 40}

	for _, chatID := range tests {
		key := chatProfileKey(chatID)
		got, err := decodeChatProfileKey(key)
		require.NoError(t, err)
		assert.Equal(t, chatID, got)
	}
}

// TestIsChatProfileKey verifies the key type detection helpers.
func TestIsChatProfileKey(t *testing.T) {
	tests := []struct {
		name     string
		key      []byte
		wantChat bool
		wantRec  bool
	}{
		{
			name:     "chat profile key",
			key:      chatProfileKey(42),
			wantChat: true,
			wantRec:  false,
		},
		{
			name:     "record key",
			key:      recordKey(42, 1),
			wantChat: false,
			wantRec:  true,
		},
		{
			name:     "schema meta key",
			key:      schemaMetaKey(),
			wantChat: false,
			wantRec:  false,
		},
		{
			name:     "empty key",
			key:      []byte{},
			wantChat: false,
			wantRec:  false,
		},
		{
			name:     "garbage key",
			key:      []byte{0xFF, 0xFE, 0xFD},
			wantChat: false,
			wantRec:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantChat, isChatProfileKey(tt.key), "isChatProfileKey")
			assert.Equal(t, tt.wantRec, isRecordKey(tt.key), "isRecordKey")
		})
	}
}

// TestRecordKeys_LexicographicOrder verifies that record keys for the same
// chatID are sorted in ascending recordID order, which enables range scans.
func TestRecordKeys_LexicographicOrder(t *testing.T) {
	chatID := int64(100)
	ids := []int64{1, 2, 10, 100, 1000}

	keys := make([][]byte, len(ids))
	for i, id := range ids {
		keys[i] = recordKey(chatID, id)
	}

	for i := 1; i < len(keys); i++ {
		assert.True(t,
			bytes.Compare(keys[i-1], keys[i]) < 0,
			"recordKey(chatID, %d) should be lexicographically less than recordKey(chatID, %d)",
			ids[i-1], ids[i],
		)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
