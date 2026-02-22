package database

import (
	"time"

	"github.com/uptrace/bun"
)

type VerificationStatus string

const (
	StatusPending    VerificationStatus = "PENDING"
	StatusVerified   VerificationStatus = "VERIFIED"
	StatusUnverified VerificationStatus = "UNVERIFIED"
	StatusRejected   VerificationStatus = "REJECTED"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID                 string             `bun:",pk,type:uuid,default:gen_random_uuid()"`
	TelegramID         int64              `bun:",unique,notnull"`
	Username           *string            `bun:",nullzero"`
	FirstName          *string            `bun:",nullzero"`
	LastName           *string            `bun:",nullzero"`
	VerificationStatus VerificationStatus `bun:",notnull,default:'PENDING'"`
	Latitude           *float64           `bun:",nullzero"`
	Longitude          *float64           `bun:",nullzero"`
	Country            *string            `bun:",nullzero"`
	State              *string            `bun:",nullzero"`
	City               *string            `bun:",nullzero"`
	VerifiedAt         *time.Time         `bun:",nullzero"`
	CreatedAt          time.Time          `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt          time.Time          `bun:",nullzero,notnull,default:current_timestamp"`
}
