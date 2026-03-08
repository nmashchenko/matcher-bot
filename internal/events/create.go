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

func (h *Handler) cmdCreate(c tele.Context) error {
	createSessions.Delete(c.Sender().ID)

	sess := &createSession{Step: stepType}
	createSessions.Store(c.Sender().ID, sess)

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

	return c.Send(messages.CreateStart, markup)
}

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
	return c.Send(messages.CreateAskTitle)
}

func (h *Handler) handleCreateText(c tele.Context) error {
	val, ok := createSessions.Load(c.Sender().ID)
	if !ok {
		return nil
	}
	sess := val.(*createSession)
	text := strings.TrimSpace(c.Text())

	switch sess.Step {
	case stepTitle:
		if text == "" {
			return c.Send(messages.CreateAskTitle)
		}
		sess.Title = text
		sess.Step = stepDesc
		return c.Send(messages.CreateAskDesc)

	case stepDesc:
		if text == "-" {
			sess.Desc = ""
		} else {
			sess.Desc = text
		}
		sess.Step = stepTime
		return c.Send(messages.CreateAskTime)

	case stepTime:
		t, err := parseEventTime(text)
		if err != nil {
			return c.Send(messages.InvalidTime)
		}
		if t.Before(time.Now()) {
			return c.Send(messages.TimePast)
		}
		sess.StartsAt = t
		sess.Step = stepLocation
		return c.Send(messages.CreateAskLocation)

	case stepCapacity:
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > 50 {
			return c.Send(messages.InvalidNumber)
		}
		sess.Capacity = n
		sess.Step = stepConfirm

		emoji := EventTypeEmoji(sess.EventType)
		label := EventTypeLabel(sess.EventType)

		markup := &tele.ReplyMarkup{}
		btnOk := markup.Data("\u2705 Создать", "cc", "ok")
		btnNo := markup.Data("\u274c Отмена", "cc", "no")
		markup.Inline(markup.Row(btnOk, btnNo))

		return c.Send(
			messages.CreateConfirm(emoji, label, sess.Title, sess.Desc, sess.City, sess.StartsAt, sess.Capacity),
			markup,
		)

	default:
		return nil
	}
}

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
	return c.Send(messages.CreateAskCapacity)
}


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
		return c.Send(messages.CreateCancelled)
	}

	event := &database.Event{
		HostTelegramID:  c.Sender().ID,
		Title:           sess.Title,
		Description:     strPtr(sess.Desc),
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
	return c.Send(messages.CreateSuccess(sess.Title, sess.City, sess.StartsAt))
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

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
