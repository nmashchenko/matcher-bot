package events

import (
	"fmt"

	"matcher-bot/internal/database"
)

type EventTypeOption struct {
	Key   database.EventType
	Label string
	Emoji string
}

var EventTypeOptions = []EventTypeOption{
	{Key: database.EventHangout, Label: "Тусовка", Emoji: "\U0001f919"},
	{Key: database.EventGaming, Label: "Игры", Emoji: "\U0001f3ae"},
	{Key: database.EventSports, Label: "Спорт", Emoji: "\u26bd"},
	{Key: database.EventConcert, Label: "Концерт / Шоу", Emoji: "\U0001f3b5"},
	{Key: database.EventRandom, Label: "Рандом", Emoji: "\U0001f3b2"},
}

func ValidEventType(t database.EventType) bool {
	for _, opt := range EventTypeOptions {
		if opt.Key == t {
			return true
		}
	}
	return false
}

func EventTypeLabel(t database.EventType) string {
	for _, opt := range EventTypeOptions {
		if opt.Key == t {
			return opt.Label
		}
	}
	return string(t)
}

func EventTypeEmoji(t database.EventType) string {
	for _, opt := range EventTypeOptions {
		if opt.Key == t {
			return opt.Emoji
		}
	}
	return "\U0001f4c5"
}

func FormatAgeRestriction(minAge, maxAge *int) string {
	if minAge == nil && maxAge == nil {
		return ""
	}
	if minAge != nil && maxAge != nil {
		return fmt.Sprintf("%d-%d", *minAge, *maxAge)
	}
	if minAge != nil {
		return fmt.Sprintf("от %d", *minAge)
	}
	return fmt.Sprintf("до %d", *maxAge)
}
