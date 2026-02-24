package bot

import (
	"context"
	"errors"
	"log/slog"

	"matcher-bot/internal/database"
	"matcher-bot/internal/messages"
	"matcher-bot/internal/onboarding"
	"matcher-bot/internal/verification"

	tele "gopkg.in/telebot.v4"
)

func handleStart(userStore database.UserRepository, verifHandler *verification.Handler, obHandler *onboarding.Handler) tele.HandlerFunc {
	return func(c tele.Context) error {
		sender := c.Sender()
		username := ptrStr(sender.Username)
		firstName := ptrStr(sender.FirstName)
		lastName := ptrStr(sender.LastName)

		user, err := userStore.FindOrCreate(context.Background(), sender.ID, username, firstName, lastName)
		if err != nil {
			slog.Error("find or create user", "error", err)
			return c.Send(messages.GenericError)
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
					messages.WelcomeBack(city, state),
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

func handleText(userStore database.UserRepository, obHandler *onboarding.Handler) tele.HandlerFunc {
	return func(c tele.Context) error {
		user, err := userStore.GetByTelegramID(context.Background(), c.Sender().ID)
		if err != nil {
			return c.Send(messages.StartPrompt)
		}

		if user.VerificationStatus == database.StatusVerified &&
			user.OnboardingStep != database.StepNone &&
			user.OnboardingStep != database.StepDone {
			err = obHandler.OnText(c)
			if !errors.Is(err, onboarding.ErrNotHandled) {
				return err
			}
		}

		return c.Send(messages.UnknownCommand)
	}
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
