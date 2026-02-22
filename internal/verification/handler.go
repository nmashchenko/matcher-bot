package verification

import (
	"context"
	"fmt"
	"log"

	tele "gopkg.in/telebot.v4"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
		return c.Send(
			fmt.Sprintf("\u2705 Подтверждено! Ты в %s, %s.\n\nОтлично, теперь можно переходить к настройке профиля. (Скоро будет доступно)", result.City, result.State),
			&tele.ReplyMarkup{RemoveKeyboard: true},
		)
	}

	return c.Send("\u274C Похоже, ты не в США. Этот бот пока работает только для людей в Штатах.\n\n" +
		"Если ты считаешь, что это ошибка — попробуй отправить геолокацию ещё раз.")
}
