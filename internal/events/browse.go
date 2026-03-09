package events

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"matcher-bot/internal/database"
	"matcher-bot/internal/messages"

	tele "gopkg.in/telebot.v4"
)

func (h *Handler) cmdBrowse(c tele.Context) error {
	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return c.Send(messages.StartPrompt)
	}

	city, state := "", ""
	if user.City != nil {
		city = *user.City
	}
	if user.State != nil {
		state = *user.State
	}

	return h.showNextEvent(c, city, state, false)
}

func (h *Handler) onBrowseNext(c tele.Context) error {
	if eventID := c.Callback().Data; eventID != "" {
		if err := h.events.MarkViewed(context.Background(), c.Sender().ID, eventID); err != nil {
			slog.Error("browse next: mark viewed", "error", err)
		}
	}

	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.BrowseExpired})
	}

	city, state := "", ""
	if user.City != nil {
		city = *user.City
	}
	if user.State != nil {
		state = *user.State
	}

	_ = c.Respond()
	return h.showNextEvent(c, city, state, true)
}

func (h *Handler) onBrowseJoin(c tele.Context) error {
	eventID := c.Callback().Data
	if eventID == "" {
		return c.Respond()
	}

	if err := h.events.MarkViewed(context.Background(), c.Sender().ID, eventID); err != nil {
		slog.Error("browse join: mark viewed", "error", err)
	}

	event, err := h.events.GetByID(context.Background(), eventID)
	if err != nil {
		slog.Error("browse join: get event", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	_, err = h.events.GetParticipant(context.Background(), eventID, c.Sender().ID)
	if err == nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.AlreadyJoined})
	}
	if !errors.Is(err, sql.ErrNoRows) {
		slog.Error("browse join: get participant", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	approved, err := h.events.CountApproved(context.Background(), eventID)
	if err != nil {
		slog.Error("browse join: count approved", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}
	if approved >= event.MaxParticipants {
		return c.Respond(&tele.CallbackResponse{Text: messages.EventFull})
	}

	if err := h.events.RequestJoin(context.Background(), eventID, c.Sender().ID); err != nil {
		slog.Error("browse join: request", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	_ = c.Respond(&tele.CallbackResponse{Text: messages.JoinSent})
	h.notifyHostJoinRequest(c, event)

	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return nil
	}
	city, state := "", ""
	if user.City != nil {
		city = *user.City
	}
	if user.State != nil {
		state = *user.State
	}
	return h.showNextEvent(c, city, state, true)
}

func (h *Handler) showNextEvent(c tele.Context, city, state string, edit bool) error {
	ctx := context.Background()
	telegramID := c.Sender().ID

	event, err := h.events.NextUnseen(ctx, city, state, telegramID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if edit {
				return c.Edit(messages.BrowseEnd)
			}
			return c.Send(messages.BrowseEmpty)
		}
		slog.Error("browse next unseen", "error", err)
		return c.Send(messages.GenericError)
	}

	emoji := EventTypeEmoji(event.EventType)
	label := EventTypeLabel(event.EventType)
	desc := ""
	if event.Description != nil {
		desc = *event.Description
	}
	approved, _ := h.events.CountApproved(ctx, event.ID)
	card := messages.EventCard(emoji, label, event.Title, desc, event.City, event.StartsAt, approved, event.MaxParticipants)

	markup := &tele.ReplyMarkup{}
	btnJoin := markup.Data("\U0001f64b Хочу пойти", "bj", event.ID)
	btnNext := markup.Data("\u25b6\ufe0f Дальше", "bn", event.ID)
	markup.Inline(markup.Row(btnJoin, btnNext))

	if edit {
		return c.Edit(card, markup)
	}
	return c.Send(card, markup)
}

func (h *Handler) notifyHostJoinRequest(c tele.Context, event *database.Event) {
	requester, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		slog.Error("notify host: get requester", "error", err)
		return
	}

	name := c.Sender().FirstName
	username := c.Sender().Username
	age := 0
	if requester.Age != nil {
		age = *requester.Age
	}
	city, state := "", ""
	if requester.City != nil {
		city = *requester.City
	}
	if requester.State != nil {
		state = *requester.State
	}

	text := messages.JoinRequest(event.Title, c.Sender().ID, name, username, age, city, state)

	markup := &tele.ReplyMarkup{}
	btnApprove := markup.Data("\u2705 Принять", "ap", fmt.Sprintf("%s:%d", event.ID, c.Sender().ID))
	btnReject := markup.Data("\u274c Отклонить", "rj", fmt.Sprintf("%s:%d", event.ID, c.Sender().ID))
	markup.Inline(markup.Row(btnApprove, btnReject))

	host := &tele.User{ID: event.HostTelegramID}
	if _, err := h.bot.Send(host, text, markup, tele.ModeHTML); err != nil {
		slog.Error("notify host: send", "error", err)
	}
}
