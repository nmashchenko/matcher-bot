package events

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"matcher-bot/internal/database"
	"matcher-bot/internal/messages"

	tele "gopkg.in/telebot.v4"
)

func (h *Handler) onApprove(c tele.Context) error {
	eventID, telegramID, err := parseEventUser(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	event, err := h.events.GetByID(context.Background(), eventID)
	if err != nil {
		slog.Error("approve: get event", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	approved, err := h.events.CountApproved(context.Background(), eventID)
	if err != nil {
		slog.Error("approve: count", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}
	if approved >= event.MaxParticipants {
		return c.Respond(&tele.CallbackResponse{Text: messages.EventFull})
	}

	if err := h.events.UpdateParticipantStatus(context.Background(), eventID, telegramID, database.StatusApproved); err != nil {
		slog.Error("approve: update", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	_ = c.Respond(&tele.CallbackResponse{Text: messages.Approved})
	_ = c.Delete()

	// Confirm to host with participant name.
	u, _ := h.users.GetByTelegramID(context.Background(), telegramID)
	name := "Участник"
	if u != nil && u.FirstName != nil {
		name = *u.FirstName
	}
	_ = c.Send(messages.HostApprovedConfirm(name, event.Title))

	h.notifyApproved(event, telegramID)

	return nil
}

func (h *Handler) onReject(c tele.Context) error {
	eventID, telegramID, err := parseEventUser(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	if err := h.events.UpdateParticipantStatus(context.Background(), eventID, telegramID, database.StatusRejected); err != nil {
		slog.Error("reject: update", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	_ = c.Respond(&tele.CallbackResponse{Text: messages.Rejected})
	_ = c.Delete()

	event, err := h.events.GetByID(context.Background(), eventID)
	if err != nil {
		return nil
	}
	user := &tele.User{ID: telegramID}
	if _, err := h.bot.Send(user, messages.ParticipantRejected(event.Title)); err != nil {
		slog.Error("reject: notify", "error", err)
	}

	return nil
}

func (h *Handler) onCancelEvent(c tele.Context) error {
	eventID := c.Callback().Data

	event, err := h.events.GetByID(context.Background(), eventID)
	if err != nil {
		slog.Error("cancel: get event", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	if event.HostTelegramID != c.Sender().ID {
		return c.Respond(&tele.CallbackResponse{Text: messages.NotHost})
	}

	if err := h.events.UpdateState(context.Background(), eventID, database.EventCancelled); err != nil {
		slog.Error("cancel: update", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	_ = c.Respond()
	_ = c.Delete()

	_ = c.Send(messages.EventCancelledConfirm(event.Title))

	h.notifyEventCancelled(event)

	return nil
}

func (h *Handler) onRemoveParticipant(c tele.Context) error {
	eventID, telegramID, err := parseEventUser(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	event, err := h.events.GetByID(context.Background(), eventID)
	if err != nil {
		slog.Error("remove: get event", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	if event.HostTelegramID != c.Sender().ID {
		return c.Respond(&tele.CallbackResponse{Text: messages.NotHost})
	}

	participant, err := h.events.GetParticipant(context.Background(), eventID, telegramID)
	if err != nil || (participant.Status != database.StatusApproved && participant.Status != database.StatusPending) {
		return c.Respond(&tele.CallbackResponse{Text: messages.AlreadyRemoved})
	}

	if err := h.events.UpdateParticipantStatus(context.Background(), eventID, telegramID, database.StatusRemoved); err != nil {
		slog.Error("remove: update", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	_ = c.Respond()

	// Update the manage card in-place.
	text, markup := h.buildManageCard(event)
	_ = c.Edit(text, markup)

	user := &tele.User{ID: telegramID}
	if _, err := h.bot.Send(user, messages.ParticipantRemoved(event.Title)); err != nil {
		slog.Error("remove: notify", "error", err)
	}

	return nil
}

func (h *Handler) cmdMy(c tele.Context) error {
	text, markup, err := h.buildMyEventsList(c.Sender().ID)
	if err != nil {
		return c.Send(messages.GenericError)
	}
	if markup != nil {
		return c.Send(text, markup)
	}
	return c.Send(text)
}

func (h *Handler) onBackToMyEvents(c tele.Context) error {
	_ = c.Respond()
	text, markup, err := h.buildMyEventsList(c.Sender().ID)
	if err != nil {
		return c.Edit(messages.GenericError)
	}
	if markup != nil {
		return c.Edit(text, markup)
	}
	return c.Edit(text)
}

func (h *Handler) buildMyEventsList(telegramID int64) (string, *tele.ReplyMarkup, error) {
	ctx := context.Background()

	hosted, err := h.events.ListByHost(ctx, telegramID)
	if err != nil {
		slog.Error("my: list hosted", "error", err)
		return "", nil, err
	}

	joinedApproved, err := h.events.ListJoinedByStatus(ctx, telegramID, database.StatusApproved)
	if err != nil {
		slog.Error("my: list joined approved", "error", err)
		return "", nil, err
	}
	joinedPending, err := h.events.ListJoinedByStatus(ctx, telegramID, database.StatusPending)
	if err != nil {
		slog.Error("my: list joined pending", "error", err)
		return "", nil, err
	}

	if len(hosted) == 0 && len(joinedApproved) == 0 && len(joinedPending) == 0 {
		return messages.MyEventsEmpty, nil, nil
	}

	var sb strings.Builder

	if len(hosted) > 0 {
		sb.WriteString("\U0001f451 Мои события:\n\n")
		for _, ev := range hosted {
			emoji := EventTypeEmoji(ev.EventType)
			approved, _ := h.events.CountApproved(ctx, ev.ID)
			pendingStatus := database.StatusPending
			pendingList, _ := h.events.ListParticipants(ctx, ev.ID, &pendingStatus)

			sb.WriteString(messages.MyEventRow(emoji, ev.Title, ev.StartsAt, approved, ev.MaxParticipants, len(pendingList)))
			sb.WriteString("\n")
		}
	}

	if len(joinedApproved) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("✅ Я участвую:\n\n")
		for _, ev := range joinedApproved {
			emoji := EventTypeEmoji(ev.EventType)
			approved, _ := h.events.CountApproved(ctx, ev.ID)
			sb.WriteString(messages.MyEventRow(emoji, ev.Title, ev.StartsAt, approved, ev.MaxParticipants, 0))
			sb.WriteString("\n")
		}
	}

	if len(joinedPending) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("⏳ Ожидают подтверждения:\n\n")
		for _, ev := range joinedPending {
			emoji := EventTypeEmoji(ev.EventType)
			approved, _ := h.events.CountApproved(ctx, ev.ID)
			sb.WriteString(messages.MyEventRow(emoji, ev.Title, ev.StartsAt, approved, ev.MaxParticipants, 0))
			sb.WriteString("\n")
		}
	}

	allJoined := append(joinedApproved, joinedPending...)
	if len(hosted) > 0 || len(allJoined) > 0 {
		markup := &tele.ReplyMarkup{}
		var rows []tele.Row
		for _, ev := range hosted {
			emoji := EventTypeEmoji(ev.EventType)
			btn := markup.Data(
				fmt.Sprintf("%s %s", emoji, ev.Title),
				"me", ev.ID,
			)
			rows = append(rows, markup.Row(btn))
		}
		for _, ev := range allJoined {
			emoji := EventTypeEmoji(ev.EventType)
			btn := markup.Data(
				fmt.Sprintf("%s %s", emoji, ev.Title),
				"ve", ev.ID,
			)
			rows = append(rows, markup.Row(btn))
		}
		markup.Inline(rows...)
		return sb.String(), markup, nil
	}

	return sb.String(), nil, nil
}

func (h *Handler) onManageEvent(c tele.Context) error {
	eventID := c.Callback().Data

	event, err := h.events.GetByID(context.Background(), eventID)
	if err != nil {
		slog.Error("manage: get event", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	if event.EventState != database.EventActive {
		return c.Respond(&tele.CallbackResponse{Text: messages.EventNoLongerActive})
	}

	if event.HostTelegramID != c.Sender().ID {
		return c.Respond(&tele.CallbackResponse{Text: messages.NotHost})
	}

	_ = c.Respond()

	text, markup := h.buildManageCard(event)
	return c.Edit(text, markup)
}

// buildManageCard builds the event management card text and inline markup.
func (h *Handler) buildManageCard(event *database.Event) (string, *tele.ReplyMarkup) {
	ctx := context.Background()
	eventID := event.ID

	approvedStatus := database.StatusApproved
	approvedList, _ := h.events.ListParticipants(ctx, eventID, &approvedStatus)
	pendingStatus := database.StatusPending
	pendingList, _ := h.events.ListParticipants(ctx, eventID, &pendingStatus)

	emoji := EventTypeEmoji(event.EventType)
	label := EventTypeLabel(event.EventType)
	desc := ""
	if event.Description != nil {
		desc = *event.Description
	}
	approved, _ := h.events.CountApproved(ctx, eventID)

	var sb strings.Builder
	sb.WriteString(messages.EventCard(emoji, label, event.Title, desc, event.City, event.StartsAt, approved, event.MaxParticipants))

	if len(approvedList) > 0 {
		sb.WriteString("\n\n\u2705 Участники:\n")
		for _, p := range approvedList {
			u, err := h.users.GetByTelegramID(ctx, p.TelegramID)
			if err != nil {
				continue
			}
			name := "?"
			if u.FirstName != nil {
				name = *u.FirstName
			}
			sb.WriteString(fmt.Sprintf("  - %s", name))
			if u.Username != nil {
				sb.WriteString(fmt.Sprintf(" (@%s)", *u.Username))
			}
			sb.WriteString("\n")
		}
	}

	if len(pendingList) > 0 {
		sb.WriteString(fmt.Sprintf("\n\u23f3 Ожидают: %d", len(pendingList)))
	}

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	for _, p := range approvedList {
		u, err := h.users.GetByTelegramID(ctx, p.TelegramID)
		if err != nil {
			continue
		}
		name := "?"
		if u.FirstName != nil {
			name = *u.FirstName
		}
		btn := markup.Data(
			fmt.Sprintf("\u274c Убрать %s", name),
			"rm", fmt.Sprintf("%s:%d", eventID, p.TelegramID),
		)
		rows = append(rows, markup.Row(btn))
	}

	btnCancel := markup.Data("\U0001f6ab Отменить событие", "cn", eventID)
	btnBack := markup.Data("◀️ Назад", "bk")
	rows = append(rows, markup.Row(btnCancel))
	rows = append(rows, markup.Row(btnBack))

	markup.Inline(rows...)

	return sb.String(), markup
}

func (h *Handler) notifyApproved(event *database.Event, telegramID int64) {
	host, err := h.users.GetByTelegramID(context.Background(), event.HostTelegramID)
	if err != nil {
		slog.Error("notify approved: get host", "error", err)
		return
	}
	hostName := "Организатор"
	hostUsername := ""
	if host.FirstName != nil {
		hostName = *host.FirstName
	}
	if host.Username != nil {
		hostUsername = *host.Username
	}

	approvedStatus := database.StatusApproved
	participants, _ := h.events.ListParticipants(context.Background(), event.ID, &approvedStatus)
	var names []string
	for _, p := range participants {
		if p.TelegramID == telegramID {
			continue
		}
		u, err := h.users.GetByTelegramID(context.Background(), p.TelegramID)
		if err != nil {
			continue
		}
		name := "?"
		if u.FirstName != nil {
			name = *u.FirstName
		}
		names = append(names, name)
	}

	text := messages.ParticipantApproved(event.Title, hostName, hostUsername, event.City, names)

	user := &tele.User{ID: telegramID}
	if _, err := h.bot.Send(user, text); err != nil {
		slog.Error("notify approved: send text", "error", err)
		return
	}

	venue := &tele.Location{
		Lat: float32(event.Latitude),
		Lng: float32(event.Longitude),
	}
	if _, err := h.bot.Send(user, venue); err != nil {
		slog.Error("notify approved: send location", "error", err)
	}
}

func (h *Handler) notifyEventCancelled(event *database.Event) {
	participants, err := h.events.ListParticipants(context.Background(), event.ID, nil)
	if err != nil {
		slog.Error("cancel notify: list", "error", err)
		return
	}
	for _, p := range participants {
		if p.Status != database.StatusPending && p.Status != database.StatusApproved {
			continue
		}
		user := &tele.User{ID: p.TelegramID}
		if _, err := h.bot.Send(user, messages.EventCancelled(event.Title)); err != nil {
			slog.Error("cancel notify: send", "error", err, "telegram_id", p.TelegramID)
		}
	}
}

func (h *Handler) onViewJoinedEvent(c tele.Context) error {
	eventID := c.Callback().Data

	event, err := h.events.GetByID(context.Background(), eventID)
	if err != nil {
		slog.Error("view joined: get event", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	if event.EventState != database.EventActive {
		return c.Respond(&tele.CallbackResponse{Text: messages.EventNoLongerActive})
	}

	participant, err := h.events.GetParticipant(context.Background(), eventID, c.Sender().ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Respond(&tele.CallbackResponse{Text: messages.NotParticipant})
		}
		slog.Error("view joined: get participant", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}
	if participant.Status != database.StatusPending && participant.Status != database.StatusApproved {
		return c.Respond(&tele.CallbackResponse{Text: messages.NotParticipant})
	}

	_ = c.Respond()

	emoji := EventTypeEmoji(event.EventType)
	label := EventTypeLabel(event.EventType)
	desc := ""
	if event.Description != nil {
		desc = *event.Description
	}
	approved, _ := h.events.CountApproved(context.Background(), eventID)

	statusText := "⏳ Ожидает подтверждения"
	if participant.Status == database.StatusApproved {
		statusText = "✅ Подтверждён"
	}

	card := messages.EventCard(emoji, label, event.Title, desc, event.City, event.StartsAt, approved, event.MaxParticipants)
	text := fmt.Sprintf("%s\n\nТвой статус: %s", card, statusText)

	markup := &tele.ReplyMarkup{}
	btnLeave := markup.Data("❌ Покинуть событие", "le", eventID)
	btnBack := markup.Data("◀️ Назад", "bk")
	markup.Inline(markup.Row(btnLeave), markup.Row(btnBack))

	return c.Edit(text, markup)
}

func (h *Handler) onLeaveEvent(c tele.Context) error {
	eventID := c.Callback().Data

	event, err := h.events.GetByID(context.Background(), eventID)
	if err != nil {
		slog.Error("leave: get event", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	participant, err := h.events.GetParticipant(context.Background(), eventID, c.Sender().ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Respond(&tele.CallbackResponse{Text: messages.NotParticipant})
		}
		slog.Error("leave: get participant", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}
	if participant.Status != database.StatusPending && participant.Status != database.StatusApproved {
		return c.Respond(&tele.CallbackResponse{Text: messages.NotParticipant})
	}

	if !event.StartsAt.After(time.Now()) {
		return c.Respond(&tele.CallbackResponse{Text: messages.EventAlreadyStarted})
	}

	if err := h.events.UpdateParticipantStatus(context.Background(), eventID, c.Sender().ID, database.StatusRemoved); err != nil {
		slog.Error("leave: update", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	_ = c.Respond()
	_ = c.Delete()

	_ = c.Send(messages.ParticipantLeft)

	// Notify host
	name := c.Sender().FirstName
	host := &tele.User{ID: event.HostTelegramID}
	if _, err := h.bot.Send(host, messages.HostNotifyLeft(event.Title, name)); err != nil {
		slog.Error("leave: notify host", "error", err)
	}

	return nil
}

// parseEventUser parses "eventID:telegramID" from callback data.
func parseEventUser(data string) (string, int64, error) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid callback data: %s", data)
	}
	tgID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid telegram ID: %s", parts[1])
	}
	return parts[0], tgID, nil
}
