package events

import (
	"testing"

	"matcher-bot/internal/database"
)

func TestValidEventType(t *testing.T) {
	known := []database.EventType{
		database.EventHangout, database.EventGaming,
		database.EventSports, database.EventConcert, database.EventRandom,
	}
	for _, et := range known {
		if !ValidEventType(et) {
			t.Errorf("ValidEventType(%q) = false, want true", et)
		}
	}

	if ValidEventType("nonexistent") {
		t.Error("ValidEventType(\"nonexistent\") = true, want false")
	}

	// Removed types should no longer be valid.
	for _, removed := range []database.EventType{"party", "date"} {
		if ValidEventType(removed) {
			t.Errorf("ValidEventType(%q) = true, want false (removed type)", removed)
		}
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
	if got := EventTypeEmoji(database.EventGaming); got != "\U0001f3ae" {
		t.Errorf("EventTypeEmoji(gaming) = %q, want 🎮", got)
	}

	// Unknown type falls back to default emoji.
	if got := EventTypeEmoji("unknown"); got != "\U0001f4c5" {
		t.Errorf("EventTypeEmoji(\"unknown\") = %q, want 📅", got)
	}
}

func TestFormatAgeRestriction(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name   string
		min    *int
		max    *int
		want   string
	}{
		{"both nil", nil, nil, ""},
		{"both set", intPtr(18), intPtr(30), "18-30"},
		{"min only", intPtr(18), nil, "от 18"},
		{"max only", nil, intPtr(30), "до 30"},
		{"same value", intPtr(25), intPtr(26), "25-26"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatAgeRestriction(tc.min, tc.max)
			if got != tc.want {
				t.Errorf("FormatAgeRestriction(%v, %v) = %q, want %q", tc.min, tc.max, got, tc.want)
			}
		})
	}
}
