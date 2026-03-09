package events

import (
	"testing"

	"matcher-bot/internal/database"
)

func TestValidEventType(t *testing.T) {
	known := []database.EventType{
		database.EventHangout, database.EventParty, database.EventGaming,
		database.EventDate, database.EventSports, database.EventConcert,
	}
	for _, et := range known {
		if !ValidEventType(et) {
			t.Errorf("ValidEventType(%q) = false, want true", et)
		}
	}

	if ValidEventType("nonexistent") {
		t.Error("ValidEventType(\"nonexistent\") = true, want false")
	}
}

func TestEventTypeLabel(t *testing.T) {
	if got := EventTypeLabel(database.EventHangout); got != "Тусовка" {
		t.Errorf("EventTypeLabel(hangout) = %q, want %q", got, "Тусовка")
	}

	// Unknown type falls back to raw string.
	if got := EventTypeLabel("unknown"); got != "unknown" {
		t.Errorf("EventTypeLabel(\"unknown\") = %q, want %q", got, "unknown")
	}
}

func TestEventTypeEmoji(t *testing.T) {
	if got := EventTypeEmoji(database.EventParty); got != "\U0001f389" {
		t.Errorf("EventTypeEmoji(party) = %q, want 🎉", got)
	}

	// Unknown type falls back to default emoji.
	if got := EventTypeEmoji("unknown"); got != "\U0001f4c5" {
		t.Errorf("EventTypeEmoji(\"unknown\") = %q, want 📅", got)
	}
}
