package events

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"matcher-bot/internal/database"
	"matcher-bot/internal/messages"
	"matcher-bot/internal/ptr"

	tele "gopkg.in/telebot.v4"
)

type createStep int

const (
	stepType createStep = iota
	stepTitle
	stepDesc
	stepTime
	stepLocation
	stepCapacity
	stepConfirm
)

type createSession struct {
	Step      createStep
	EventType database.EventType
	Title     string
	Desc      string
	StartsAt  time.Time
	Lat       float64
	Lon       float64
	City      string
	State     string
	Capacity  int
}

var createSessions sync.Map // map[int64]*createSession

// cancelKeyboard returns a reply keyboard with a cancel button.
func cancelKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	markup.Reply(markup.Row(markup.Text(messages.CreateCancelBtn)))
	return markup
}

// removeKeyboard removes the reply keyboard.
func removeKeyboard() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{RemoveKeyboard: true}
}

// cmdCreate starts the event creation wizard.
func (h *Handler) cmdCreate(c tele.Context) error {
	createSessions.Delete(c.Sender().ID)

	sess := &createSession{Step: stepType}
	createSessions.Store(c.Sender().ID, sess)

	// Show cancel reply keyboard first (persists across messages).
	_ = c.Send(messages.CreateStart, cancelKeyboard())

	// Inline buttons for event type selection.
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, opt := range EventTypeOptions {
		btn := markup.Data(
			fmt.Sprintf("%s %s", opt.Emoji, opt.Label),
			"et", string(opt.Key),
		)
		rows = append(rows, markup.Row(btn))
	}
	markup.Inline(rows...)

	return c.Send(messages.CreatePickType, markup)
}

// onEventTypeSelect handles the inline event type callback and advances to the title step.
func (h *Handler) onEventTypeSelect(c tele.Context) error {
	val, ok := createSessions.Load(c.Sender().ID)
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: messages.SessionExpired})
	}
	sess := val.(*createSession)
	if sess.Step != stepType {
		return c.Respond()
	}

	et := database.EventType(c.Callback().Data)
	if !ValidEventType(et) {
		return c.Respond(&tele.CallbackResponse{Text: messages.UnknownEventType})
	}

	sess.EventType = et
	sess.Step = stepTitle

	_ = c.Respond()
	_ = c.Delete()
	return c.Send(messages.CreateAskTitle, cancelKeyboard())
}

// handleCreateText dispatches text input to the appropriate creation step.
func (h *Handler) handleCreateText(c tele.Context) error {
	val, ok := createSessions.Load(c.Sender().ID)
	if !ok {
		return nil
	}
	sess := val.(*createSession)
	text := strings.TrimSpace(c.Text())

	if text == messages.CreateCancelBtn {
		createSessions.Delete(c.Sender().ID)
		return c.Send(messages.CreateCancelled, removeKeyboard())
	}

	switch sess.Step {
	case stepTitle:
		return h.onStepTitle(c, sess, text)
	case stepDesc:
		return h.onStepDesc(c, sess, text)
	case stepTime:
		return h.onStepTime(c, sess, text)
	case stepCapacity:
		return h.onStepCapacity(c, sess, text)
	default:
		return nil
	}
}

func (h *Handler) onStepTitle(c tele.Context, sess *createSession, text string) error {
	if text == "" {
		return c.Send(messages.CreateAskTitle, cancelKeyboard())
	}
	sess.Title = text
	sess.Step = stepDesc
	return c.Send(messages.CreateAskDesc, cancelKeyboard())
}

func (h *Handler) onStepDesc(c tele.Context, sess *createSession, text string) error {
	if text == "-" {
		sess.Desc = ""
	} else {
		sess.Desc = text
	}
	sess.Step = stepTime
	return c.Send(messages.CreateAskTime, cancelKeyboard())
}

