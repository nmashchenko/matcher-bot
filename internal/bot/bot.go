package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"matcher-bot/internal/database"
	"matcher-bot/internal/events"
	"matcher-bot/internal/geocoding"
	"matcher-bot/internal/onboarding"
	"matcher-bot/internal/verification"

	"github.com/uptrace/bun"
	tele "gopkg.in/telebot.v4"
)

func New(token string, db *bun.DB) (*tele.Bot, error) {
	b, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	userStore := database.NewUserStore(db)
	eventStore := database.NewEventStore(db)
	geo := geocoding.NewGeocoder()
	verifSvc := verification.NewService(userStore, geo)
	obHandler := onboarding.NewHandler(userStore)
	evHandler := events.NewHandler(userStore, eventStore, geo, b)

	verifHandler := verification.NewHandler(verifSvc, obHandler.StartOnboarding)

	b.Handle("/start", handleStart(userStore, verifHandler, obHandler))

	// Location must be before middleware — unverified users need to share location.
	b.Handle(tele.OnLocation, locationDispatcher(userStore, verifHandler, evHandler))

	// Block all commands except /start and location for users who haven't completed onboarding.
	b.Use(requireReady(userStore))

	obHandler.Register(b)
	evHandler.Register(b)

	b.Handle(tele.OnText, handleText(userStore, obHandler, evHandler))

	b.Handle(tele.OnSticker, func(c tele.Context) error {
		return c.Send(fmt.Sprintf("Sticker file_id:\n<code>%s</code>", c.Message().Sticker.FileID), tele.ModeHTML)
	})

	_ = b.SetCommands([]tele.Command{
		{Text: "start", Description: "Начать / перезапустить бота"},
		{Text: "events", Description: "Найти события рядом"},
		{Text: "create", Description: "Создать событие"},
		{Text: "myevents", Description: "Мои события"},
	})

	// Background: expire events every 5 minutes.
	startEventLifecycle(eventStore)

	return b, nil
}

func startEventLifecycle(eventStore database.EventRepository) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			n, err := eventStore.ExpireEvents(ctx)
			if err != nil {
				slog.Error("expire events", "error", err)
				continue
			}
			if n > 0 {
				slog.Info("expired events", "count", n)
			}
		}
	}()
}
