package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Environment struct {
	DB        *badger.DB
	botHandle string
}

func main() {
	var dataDir, token string
	flag.StringVar(&dataDir, "data", "./data/", "dir for data storing")
	flag.StringVar(&token, "token", "", "bot token")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := badger.Open(badger.DefaultOptions(dataDir))
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	// env := &Environment{DB: db}

	opts := []bot.Option{
		bot.WithDefaultHandler(getMessageHandler(db)),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, getStartHandler(db)),
		bot.WithMessageTextHandler("/setLimit", bot.MatchTypePrefix, getSetLimitHandler(db)),
	}
	b, err := bot.New(token, opts...)
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
	info, err := b.GetWebhookInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if info.URL != "" {
		b.DeleteWebhook(ctx, &bot.DeleteWebhookParams{})
	}
	slog.Info("Starting bot...")
	b.Start(ctx)
}

func ParseMessage(text string) []Record {
	dateRE, err := regexp.Compile("^\\d{1,2}\\.\\d{1,2}$")
	if err != nil {
		panic(err)
	}
	date := time.Now()
	lines := strings.Split(text, "\n")
	result := make([]Record, 0, len(lines))
	for _, line := range lines {
		if dateRE.Match([]byte(line)) {
			ints := strings.Split(line, ".")
			day, _ := strconv.ParseInt(ints[0], 10, 64)
			month, _ := strconv.ParseInt(ints[1], 10, 64)
			date = time.Date(date.Year(), time.Month(month), int(day), 0, 0, 0, 0, time.Now().UTC().Location())
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}
		nameBits := make([]string, 0, len(parts)-1)
		var amount uint32 = 0
		for _, part := range parts {
			parsed, err := strconv.ParseInt(part, 10, 32)
			if err != nil {
				nameBits = append(nameBits, part)
			} else {
				amount = uint32(parsed)
			}
		}
		if amount > 0 {
			result = append(result, Record{
				Date:   date,
				Name:   strings.Join(nameBits, " "),
				Amount: amount,
			})
		}
	}
	return result
}

var ChatTopicMismatch = fmt.Errorf("Wrong topic ID")
var NoRecordsError = fmt.Errorf("No records found")

func getMessageHandler(db *badger.DB) func(context.Context, *bot.Bot, *models.Update) {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			slog.Debug("update without message: %+v", update)
			return
		}
		if update.Message.Chat.Type == "private" {
			return
		}
		chatID := update.Message.Chat.ID
		var chatSettings ChatProfile
		err := db.Update(func(txn *badger.Txn) error {
			var err error
			chatSettings, err = GetChat(txn, chatID)
			if err != nil {
				return err
			}
			slog.Debug(fmt.Sprintf("GetChatProfile: %v", chatSettings))
			if chatSettings.TopicID != 0 && update.Message.MessageThreadID != chatSettings.TopicID {
				slog.Debug("Message from different topic", "chatID", chatID)
				return ChatTopicMismatch
			}
			records := ParseMessage(update.Message.Text)
			if len(records) == 0 {
				return NoRecordsError
			}
			for idx := range records {
				records[idx].ID = chatSettings.MaxRecordID + 1
				chatSettings.MaxRecordID += 1
			}
			slog.Debug(fmt.Sprintf("records: %v", records))
			for _, record := range records {
				chatSettings.ConsumeRecord(record)
				key := RecordKey{chatID, record.ID}
				encodedKey, err := Encode(key)
				if err != nil {
					log.Printf("error serializing record key %v : %v", key, err)
					return err
				}
				encodedRecord, err := Encode(record)
				if err != nil {
					log.Printf("error serializing record : %v", err)
					return err
				}
				e := badger.NewEntry(encodedKey, encodedRecord)
				txn.SetEntry(e)
			}
			slog.Debug(fmt.Sprintf("Put key %v, setting %+v", chatID, chatSettings))
			err = SetChat(txn, chatID, &chatSettings)
			return err
		})
		if err == ChatTopicMismatch || err == NoRecordsError {
			return
		}
		if err != nil {
			slog.Error("transaction failed", "error", err)
			_, err = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:           update.Message.Chat.ID,
				MessageThreadID:  update.Message.MessageThreadID,
				ReplyToMessageID: update.Message.ID,
				Text:             "Internal server error",
			})
		} else {
			_, err = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:           update.Message.Chat.ID,
				MessageThreadID:  update.Message.MessageThreadID,
				ReplyToMessageID: update.Message.ID,
				Text:             chatSettings.Describe(),
			})
		}
		if err != nil {
			slog.Error("Failed to send message: %v", err)
		}
	}
}

func getStartHandler(db *badger.DB) func(context.Context, *bot.Bot, *models.Update) {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message.Chat.Type == "private" {
			return
		}
		chatID := update.Message.Chat.ID
		err := db.Update(func(txn *badger.Txn) error {
			chatSettings, err := GetChat(txn, chatID)
			if err != nil {
				return err
			}
			chatSettings.TopicID = update.Message.MessageThreadID
			err = SetChat(txn, chatID, &chatSettings)
			return err
		})
		if err != nil {
			slog.Error("Error starting chat", "chatID", chatID, "error", err)
		}
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:           chatID,
			MessageThreadID:  update.Message.MessageThreadID,
			ReplyToMessageID: update.Message.ID,
			Text:             "Your chat is sucessfully registered!",
		})
		if err != nil {
			slog.Error("Failed to send message", "error", err)
		}

	}
}

func getSetLimitHandler(db *badger.DB) func(context.Context, *bot.Bot, *models.Update) {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message.Chat.Type == "private" {
			return
		}
		chatID := update.Message.Chat.ID
		parts := strings.Fields(update.Message.Text)
		if len(parts) != 2 {
			return
		}
		limit, err := strconv.ParseFloat(parts[1], 32)
		if err != nil {
			return
		}
		err = db.Update(func(txn *badger.Txn) error {
			chatSettings, err := GetChat(txn, chatID)
			if err != nil {
				return err
			}
			chatSettings.DayLimit = int64(limit)
			err = SetChat(txn, chatID, &chatSettings)
			return err
		})
		if err != nil {
			slog.Error("Error setting limits for chat", "chatID", chatID, "error", err)
		}
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:           chatID,
			MessageThreadID:  update.Message.MessageThreadID,
			ReplyToMessageID: update.Message.ID,
			Text:             fmt.Sprintf("New dayly limit:  %v", limit),
		})
		if err != nil {
			slog.Error("Failed to send message", "error", err)
		}
	}
}
