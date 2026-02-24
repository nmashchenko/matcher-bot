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
	// Guard: don't restart if onboarding is already in progress or done.
	if user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID); err == nil {
		if user.OnboardingStep != database.StepNone {
			slog.Info("onboarding already in progress", "telegram_id", c.Sender().ID, "step", user.OnboardingStep)
			return nil
		}
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
			step := database.StepGoal
			save.OnboardingStep = &step
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

	step := database.StepAge
	save.OnboardingStep = &step
	if err := h.users.Update(context.Background(), c.Sender().ID, save); err != nil {
		slog.Error("onboarding save step", "error", err)
		return c.Send(messages.RestartError)
	}
	return c.Send(messages.AskAge)
}

func (h *Handler) ResumeOnboarding(c tele.Context, step database.OnboardingStep) error {
	switch step {
	case database.StepAge:
		return c.Send(messages.AskAge)
	case database.StepGoal:
		return c.Send(messages.AskGoal, h.buildGoalKeyboard())
	case database.StepBio:
		return c.Send(messages.AskBio)
	case database.StepLookingFor:
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

	switch user.OnboardingStep {
	case database.StepAge:
		return h.onAge(c)
	case database.StepBio:
		return h.onBio(c)
	case database.StepLookingFor:
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

	step := database.StepGoal
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{Age: &age, OnboardingStep: &step}); err != nil {
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
	if user.OnboardingStep != database.StepGoal {
		return c.Respond(&tele.CallbackResponse{Text: messages.StepAlready})
	}

	goal := database.Goal(c.Callback().Data)
	if !ValidGoal(goal) {
		return c.Respond(&tele.CallbackResponse{Text: messages.UnknownGoal})
	}
	step := database.StepBio
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{Goal: &goal, OnboardingStep: &step}); err != nil {
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

	step := database.StepLookingFor
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{
		Bio:            &text,
		BioEmbedding:   &vec,
		OnboardingStep: &step,
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

	step := database.StepDone
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{
		LookingFor:          &text,
		LookingForEmbedding: &vec,
		OnboardingStep:      &step,
	}); err != nil {
		slog.Error("onboarding save looking_for", "error", err)
		return c.Send(messages.RestartError)
	}

	return c.Send(messages.OnboardingDone)
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
