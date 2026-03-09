package database

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	UpdateState(ctx context.Context, id string, state EventState) error
	NextUnseen(ctx context.Context, city, state string, telegramID int64, eventType *EventType) (*Event, error)
	MarkViewed(ctx context.Context, telegramID int64, eventID string) error
	ListByHost(ctx context.Context, hostTelegramID int64) ([]*Event, error)
	ListJoined(ctx context.Context, telegramID int64) ([]*Event, error)
	ListJoinedByStatus(ctx context.Context, telegramID int64, status ParticipantStatus) ([]*Event, error)
	RequestJoin(ctx context.Context, eventID string, telegramID int64) error
	GetParticipant(ctx context.Context, eventID string, telegramID int64) (*EventParticipant, error)
	UpdateParticipantStatus(ctx context.Context, eventID string, telegramID int64, status ParticipantStatus) error
	ListParticipants(ctx context.Context, eventID string, status *ParticipantStatus) ([]*EventParticipant, error)
	CountApproved(ctx context.Context, eventID string) (int, error)
	ExpireEvents(ctx context.Context) (int, error)
}

type EventStore struct {
	db *bun.DB
}

func NewEventStore(db *bun.DB) *EventStore {
	return &EventStore{db: db}
}

func (s *EventStore) Create(ctx context.Context, event *Event) error {
	_, err := s.db.NewInsert().Model(event).Exec(ctx)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

func (s *EventStore) GetByID(ctx context.Context, id string) (*Event, error) {
	event := new(Event)
	err := s.db.NewSelect().
		Model(event).
		Where("e.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	return event, nil
}

func (s *EventStore) UpdateState(ctx context.Context, id string, state EventState) error {
	_, err := s.db.NewUpdate().
		TableExpr("events").
		Set("event_state = ?", string(state)).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// NextUnseen returns the next active event that the user hasn't viewed yet.
// If eventType is nil: shows all types, filtered by same state, same-city priority.
// If eventType is "gaming": global search (no city/state filter), filtered by type.
// If eventType is anything else: filtered by same city and event type.
func (s *EventStore) NextUnseen(ctx context.Context, city, state string, telegramID int64, eventType *EventType) (*Event, error) {
	event := new(Event)
	q := s.db.NewSelect().
		Model(event).
		Where("e.event_state = ?", string(EventActive)).
		Where("e.host_telegram_id != ?", telegramID).
		Where("e.starts_at > ?", time.Now()).
		Where("e.id NOT IN (?)",
			s.db.NewSelect().
				TableExpr("event_views").
				Column("event_id").
				Where("telegram_id = ?", telegramID),
		)

	if eventType != nil {
		q = q.Where("e.event_type = ?", string(*eventType))
		if *eventType != EventGaming {
			q = q.Where("e.city = ?", city)
		}
	} else {
		q = q.Where("e.state = ?", state)
	}

	q = q.OrderExpr("CASE WHEN e.city = ? THEN 0 ELSE 1 END", city).
		Order("e.starts_at ASC").
		Limit(1)

	err := q.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// MarkViewed records that a user has seen an event.
func (s *EventStore) MarkViewed(ctx context.Context, telegramID int64, eventID string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO event_views (telegram_id, event_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		telegramID, eventID,
	)
	return err
}

func (s *EventStore) ListByHost(ctx context.Context, hostTelegramID int64) ([]*Event, error) {
	var events []*Event
	err := s.db.NewSelect().
		Model(&events).
		Where("e.host_telegram_id = ?", hostTelegramID).
		Where("e.event_state = ?", string(EventActive)).
		Order("e.starts_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list host events: %w", err)
	}
	return events, nil
}

func (s *EventStore) ListJoined(ctx context.Context, telegramID int64) ([]*Event, error) {
	var events []*Event
	err := s.db.NewSelect().
		Model(&events).
		Join("JOIN event_participants AS ep ON ep.event_id = e.id").
		Where("ep.telegram_id = ?", telegramID).
		Where("ep.status IN (?)", bun.In([]string{string(StatusPending), string(StatusApproved)})).
		Where("e.event_state = ?", string(EventActive)).
		Order("e.starts_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list joined events: %w", err)
	}
	return events, nil
}

func (s *EventStore) ListJoinedByStatus(ctx context.Context, telegramID int64, status ParticipantStatus) ([]*Event, error) {
	var events []*Event
	err := s.db.NewSelect().
		Model(&events).
		Join("JOIN event_participants AS ep ON ep.event_id = e.id").
		Where("ep.telegram_id = ?", telegramID).
		Where("ep.status = ?", string(status)).
		Where("e.event_state = ?", string(EventActive)).
		Order("e.starts_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list joined events by status: %w", err)
	}
	return events, nil
}

func (s *EventStore) RequestJoin(ctx context.Context, eventID string, telegramID int64) error {
	ep := &EventParticipant{
		EventID:    eventID,
		TelegramID: telegramID,
	}
	_, err := s.db.NewInsert().Model(ep).Exec(ctx)
	if err != nil {
		return fmt.Errorf("request join: %w", err)
	}
	return nil
}

func (s *EventStore) GetParticipant(ctx context.Context, eventID string, telegramID int64) (*EventParticipant, error) {
	ep := new(EventParticipant)
	err := s.db.NewSelect().
		Model(ep).
		Where("ep.event_id = ?", eventID).
		Where("ep.telegram_id = ?", telegramID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return ep, nil
}

func (s *EventStore) UpdateParticipantStatus(ctx context.Context, eventID string, telegramID int64, status ParticipantStatus) error {
	now := time.Now()
	_, err := s.db.NewUpdate().
		TableExpr("event_participants").
		Set("status = ?", string(status)).
		Set("responded_at = ?", now).
		Where("event_id = ?", eventID).
		Where("telegram_id = ?", telegramID).
		Exec(ctx)
	return err
}

func (s *EventStore) ListParticipants(ctx context.Context, eventID string, status *ParticipantStatus) ([]*EventParticipant, error) {
	var participants []*EventParticipant
	q := s.db.NewSelect().
		Model(&participants).
		Where("ep.event_id = ?", eventID)
	if status != nil {
		q = q.Where("ep.status = ?", string(*status))
	}
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	return participants, nil
}

func (s *EventStore) CountApproved(ctx context.Context, eventID string) (int, error) {
	count, err := s.db.NewSelect().
		TableExpr("event_participants").
		Where("event_id = ?", eventID).
		Where("status = ?", string(StatusApproved)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count approved: %w", err)
	}
	return count, nil
}

// ExpireEvents marks active events with starts_at in the past as expired.
func (s *EventStore) ExpireEvents(ctx context.Context) (int, error) {
	res, err := s.db.NewUpdate().
		TableExpr("events").
		Set("event_state = ?", string(EventExpired)).
		Set("updated_at = ?", time.Now()).
		Where("event_state = ?", string(EventActive)).
		Where("starts_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("expire events: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
