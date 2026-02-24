package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"matcher-bot/internal/bot"
	"matcher-bot/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}

	db, err := database.New(dsn)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	b, err := bot.New(token, db, openaiKey)
	if err != nil {
		log.Fatalf("bot creation failed: %v", err)
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		slog.Info("shutting down")
		b.Stop()
	}()

	slog.Info("bot started")
	b.Start()
}
