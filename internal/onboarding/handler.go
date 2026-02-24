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

	pgvector "github.com/pgvector/pgvector-go"
	tele "gopkg.in/telebot.v4"
)

// ErrNotHandled indicates the message was not consumed by this handler.
var ErrNotHandled = errors.New("not handled")

type embedder interface {
	Embed(ctx context.Context, text string) (pgvector.Vector, error)
}

const cbGoal = "ob_goal"

type Handler struct {
	users database.UserRepository
	emb   embedder
}

func NewHandler(users database.UserRepository, emb embedder) *Handler {
	return &Handler{users: users, emb: emb}
}

func (h *Handler) Register(b *tele.Bot) {
	b.Handle("\f"+cbGoal, h.onGoal)
}

func (h *Handler) StartOnboarding(c tele.Context) error {
	// Guard: don't restart if onboarding is already in progress (age already set).
	if user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID); err == nil && user.Age != nil {
		slog.Info("onboarding already started, resuming", "telegram_id", c.Sender().ID)
		return h.ResumeOnboarding(c, user)
	}

	slog.Info("onboarding started", "telegram_id", c.Sender().ID)

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
			if err := h.users.Update(context.Background(), c.Sender().ID, save); err != nil {
				slog.Error("onboarding save", "error", err)
				return c.Send(messages.RestartError)
			}
			return c.Send(
				messages.AgeFromTelegram(age),
				h.buildGoalKeyboard(),
			)
		}
	}

	// Save avatar if we got one, then ask for age.
	if save.AvatarFileID != nil {
		if err := h.users.Update(context.Background(), c.Sender().ID, save); err != nil {
			slog.Error("onboarding save avatar", "error", err)
			return c.Send(messages.RestartError)
		}
	}
	return c.Send(messages.AskAge)
}

func (h *Handler) ResumeOnboarding(c tele.Context, user *database.User) error {
	switch {
	case user.Age == nil:
		return c.Send(messages.AskAge)
	case user.Goal == nil:
		return c.Send(messages.AskGoal, h.buildGoalKeyboard())
	case user.Bio == nil:
		return c.Send(messages.AskBio)
	case user.LookingFor == nil:
		return c.Send(messages.AskLookingFor)
	default:
		return nil
	}
}

// OnText handles text input during onboarding.
// Returns nil on success, or ErrNotHandled if the message was not consumed.
func (h *Handler) OnText(c tele.Context) error {
	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return ErrNotHandled
	}

	switch {
	case user.Age == nil:
		return h.onAge(c)
	case user.Goal == nil:
		return ErrNotHandled // waiting for callback
	case user.Bio == nil:
		return h.onBio(c)
	case user.LookingFor == nil:
		return h.onLookingFor(c)
	default:
		return ErrNotHandled
	}
}

func (h *Handler) onAge(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	age, err := strconv.Atoi(text)
	if err != nil || age < 16 || age > 99 {
		return c.Send(messages.InvalidAge)
	}

	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{Age: &age}); err != nil {
		slog.Error("onboarding save age", "error", err)
		return c.Send(messages.RestartError)
	}

	return c.Send(messages.AskGoal, h.buildGoalKeyboard())
}

func (h *Handler) onGoal(c tele.Context) error {
	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		slog.Error("onboarding get user", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.CallbackError})
	}
	if user.Goal != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.StepAlready})
	}

	goal := database.Goal(c.Callback().Data)
	if !ValidGoal(goal) {
		return c.Respond(&tele.CallbackResponse{Text: messages.UnknownGoal})
	}
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{Goal: &goal}); err != nil {
		slog.Error("onboarding save goal", "error", err)
		return c.Send(messages.RestartError)
	}

	_ = c.Respond()
	_ = c.Delete()
	return c.Send(messages.AskBio)
}

func (h *Handler) onBio(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if len([]rune(text)) < 20 {
		return c.Send(messages.TooShort)
	}

	vec, err := h.emb.Embed(context.Background(), text)
	if err != nil {
		slog.Error("onboarding embed bio", "error", err)
		return c.Send(messages.GenericError)
	}

	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{
		Bio:          &text,
		BioEmbedding: &vec,
	}); err != nil {
		slog.Error("onboarding save bio", "error", err)
		return c.Send(messages.RestartError)
	}

	return c.Send(messages.AskLookingFor)
}

func (h *Handler) onLookingFor(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if len([]rune(text)) < 20 {
		return c.Send(messages.TooShort)
	}

	vec, err := h.emb.Embed(context.Background(), text)
	if err != nil {
		slog.Error("onboarding embed looking_for", "error", err)
		return c.Send(messages.GenericError)
	}

	state := database.StateReady
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{
		LookingFor:          &text,
		LookingForEmbedding: &vec,
		UserState:           &state,
	}); err != nil {
		slog.Error("onboarding save looking_for", "error", err)
		return c.Send(messages.RestartError)
	}

	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		slog.Error("onboarding fetch completed user", "error", err)
		return c.Send(messages.RestartError)
	}

	name := c.Sender().FirstName
	age := 0
	if user.Age != nil {
		age = *user.Age
	}
	goalLabel := ""
	if user.Goal != nil {
		goalLabel = GoalLabel(*user.Goal)
	}
	bio := ""
	if user.Bio != nil {
		bio = *user.Bio
	}
	lookingFor := ""
	if user.LookingFor != nil {
		lookingFor = *user.LookingFor
	}
	city, geoState := "", ""
	if user.City != nil {
		city = *user.City
	}
	if user.State != nil {
		geoState = *user.State
	}

	caption := messages.ProfileComplete(name, age, goalLabel, bio, lookingFor, city, geoState)

	if user.AvatarFileID != nil {
		photo := &tele.Photo{
			File:    tele.File{FileID: *user.AvatarFileID},
			Caption: caption,
		}
		return c.Send(photo)
	}

	return c.Send(caption)
}

func (h *Handler) buildGoalKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, opt := range GoalOptions {
		btn := markup.Data(opt.Label, cbGoal, string(opt.Key))
		rows = append(rows, markup.Row(btn))
	}
	markup.Inline(rows...)
	return markup
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
