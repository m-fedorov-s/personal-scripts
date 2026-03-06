package handler

import (
	"context"
	"financer/internal/storage"
	"sort"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
)

type MockStorage struct {
	chats   map[int64]storage.ChatProfile
	records map[int64][]storage.Record
}

func (m *MockStorage) GetChat(chatID int64) (storage.ChatProfile, error) {
	cp, ok := m.chats[chatID]
	if !ok {
		return storage.ChatProfile{}, nil
	}
	return cp, nil
}

func (m *MockStorage) SetChat(chatID int64, cp *storage.ChatProfile) error {
	m.chats[chatID] = *cp
	return nil
}

func (m *MockStorage) SaveRecord(chatID int64, record storage.Record) error {
	m.records[chatID] = append(m.records[chatID], record)
	return nil
}

func (m *MockStorage) GetRecords(chatID int64, since time.Time) ([]storage.Record, error) {
	var result []storage.Record
	for _, r := range m.records[chatID] {
		if r.Date.After(since) || r.Date.Equal(since) {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockStorage) Chats() storage.ChatIterator {
	var ids []int64
	for id := range m.chats {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return &MockChatIterator{
		m:      m,
		ids:    ids,
		cursor: -1,
	}
}

type MockChatIterator struct {
	m      *MockStorage
	ids    []int64
	cursor int
}

func (it *MockChatIterator) Next() bool {
	it.cursor++
	return it.cursor < len(it.ids)
}

func (it *MockChatIterator) Value() (int64, storage.ChatProfile, error) {
	id := it.ids[it.cursor]
	return id, it.m.chats[id], nil
}

func (it *MockChatIterator) Close() error {
	return nil
}

func TestMessageHandler_Handle(t *testing.T) {
	s := &MockStorage{
		chats:   make(map[int64]storage.ChatProfile),
		records: make(map[int64][]storage.Record),
	}

	chatID := int64(123)
	s.chats[chatID] = storage.ChatProfile{
		DayLimit:       1000,
		CurrentBalance: 5000,
		LastRecordTime: time.Now(),
	}

	h := &MessageHandler{
		BaseHandler: BaseHandler{Storage: s},
	}

	update := &models.Update{
		Message: &models.Message{
			Chat: models.Chat{ID: chatID, Type: "group"},
			Text: "lunch 500",
		},
	}

	// We can't easily mock *bot.Bot because it's a struct, but Handle uses it for Reply.
	// For this test, we'll just check if the storage was updated correctly.
	// In a real scenario, we might want to interface-ize the bot interactions.

	h.Handle(context.Background(), nil, update)

	cp, _ := s.GetChat(chatID)
	if cp.CurrentBalance != 4500 {
		t.Errorf("Expected balance 4500, got %d", cp.CurrentBalance)
	}

	if len(s.records[chatID]) != 1 {
		t.Errorf("Expected 1 record, got %d", len(s.records[chatID]))
	}

	if s.records[chatID][0].Amount != 500 || s.records[chatID][0].Name != "lunch" {
		t.Errorf("Unexpected record: %+v", s.records[chatID][0])
	}
}
