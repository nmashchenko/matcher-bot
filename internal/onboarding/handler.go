package onboarding

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"matcher-bot/internal/database"

	pgvector "github.com/pgvector/pgvector-go"
	tele "gopkg.in/telebot.v4"
)

type userStore interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*database.User, error)
	Update(ctx context.Context, telegramID int64, data *database.UserUpdateData) error
}

type embedder interface {
	Embed(ctx context.Context, text string) (pgvector.Vector, error)
}

const cbGoal = "ob_goal"

type Handler struct {
	users userStore
	emb   embedder
}

func NewHandler(users userStore, emb embedder) *Handler {
	return &Handler{users: users, emb: emb}
}

func (h *Handler) Register(b *tele.Bot) {
	b.Handle("\f"+cbGoal, h.onGoal)
}

func (h *Handler) StartOnboarding(c tele.Context) error {
	// Guard: don't restart if onboarding is already in progress or done.
	if user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID); err == nil {
		if user.OnboardingStep != database.StepNone {
			log.Printf("user %d: onboarding already at step %q, skipping start", c.Sender().ID, user.OnboardingStep)
			return nil
		}
	}

	log.Printf("user %d started onboarding", c.Sender().ID)

	save := &database.UserUpdateData{}

	// Try to grab avatar from Telegram profile.
	if photos, err := c.Bot().ProfilePhotosOf(c.Sender()); err == nil && len(photos) > 0 {
		fid := photos[0].FileID
		save.AvatarFileID = &fid
	}

	// Try to grab birthdate from Telegram and derive age.
	if chat, err := c.Bot().ChatByID(c.Sender().ID); err == nil {
		log.Printf("Parsed birthdate: %d", chat.Birthdate)
		age := calcAge(chat.Birthdate)
		if age >= 16 && age <= 99 {
			save.Age = &age
			step := database.StepGoal
			save.OnboardingStep = &step
			if err := h.users.Update(context.Background(), c.Sender().ID, save); err != nil {
				log.Printf("onboarding save error: %v", err)
				return c.Send("\u274C Произошла ошибка. Попробуй /start заново.")
			}
			return c.Send(
				fmt.Sprintf("Мне удалось определить твой возраст из Telegram: %d. Что ищешь?", age),
				h.buildGoalKeyboard(),
			)
		}
	}

	// Default path: ask for age.
	step := database.StepAge
	save.OnboardingStep = &step
	if err := h.users.Update(context.Background(), c.Sender().ID, save); err != nil {
		log.Printf("onboarding save step error: %v", err)
		return c.Send("\u274C Произошла ошибка. Попробуй /start заново.")
	}
	return c.Send("Сколько тебе лет?")
}

func (h *Handler) ResumeOnboarding(c tele.Context, step database.OnboardingStep) error {
	switch step {
	case database.StepAge:
		return c.Send("Сколько тебе лет?")
	case database.StepGoal:
		return c.Send("Что ищешь?", h.buildGoalKeyboard())
	case database.StepBio:
		return c.Send("Расскажи о себе в 2-3 предложениях:")
	case database.StepLookingFor:
		return c.Send("Опиши, кого ищешь или что для тебя важно в общении:")
	default:
		return nil
	}
}

// OnText handles text input during onboarding. Returns (handled, error).
func (h *Handler) OnText(c tele.Context) (bool, error) {
	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return false, nil
	}

	switch user.OnboardingStep {
	case database.StepAge:
		return h.onAge(c)
	case database.StepBio:
		return h.onBio(c)
	case database.StepLookingFor:
		return h.onLookingFor(c)
	default:
		return false, nil
	}
}

func (h *Handler) onAge(c tele.Context) (bool, error) {
	text := strings.TrimSpace(c.Text())
	age, err := strconv.Atoi(text)
	if err != nil || age < 16 || age > 99 {
		return true, c.Send("Введи возраст от 16 до 99.")
	}

	step := database.StepGoal
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{Age: &age, OnboardingStep: &step}); err != nil {
		log.Printf("onboarding save age error: %v", err)
		return true, c.Send("\u274C Произошла ошибка. Попробуй /start заново.")
	}

	return true, c.Send("Что ищешь?", h.buildGoalKeyboard())
}

func (h *Handler) onGoal(c tele.Context) error {
	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		log.Printf("onboarding get user error: %v", err)
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка, попробуй /start."})
	}
	if user.OnboardingStep != database.StepGoal {
		return c.Respond(&tele.CallbackResponse{Text: "Этот шаг уже пройден."})
	}

	goal := database.Goal(c.Callback().Data)
	step := database.StepBio
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{Goal: &goal, OnboardingStep: &step}); err != nil {
		log.Printf("onboarding save goal error: %v", err)
		return c.Send("\u274C Произошла ошибка. Попробуй /start заново.")
	}

	_ = c.Respond()
	_ = c.Delete()
	return c.Send("Расскажи о себе в 2-3 предложениях:")
}

func (h *Handler) onBio(c tele.Context) (bool, error) {
	text := strings.TrimSpace(c.Text())
	if len([]rune(text)) < 20 {
		return true, c.Send("Слишком коротко — напиши хотя бы 20 символов.")
	}

	vec, err := h.emb.Embed(context.Background(), text)
	if err != nil {
		log.Printf("onboarding embed bio error: %v", err)
		return true, c.Send("\u274C Произошла ошибка. Попробуй ещё раз.")
	}

	step := database.StepLookingFor
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{
		Bio:            &text,
		BioEmbedding:   &vec,
		OnboardingStep: &step,
	}); err != nil {
		log.Printf("onboarding save bio error: %v", err)
		return true, c.Send("\u274C Произошла ошибка. Попробуй /start заново.")
	}

	return true, c.Send("Опиши, кого ищешь или что для тебя важно в общении:")
}

func (h *Handler) onLookingFor(c tele.Context) (bool, error) {
	text := strings.TrimSpace(c.Text())
	if len([]rune(text)) < 20 {
		return true, c.Send("Слишком коротко — напиши хотя бы 20 символов.")
	}

	vec, err := h.emb.Embed(context.Background(), text)
	if err != nil {
		log.Printf("onboarding embed looking_for error: %v", err)
		return true, c.Send("\u274C Произошла ошибка. Попробуй ещё раз.")
	}

	step := database.StepDone
	if err := h.users.Update(context.Background(), c.Sender().ID, &database.UserUpdateData{
		LookingFor:          &text,
		LookingForEmbedding: &vec,
		OnboardingStep:      &step,
	}); err != nil {
		log.Printf("onboarding save looking_for error: %v", err)
		return true, c.Send("\u274C Произошла ошибка. Попробуй /start заново.")
	}

	return true, c.Send("Окей, я запомнил. Скоро начнём подбор!")
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
