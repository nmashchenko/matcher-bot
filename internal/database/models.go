package database

import (
	"time"

	"github.com/uptrace/bun"
)

type UserState string

const (
	StateUnverified UserState = "unverified"
	StateOnboarding UserState = "onboarding"
	StateReady      UserState = "ready"
)

func (s UserState) String() string { return string(s) }

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           string    `bun:",pk,type:uuid,default:gen_random_uuid()"`
	TelegramID   int64     `bun:",unique,notnull"`
	Username     *string   `bun:",nullzero"`
	FirstName    *string   `bun:",nullzero"`
	LastName     *string   `bun:",nullzero"`
	UserState    UserState `bun:"column:user_state,notnull,default:'unverified'"`
	Latitude     *float64  `bun:",nullzero"`
	Longitude    *float64  `bun:",nullzero"`
	Country      *string   `bun:",nullzero"`
	State        *string   `bun:",nullzero"`
	City         *string   `bun:",nullzero"`
	AvatarFileID       *string   `bun:",nullzero"`
	Age                *int      `bun:",nullzero"`
	PreferredEventType *string   `bun:"preferred_event_type,nullzero"`
	VerifiedAt         *time.Time `bun:",nullzero"`
	CreatedAt    time.Time  `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time  `bun:",nullzero,notnull,default:current_timestamp"`
}

type EventType string

const (
	EventHangout EventType = "hangout"
	EventGaming  EventType = "gaming"
	EventSports  EventType = "sports"
	EventConcert EventType = "concert"
	EventRandom  EventType = "random"
)

func (e EventType) String() string { return string(e) }

type EventState string

const (
	EventActive    EventState = "active"
	EventCancelled EventState = "cancelled"
	EventExpired   EventState = "expired"
)

func (e EventState) String() string { return string(e) }

type Event struct {
	bun.BaseModel `bun:"table:events,alias:e"`

	ID              string     `bun:",pk,type:uuid,default:gen_random_uuid()"`
	HostTelegramID  int64      `bun:",notnull"`
	Title           string     `bun:",notnull"`
	Description     *string    `bun:",nullzero"`
	EventType       EventType  `bun:"column:event_type,notnull"`
	EventState      EventState `bun:"column:event_state,notnull,default:'active'"`
	Latitude        float64    `bun:",notnull"`
	Longitude       float64    `bun:",notnull"`
	City            string     `bun:",notnull"`
	State           string     `bun:",notnull"`
	MaxParticipants int        `bun:",notnull,default:10"`
	MinAge          *int       `bun:",nullzero"`
	MaxAge          *int       `bun:",nullzero"`
	StartsAt        time.Time  `bun:",notnull"`
	CreatedAt       time.Time  `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt       time.Time  `bun:",nullzero,notnull,default:current_timestamp"`
}

type ParticipantStatus string

const (
	StatusPending  ParticipantStatus = "pending"
	StatusApproved ParticipantStatus = "approved"
	StatusRejected ParticipantStatus = "rejected"
	StatusRemoved  ParticipantStatus = "removed"
)

func (p ParticipantStatus) String() string { return string(p) }

type EventParticipant struct {
	bun.BaseModel `bun:"table:event_participants,alias:ep"`

	ID          string            `bun:",pk,type:uuid,default:gen_random_uuid()"`
	EventID     string            `bun:",notnull,type:uuid"`
	TelegramID  int64             `bun:",notnull"`
	Status      ParticipantStatus `bun:",notnull,default:'pending'"`
	RequestedAt time.Time         `bun:",notnull,default:current_timestamp"`
	RespondedAt *time.Time        `bun:",nullzero"`
}
