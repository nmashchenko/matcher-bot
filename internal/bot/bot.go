package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"matcher-bot/internal/database"
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

	svc := verification.NewService(db)
	handler := verification.NewHandler(svc)

	b.Handle("/start", handleStart(svc, handler))
	handler.Register(b)

	return b, nil
}

func handleStart(svc *verification.Service, handler *verification.Handler) tele.HandlerFunc {
	return func(c tele.Context) error {
		sender := c.Sender()
		username := ptrStr(sender.Username)
		firstName := ptrStr(sender.FirstName)
		lastName := ptrStr(sender.LastName)

		_, err := svc.FindOrCreateUser(context.Background(), sender.ID, username, firstName, lastName)
		if err != nil {
			log.Printf("find or create user error: %v", err)
			return c.Send("\u274C Произошла ошибка. Попробуй ещё раз.")
		}

		status, err := svc.GetVerificationStatus(context.Background(), sender.ID)
		if err != nil {
			log.Printf("get verification status error: %v", err)
			return c.Send("\u274C Произошла ошибка. Попробуй ещё раз.")
		}

		switch status.Status {
		case database.StatusVerified:
			return c.Send(
				fmt.Sprintf("С возвращением! Ты уже зарегистрирован (%s, %s). Скоро здесь будет подбор.", status.City, status.State),
				&tele.ReplyMarkup{RemoveKeyboard: true},
			)
		default:
			return handler.SendVerificationPrompt(c)
		}
	}
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
