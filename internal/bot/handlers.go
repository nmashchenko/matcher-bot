package bot

import (
	"context"
	"errors"
	"log/slog"

	"matcher-bot/internal/database"
	"matcher-bot/internal/events"
	"matcher-bot/internal/messages"
	"matcher-bot/internal/onboarding"
	"matcher-bot/internal/settings"
	"matcher-bot/internal/util"
	"matcher-bot/internal/verification"

	tele "gopkg.in/telebot.v4"
)

// requireReady is middleware that blocks commands/callbacks for users who haven't completed onboarding.
// /start is registered before this middleware so it's unaffected.
func requireReady(userStore database.UserRepository) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			user, err := userStore.GetByTelegramID(context.Background(), c.Sender().ID)
			if err != nil || user.UserState != database.StateReady {
				if err == nil {
					switch user.UserState {
					case database.StateOnboarding:
						// Allow plain text and location through for onboarding handlers.
						if c.Callback() == nil && c.Message() != nil && !c.Message().IsService() {
							return next(c)
						}
						return c.Send(messages.FinishOnboardingFirst)
					case database.StateUnverified:
						return c.Send(messages.ShareLocationFirst)
					}
				}
				return c.Send(messages.StartPrompt)
			}
			return next(c)
		}
	}
}

func handleStart(userStore database.UserRepository, verifHandler *verification.Handler, obHandler *onboarding.Handler) tele.HandlerFunc {
	return func(c tele.Context) error {
		sender := c.Sender()
		username := util.Str(sender.Username)
		firstName := util.Str(sender.FirstName)
		lastName := util.Str(sender.LastName)

		user, err := userStore.FindOrCreate(context.Background(), sender.ID, username, firstName, lastName)
		if err != nil {
			slog.Error("find or create user", "error", err)
			return c.Send(messages.GenericError)
		}

		switch user.UserState {
		case database.StateReady:
			city, state := "", ""
			if user.City != nil {
				city = *user.City
			}
			if user.State != nil {
				state = *user.State
			}
			return c.Send(
				messages.MainMenu(city, state),
				&tele.ReplyMarkup{RemoveKeyboard: true},
			)
		case database.StateOnboarding:
			return obHandler.ResumeOnboarding(c, user)
		default:
			return verifHandler.SendVerificationPrompt(c)
		}
	}
}

func handleText(userStore database.UserRepository, obHandler *onboarding.Handler, evHandler *events.Handler, setHandler *settings.Handler) tele.HandlerFunc {
	return func(c tele.Context) error {
		user, err := userStore.GetByTelegramID(context.Background(), c.Sender().ID)
		if err != nil {
			return c.Send(messages.StartPrompt)
		}

		// Onboarding takes priority.
		if user.UserState == database.StateOnboarding {
			err = obHandler.OnText(c)
			if !errors.Is(err, onboarding.ErrNotHandled) {
				return err
			}
		}

		// Event creation wizard.
		if evHandler.IsCreating(c.Sender().ID) {
			return evHandler.OnCreateText(c)
		}

		// Settings reply keyboard buttons.
		if setHandler.OnText(c) {
			return nil
		}

		return c.Send(messages.UnknownCommand)
	}
}

func locationDispatcher(userStore database.UserRepository, verifHandler *verification.Handler, evHandler *events.Handler) tele.HandlerFunc {
	return func(c tele.Context) error {
		user, err := userStore.GetByTelegramID(context.Background(), c.Sender().ID)
		if err != nil {
			// User not found — treat as unverified.
			return verifHandler.OnLocation(c)
		}

		switch {
		case user.UserState == database.StateUnverified:
			return verifHandler.OnLocation(c)
		case evHandler.IsCreating(c.Sender().ID):
			return evHandler.OnCreateLocation(c)
		default:
			return nil
		}
	}
}

