package main

import (
	"log"
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

	db, err := database.New(dsn)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	b, err := bot.New(token, db)
	if err != nil {
		log.Fatalf("bot creation failed: %v", err)
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("Shutting down...")
		b.Stop()
	}()

	log.Println("Bot started")
	b.Start()
}
