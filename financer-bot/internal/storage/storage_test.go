package storage

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"
)

func TestChatProfile_ConsumeRecord(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	tests := []struct {
		name           string
		profile        ChatProfile
		record         Record
		expectedBal    int64
		expectedLastRT time.Time
	}{
		{
			name: "first record of the day",
			profile: ChatProfile{
				DayLimit:       1000,
				CurrentBalance: 0,
				LastRecordTime: yesterday,
			},
			record: Record{
				Date:   today,
				Amount: 300,
			},
			expectedBal:    700, // 0 + 1000 (1 day passed) - 300
			expectedLastRT: today,
		},
		{
			name: "second record of the day",
			profile: ChatProfile{
				DayLimit:       1000,
				CurrentBalance: 700,
				LastRecordTime: today,
			},
			record: Record{
				Date:   today,
				Amount: 200,
			},
			expectedBal:    500, // 700 - 200
			expectedLastRT: today,
		},
		{
			name: "multiple days passed",
			profile: ChatProfile{
				DayLimit:       1000,
				CurrentBalance: 500,
				LastRecordTime: today.AddDate(0, 0, -3),
			},
			record: Record{
				Date:   today,
				Amount: 400,
			},
			expectedBal:    3100, // 500 + 1000*3 - 400
			expectedLastRT: today,
		},
		{
			name: "new month",
			profile: ChatProfile{
				DayLimit:       1000,
				CurrentBalance: 5000,
				LastRecordTime: time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			record: Record{
				Date:   time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
				Amount: 100,
			},
			expectedBal:    900, // 1000*1 - 100 (balance resets on new month in current implementation)
			expectedLastRT: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.profile.ConsumeRecord(tt.record)
			if tt.profile.CurrentBalance != tt.expectedBal {
				t.Errorf("ConsumeRecord() balance = %v, want %v", tt.profile.CurrentBalance, tt.expectedBal)
			}
			if !tt.profile.LastRecordTime.Equal(tt.expectedLastRT) {
				t.Errorf("ConsumeRecord() lastRecordTime = %v, want %v", tt.profile.LastRecordTime, tt.expectedLastRT)
			}
		})
	}
}

func TestChatProfile_DecodeFrom(t *testing.T) {
	original := ChatProfile{
		TopicID:          123,
		DayLimit:         1500,
		CurrentBalance:   4500,
		ImmediateReports: true,
		ReportTime:       &DayTime{Hour: 10, Minute: 30},
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(original)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	var decoded ChatProfile
	err = decoded.DecodeFrom(buf.Bytes())
	if err != nil {
		t.Fatalf("DecodeFrom() error = %v", err)
	}

	if decoded.TopicID != original.TopicID ||
		decoded.DayLimit != original.DayLimit ||
		decoded.CurrentBalance != original.CurrentBalance ||
		decoded.ImmediateReports != original.ImmediateReports ||
		decoded.ReportTime.Hour != original.ReportTime.Hour ||
		decoded.ReportTime.Minute != original.ReportTime.Minute {
		t.Errorf("DecodeFrom() = %+v, want %+v", decoded, original)
	}
}
