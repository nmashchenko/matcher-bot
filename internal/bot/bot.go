package bot

import (
	"time"

	"matcher-bot/internal/database"
	"matcher-bot/internal/embeddings"
	"matcher-bot/internal/onboarding"
	"matcher-bot/internal/verification"

	"github.com/uptrace/bun"
	tele "gopkg.in/telebot.v4"
)

func New(token string, db *bun.DB, openaiKey string) (*tele.Bot, error) {
	b, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	userStore := database.NewUserStore(db)
	verifSvc := verification.NewService(db)
	embClient := embeddings.NewClient(openaiKey)
	obHandler := onboarding.NewHandler(userStore, embClient)

	verifHandler := verification.NewHandler(verifSvc, obHandler.StartOnboarding)

	b.Handle("/start", handleStart(userStore, verifHandler, obHandler))

	verifHandler.Register(b)
	obHandler.Register(b)

	b.Handle(tele.OnText, handleText(userStore, obHandler))

	return b, nil
}