func (h *Handler) onStepTime(c tele.Context, sess *createSession, text string) error {
	t, err := parseEventTime(text)
	if err != nil {
		return c.Send(messages.InvalidTime, cancelKeyboard())
	}
	if t.Before(time.Now()) {
		return c.Send(messages.TimePast, cancelKeyboard())
	}
	sess.StartsAt = t
	sess.Step = stepLocation
	return c.Send(messages.CreateAskLocation, cancelKeyboard())
}

func (h *Handler) onStepCapacity(c tele.Context, sess *createSession, text string) error {
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > 50 {
		return c.Send(messages.InvalidNumber, cancelKeyboard())
	}
	sess.Capacity = n
	sess.Step = stepConfirm

	emoji := EventTypeEmoji(sess.EventType)
	label := EventTypeLabel(sess.EventType)

	markup := &tele.ReplyMarkup{}
	btnOk := markup.Data("\u2705 Создать", "cc", "ok")
	btnNo := markup.Data("\u274c Отмена", "cc", "no")
	markup.Inline(markup.Row(btnOk, btnNo))

	// Send a blank message to remove the reply keyboard, then show confirm with inline buttons.
	_ = c.Send("\u2705 Отлично! Проверь данные:", removeKeyboard())
	return c.Send(
		messages.CreateConfirm(emoji, label, sess.Title, sess.Desc, sess.City, sess.StartsAt, sess.Capacity),
		markup,
	)
}

// handleCreateLocation processes the shared location and advances to the capacity step.
func (h *Handler) handleCreateLocation(c tele.Context) error {
	val, ok := createSessions.Load(c.Sender().ID)
	if !ok {
		return nil
	}
	sess := val.(*createSession)
	if sess.Step != stepLocation {
		return nil
	}

	loc := c.Message().Location
	if loc == nil {
		return nil
	}

	geoResult, err := h.geo.ReverseGeocode(context.Background(), float64(loc.Lat), float64(loc.Lng))
	if err != nil || geoResult.City == "" {
		slog.Error("create event geocode", "error", err)
		return c.Send(messages.GeocodingError)
	}

	sess.Lat = float64(loc.Lat)
	sess.Lon = float64(loc.Lng)
	sess.City = geoResult.City
	sess.State = geoResult.State
	sess.Step = stepCapacity
	return c.Send(messages.CreateAskCapacity, cancelKeyboard())
}

// onCreateConfirm handles the final confirm/cancel callback and persists the event.
func (h *Handler) onCreateConfirm(c tele.Context) error {
	val, ok := createSessions.Load(c.Sender().ID)
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: messages.SessionExpired})
	}
	sess := val.(*createSession)

	action := c.Callback().Data
	_ = c.Respond()
	_ = c.Delete()

	if action != "ok" {
		createSessions.Delete(c.Sender().ID)
		return c.Send(messages.CreateCancelled, removeKeyboard())
	}

	event := &database.Event{
		HostTelegramID:  c.Sender().ID,
		Title:           sess.Title,
		Description:     ptr.Str(sess.Desc),
		EventType:       sess.EventType,
		Latitude:        sess.Lat,
		Longitude:       sess.Lon,
		City:            sess.City,
		State:           sess.State,
		MaxParticipants: sess.Capacity,
		StartsAt:        sess.StartsAt,
	}
	if err := h.events.Create(context.Background(), event); err != nil {
		slog.Error("create event", "error", err)
		createSessions.Delete(c.Sender().ID)
		return c.Send(messages.GenericError)
	}

	createSessions.Delete(c.Sender().ID)
	return c.Send(messages.CreateSuccess(sess.Title, sess.City, sess.StartsAt), removeKeyboard())
}

// parseEventTime parses "DD.MM HH:MM" into a time.Time for the current year.
func parseEventTime(text string) (time.Time, error) {
	text = strings.TrimSpace(text)
	now := time.Now()

	t, err := time.Parse("02.01 15:04", text)
	if err != nil {
		return time.Time{}, err
	}

	result := time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())

	// If the date has passed this year, assume next year.
	if result.Before(now) {
		result = result.AddDate(1, 0, 0)
	}

	return result, nil
}

