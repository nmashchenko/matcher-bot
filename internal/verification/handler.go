package verification

import (
	"context"
	"fmt"
	"log"

	tele "gopkg.in/telebot.v4"
)

type verificationService interface {
	VerifyByLocation(ctx context.Context, telegramID int64, lat, lon float64) (*VerifyResult, error)
	GetVerificationStatus(ctx context.Context, telegramID int64) (*StatusResult, error)
}

type Handler struct {
	svc        verificationService
	onVerified func(tele.Context) error
}

func NewHandler(svc verificationService, onVerified func(tele.Context) error) *Handler {
	return &Handler{svc: svc, onVerified: onVerified}
}

func (h *Handler) Register(b *tele.Bot) {
	b.Handle(tele.OnLocation, h.OnLocation)
}

func (h *Handler) SendVerificationPrompt(c tele.Context) error {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	markup.Reply(
		markup.Row(markup.Location("\U0001F4CD Поделиться геолокацией")),
	)

	return c.Send(
		"Привет! Я — Matcher Bot. Помогу найти интересных людей из СНГ рядом с тобой в США.\n\n"+
			"Для начала мне нужно убедиться, что ты в США. Поделись геолокацией — это одноразово и безопасно.",
		markup,
	)
}

func (h *Handler) OnLocation(c tele.Context) error {
	loc := c.Message().Location
	if loc == nil {
		return nil
	}

	if err := c.Send("\u23F3 Проверяю твою геолокацию..."); err != nil {
		log.Printf("send error: %v", err)
	}

	result, err := h.svc.VerifyByLocation(context.Background(), c.Sender().ID, float64(loc.Lat), float64(loc.Lng))
	if err != nil {
		log.Printf("verify by location error: %v", err)
		return c.Send("\u274C Произошла ошибка. Попробуй ещё раз.")
	}

	if result.Error == "geocoding_failed" {
		return c.Send("\u274C Не удалось определить местоположение. Попробуй ещё раз.")
	}

	if result.Verified {
		if err := c.Send(
			fmt.Sprintf("\u2705 Подтверждено! Ты в %s, %s.", result.City, result.State),
			&tele.ReplyMarkup{RemoveKeyboard: true},
		); err != nil {
			log.Printf("send verified msg error: %v", err)
		}
		if h.onVerified != nil {
			return h.onVerified(c)
		}
		return nil
	}

	return c.Send("\u274C Похоже, ты не в США. Этот бот пока работает только для людей в Штатах.\n\n" +
		"Если ты считаешь, что это ошибка — попробуй отправить геолокацию ещё раз.")
}
