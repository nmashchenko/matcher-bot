package database

import (
	"time"

	pgvector "github.com/pgvector/pgvector-go"
	"github.com/uptrace/bun"
)

type UserState string

const (
	StateUnverified UserState = "unverified"
	StateOnboarding UserState = "onboarding"
	StateReady      UserState = "ready"
)

func (s UserState) String() string { return string(s) }

type Goal string

const (
	GoalFriends  Goal = "friends"
	GoalHangouts Goal = "hangouts"
	GoalDating   Goal = "dating"
	GoalMixed    Goal = "mixed"
)

func (g Goal) String() string { return string(g) }

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID                  string          `bun:",pk,type:uuid,default:gen_random_uuid()"`
	TelegramID          int64           `bun:",unique,notnull"`
	Username            *string         `bun:",nullzero"`
	FirstName           *string         `bun:",nullzero"`
	LastName            *string         `bun:",nullzero"`
	UserState           UserState       `bun:"column:user_state,notnull,default:'unverified'"`
	Latitude            *float64        `bun:",nullzero"`
	Longitude           *float64        `bun:",nullzero"`
	Country             *string         `bun:",nullzero"`
	State               *string         `bun:",nullzero"`
	City                *string         `bun:",nullzero"`
	AvatarFileID        *string         `bun:",nullzero"`
	Age                 *int            `bun:",nullzero"`
	Goal                *Goal           `bun:",nullzero"`
	Bio                 *string         `bun:",nullzero"`
	LookingFor          *string         `bun:",nullzero,column:looking_for"`
	BioEmbedding        *pgvector.Vector `bun:"type:vector(1536),nullzero"`
	LookingForEmbedding *pgvector.Vector `bun:"type:vector(1536),nullzero,column:looking_for_embedding"`
	VerifiedAt          *time.Time      `bun:",nullzero"`
	CreatedAt           time.Time       `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt           time.Time       `bun:",nullzero,notnull,default:current_timestamp"`
}
