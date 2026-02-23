package bot

import (
	"context"
	"fmt"
	"log"

	"matcher-bot/internal/database"
	"matcher-bot/internal/onboarding"
	"matcher-bot/internal/verification"

	tele "gopkg.in/telebot.v4"
)

func handleStart(userStore *database.UserStore, verifHandler *verification.Handler, obHandler *onboarding.Handler) tele.HandlerFunc {
	return func(c tele.Context) error {
		sender := c.Sender()
		username := ptrStr(sender.Username)
		firstName := ptrStr(sender.FirstName)
		lastName := ptrStr(sender.LastName)

		user, err := userStore.FindOrCreate(context.Background(), sender.ID, username, firstName, lastName)
		if err != nil {
			log.Printf("find or create user error: %v", err)
			return c.Send("\u274C Произошла ошибка. Попробуй ещё раз.")
		}

		switch user.VerificationStatus {
		case database.StatusVerified:
			switch user.OnboardingStep {
			case database.StepDone:
				city, state := "", ""
				if user.City != nil {
					city = *user.City
				}
				if user.State != nil {
					state = *user.State
				}
				return c.Send(
					fmt.Sprintf("С возвращением! Ты уже зарегистрирован (%s, %s). Скоро здесь будет подбор.", city, state),
					&tele.ReplyMarkup{RemoveKeyboard: true},
				)
			case database.StepNone:
				return obHandler.StartOnboarding(c)
			default:
				return obHandler.ResumeOnboarding(c, user.OnboardingStep)
			}
		default:
			return verifHandler.SendVerificationPrompt(c)
		}
	}
}

func handleText(userStore *database.UserStore, obHandler *onboarding.Handler) tele.HandlerFunc {
	return func(c tele.Context) error {
		user, err := userStore.GetByTelegramID(context.Background(), c.Sender().ID)
		if err != nil {
			return c.Send("Напиши /start чтобы начать.")
		}

		if user.VerificationStatus == database.StatusVerified &&
			user.OnboardingStep != database.StepNone &&
			user.OnboardingStep != database.StepDone {
			handled, err := obHandler.OnText(c)
			if handled {
				return err
			}
		}

		return c.Send("Не понял тебя. Выбери, что хочешь сделать, или напиши /start.")
	}
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
