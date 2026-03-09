package onboarding

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"matcher-bot/internal/database"
	"matcher-bot/internal/messages"
	"matcher-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

// ErrNotHandled indicates the message was not consumed by this handler.
var ErrNotHandled = errors.New("not handled")

type Handler struct {
	users database.UserRepository
}

func NewHandler(users database.UserRepository) *Handler {
	return &Handler{users: users}
}

func (h *Handler) Register(_ *tele.Bot) {
	// No callbacks to register after simplification.
}

func (h *Handler) StartOnboarding(c tele.Context) error {
	// Guard: don't restart if onboarding is already in progress (age already set).
	if user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID); err == nil && user.Age != nil {
		slog.Info("onboarding already started, resuming", "telegram_id", c.Sender().ID)
		return h.ResumeOnboarding(c, user)
	}

	// Send welcome sticker.
	if messages.WelcomeSticker != "REPLACE_ME" {
		_ = c.Send(&tele.Sticker{File: tele.File{FileID: messages.WelcomeSticker}})
	}

	slog.Info("onboarding started", "telegram_id", c.Sender().ID)

	// Fetch user to get city/state from verification.
	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		slog.Error("onboarding fetch user", "error", err)
		return c.Send(messages.RestartError)
	}
	city, geoState := util.Deref(user.City), util.Deref(user.State)

	save := &database.UserUpdateData{}

	// Try to grab avatar from Telegram profile.
	if photos, err := c.Bot().ProfilePhotosOf(c.Sender()); err == nil && len(photos) > 0 {
		fid := photos[0].FileID
		save.AvatarFileID = &fid
	}

	// Try to grab birthdate from Telegram and derive age.
	if chat, err := c.Bot().ChatByID(c.Sender().ID); err == nil {
		slog.Debug("parsed birthdate", "birthdate", chat.Birthdate)
		age := calcAge(chat.Birthdate)
		if age >= 16 && age <= 99 {
			save.Age = &age
			state := database.StateReady
			save.UserState = &state
			if err := h.users.Update(context.Background(), c.Sender().ID, save); err != nil {
				slog.Error("onboarding save", "error", err)
				return c.Send(messages.RestartError)
			}
			name := c.Sender().FirstName
			if err := c.Send(messages.OnboardingComplete(name, age, city, geoState)); err != nil {
				return err
			}
			if c.Sender().Username == "" {
				return c.Send(messages.UsernameWarning)
			}
			return nil
		}
	}

	// Save avatar if we got one, then ask for age.
	if save.AvatarFileID != nil {
		if err := h.users.Update(context.Background(), c.Sender().ID, save); err != nil {
			slog.Error("onboarding save avatar", "error", err)
			return c.Send(messages.RestartError)
		}
	}
	return c.Send(messages.AskAge(city, geoState))
}

func (h *Handler) ResumeOnboarding(c tele.Context, user *database.User) error {
	city, geoState := util.Deref(user.City), util.Deref(user.State)
	if user.Age == nil {
		return c.Send(messages.AskAge(city, geoState))
	}
	// Age is set — mark as ready.
	return nil
}

// OnText handles text input during onboarding.
func (h *Handler) OnText(c tele.Context) error {
	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return ErrNotHandled
	}
	if user.Age == nil {
		return h.onAge(c)
	}
	return ErrNotHandled
}

func (h *Handler) onAge(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	age, err := strconv.Atoi(text)
	if err != nil || age < 16 || age > 99 {
		return c.Send(messages.InvalidAge)
	}

	state := database.StateReady
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{
		Age:       &age,
		UserState: &state,
	}); err != nil {
		slog.Error("onboarding save age", "error", err)
		return c.Send(messages.RestartError)
	}

	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return c.Send(messages.RestartError)
	}
	city, geoState := util.Deref(user.City), util.Deref(user.State)
	name := c.Sender().FirstName

	if err := c.Send(messages.OnboardingComplete(name, age, city, geoState)); err != nil {
		return err
	}
	if c.Sender().Username == "" {
		return c.Send(messages.UsernameWarning)
	}
	return nil
}

func calcAge(bd tele.Birthdate) int {
	now := time.Now()
	age := now.Year() - bd.Year
	if now.Month() < time.Month(bd.Month) ||
		(now.Month() == time.Month(bd.Month) && now.Day() < bd.Day) {
		age--
	}
	return age
}
