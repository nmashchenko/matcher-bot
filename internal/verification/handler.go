package verification

import (
	"context"
	"log/slog"

	"matcher-bot/internal/messages"

	tele "gopkg.in/telebot.v4"
)

type Handler struct {
	svc        *Service
	onVerified func(tele.Context) error
}

func NewHandler(svc *Service, onVerified func(tele.Context) error) *Handler {
	return &Handler{svc: svc, onVerified: onVerified}
}

func (h *Handler) Register(b *tele.Bot) {
	b.Handle(tele.OnLocation, h.OnLocation)
}

func (h *Handler) SendVerificationPrompt(c tele.Context) error {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	markup.Reply(
		markup.Row(markup.Location(messages.VerificationButton)),
	)

	return c.Send(messages.VerificationIntro, markup)
}

func (h *Handler) OnLocation(c tele.Context) error {
	loc := c.Message().Location
	if loc == nil {
		return nil
	}

	if err := c.Send(messages.CheckingLocation); err != nil {
		slog.Error("send message", "error", err)
	}

	result, err := h.svc.VerifyByLocation(context.Background(), c.Sender().ID, float64(loc.Lat), float64(loc.Lng))
	if err != nil {
		slog.Error("verify by location", "error", err, "telegram_id", c.Sender().ID)
		return c.Send(messages.GenericError)
	}

	if result.Error == "geocoding_failed" {
		return c.Send(messages.GeocodingError)
	}

	if result.Verified {
		if err := c.Send(
			messages.Verified(result.City, result.State),
			&tele.ReplyMarkup{RemoveKeyboard: true},
		); err != nil {
			slog.Error("send verified message", "error", err)
		}
		if h.onVerified != nil {
			return h.onVerified(c)
		}
		return nil
	}

	return c.Send(messages.NotInUSA)
}
