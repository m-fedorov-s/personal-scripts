package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-telegram/bot"

	"financer/internal/handler"
	"financer/internal/storage"
	"financer/internal/worker"
)

func main() {
	var dataDir, token string
	flag.StringVar(&dataDir, "data", "./data/", "dir for data storing")
	flag.StringVar(&token, "token", "", "bot token")
	flag.Parse()

	if token == "" {
		token = os.Getenv("BOT_TOKEN")
	}
	if dataDir == "./data/" && os.Getenv("DATA_DIR") != "" {
		dataDir = os.Getenv("DATA_DIR")
	}

	if token == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := badger.Open(badger.DefaultOptions(dataDir))
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	s := storage.NewBadgerStorage(db)

	// Create bot instance first
	b, err := bot.New(token)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start report worker
	rw := worker.NewReportWorker(b, s, 10)
	go rw.Start(ctx)

	// Register handlers
	opts := handler.RegisterHandlers(b, s, rw.Reschedule)
	for _, opt := range opts {
		opt(b)
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
