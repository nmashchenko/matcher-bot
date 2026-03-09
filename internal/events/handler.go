package events

import (
	"matcher-bot/internal/database"
	"matcher-bot/internal/geocoding"
	"matcher-bot/internal/messages"

	tele "gopkg.in/telebot.v4"
)

type Handler struct {
	users  database.UserRepository
	events database.EventRepository
	geo    *geocoding.Geocoder
	bot    *tele.Bot
}

func NewHandler(users database.UserRepository, events database.EventRepository, geo *geocoding.Geocoder, bot *tele.Bot) *Handler {
	return &Handler{
		users:  users,
		events: events,
		geo:    geo,
		bot:    bot,
	}
}

func (h *Handler) Register(b *tele.Bot) {
	b.Handle("/create", h.cmdCreate)
	b.Handle("/events", h.blockIfCreating(h.cmdBrowse))
	b.Handle("/myevents", h.blockIfCreating(h.cmdMy))
	// Event type select
	b.Handle("\fet", h.onEventTypeSelect)

	// Create confirm / cancel
	b.Handle("\fcc", h.onCreateConfirm)

	// Browse next / join
	b.Handle("\fbn", h.onBrowseNext)
	b.Handle("\fbj", h.onBrowseJoin)

	// Approve / reject
	b.Handle("\fap", h.onApprove)
	b.Handle("\frj", h.onReject)

	// Cancel event
	b.Handle("\fcn", h.onCancelEvent)

	// Remove participant
	b.Handle("\frm", h.onRemoveParticipant)

	// Manage event (from /my)
	b.Handle("\fme", h.onManageEvent)

	// View joined event / leave event
	b.Handle("\fve", h.onViewJoinedEvent)
	b.Handle("\fle", h.onLeaveEvent)

	// Back to /myevents list
	b.Handle("\fbk", h.onBackToMyEvents)
}

// blockIfCreating wraps a handler to block it while user is in event creation mode.
func (h *Handler) blockIfCreating(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if h.IsCreating(c.Sender().ID) {
			return c.Send(messages.CreateInProgress)
		}
		return next(c)
	}
}

// IsCreating returns true if the user has an active create session.
func (h *Handler) IsCreating(telegramID int64) bool {
	_, ok := createSessions.Load(telegramID)
	return ok
}

// OnCreateText handles text input during event creation.
func (h *Handler) OnCreateText(c tele.Context) error {
	return h.handleCreateText(c)
}

// OnCreateLocation handles location input during event creation.
func (h *Handler) OnCreateLocation(c tele.Context) error {
	return h.handleCreateLocation(c)
}
