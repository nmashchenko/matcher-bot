package settings

import (
	"strings"
	"testing"

	"matcher-bot/internal/database"
	"matcher-bot/internal/messages"
)

func TestSettingsText_NoFilter(t *testing.T) {
	h := &Handler{}
	user := &database.User{}

	got := h.settingsText(user)
	if !strings.Contains(got, messages.SettingsFilterAll) {
		t.Errorf("settingsText with no filter should show %q: %q", messages.SettingsFilterAll, got)
	}
}

func TestSettingsText_WithFilter(t *testing.T) {
	h := &Handler{}
	pref := "party"
	user := &database.User{PreferredEventType: &pref}

	got := h.settingsText(user)
	if !strings.Contains(got, "Вечеринка") {
		t.Errorf("settingsText with party filter should show label: %q", got)
	}
	if !strings.Contains(got, "🎉") {
		t.Errorf("settingsText with party filter should show emoji: %q", got)
	}
}

func TestSettingsText_AllTypes(t *testing.T) {
	h := &Handler{}
	types := map[string]string{
		"hangout": "Тусовка",
		"gaming":  "Игры",
		"date":    "Свидание",
		"sports":  "Спорт",
		"concert": "Концерт / Шоу",
	}
	for typ, label := range types {
		pref := typ
		user := &database.User{PreferredEventType: &pref}
		got := h.settingsText(user)
		if !strings.Contains(got, label) {
			t.Errorf("settingsText(%q) should contain %q: %q", typ, label, got)
		}
	}
}

func TestSettingsText_UnknownType(t *testing.T) {
	h := &Handler{}
	pref := "unknown_type"
	user := &database.User{PreferredEventType: &pref}

	got := h.settingsText(user)
	// Unknown types fall back to raw string via EventTypeLabel.
	if !strings.Contains(got, "unknown_type") {
		t.Errorf("settingsText with unknown type should fall back to raw: %q", got)
	}
}
