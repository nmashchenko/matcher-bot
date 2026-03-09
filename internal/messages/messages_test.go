package messages

import (
	"strings"
	"testing"
	"time"
)

func TestVerified(t *testing.T) {
	got := Verified("Boston", "MA")
	if !strings.Contains(got, "Boston") || !strings.Contains(got, "MA") {
		t.Errorf("Verified missing city/state: %q", got)
	}
}

func TestCreateConfirm(t *testing.T) {
	ts := time.Date(2025, 3, 15, 20, 0, 0, 0, time.UTC)
	got := CreateConfirm("🎉", "Вечеринка", "Test", "A description", "NYC", ts, 10)
	if !strings.Contains(got, "Test") {
		t.Errorf("CreateConfirm missing title: %q", got)
	}
	if !strings.Contains(got, "A description") {
		t.Errorf("CreateConfirm missing desc: %q", got)
	}
	if !strings.Contains(got, "NYC") {
		t.Errorf("CreateConfirm missing city: %q", got)
	}
	if !strings.Contains(got, "10") {
		t.Errorf("CreateConfirm missing capacity: %q", got)
	}
}

func TestCreateConfirm_NoDesc(t *testing.T) {
	ts := time.Date(2025, 3, 15, 20, 0, 0, 0, time.UTC)
	got := CreateConfirm("🎉", "Вечеринка", "Test", "", "NYC", ts, 5)
	if strings.Contains(got, "\U0001f4ac") {
		t.Errorf("CreateConfirm with empty desc should omit desc line: %q", got)
	}
}

func TestEventCard(t *testing.T) {
	ts := time.Date(2025, 6, 1, 18, 30, 0, 0, time.UTC)
	got := EventCard("⚽", "Спорт", "Football", "Friendly match", "LA", ts, 3, 10)
	if !strings.Contains(got, "Football") {
		t.Errorf("EventCard missing title: %q", got)
	}
	if !strings.Contains(got, "Friendly match") {
		t.Errorf("EventCard missing desc: %q", got)
	}
	if !strings.Contains(got, "3/10") {
		t.Errorf("EventCard missing participants: %q", got)
	}
}

func TestJoinRequest(t *testing.T) {
	got := JoinRequest("Party", 12345, "Alice", "alice_tg", 25, "NYC", "NY")
	if !strings.Contains(got, "Party") {
		t.Errorf("JoinRequest missing event title: %q", got)
	}
	if !strings.Contains(got, "Alice") {
		t.Errorf("JoinRequest missing name: %q", got)
	}
	if !strings.Contains(got, "@alice_tg") {
		t.Errorf("JoinRequest missing username: %q", got)
	}
	if !strings.Contains(got, "25") {
		t.Errorf("JoinRequest missing age: %q", got)
	}
}

func TestJoinRequest_NoUsername(t *testing.T) {
	got := JoinRequest("Party", 12345, "Alice", "", 25, "NYC", "NY")
	if !strings.Contains(got, "юзернейм не указан") {
		t.Errorf("JoinRequest without username should show fallback: %q", got)
	}
}

func TestBrowseFilterNotice(t *testing.T) {
	got := BrowseFilterNotice("🎮", "Игры", "Boston", true)
	if !strings.Contains(got, "по всем городам") {
		t.Errorf("BrowseFilterNotice gaming should say all cities: %q", got)
	}

	got = BrowseFilterNotice("🎉", "Вечеринка", "Boston", false)
	if !strings.Contains(got, "Boston") {
		t.Errorf("BrowseFilterNotice non-gaming should include city: %q", got)
	}
}

func TestSettingsView(t *testing.T) {
	got := SettingsView("Все типы")
	if !strings.Contains(got, "Настройки") {
		t.Errorf("SettingsView missing title: %q", got)
	}
	if !strings.Contains(got, "Все типы") {
		t.Errorf("SettingsView missing filter: %q", got)
	}
}

func TestMyEventRow(t *testing.T) {
	ts := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	got := MyEventRow("🎉", "Party", ts, 2, 10, 3)
	if !strings.Contains(got, "Party") {
		t.Errorf("MyEventRow missing title: %q", got)
	}
	if !strings.Contains(got, "2/10") {
		t.Errorf("MyEventRow missing counts: %q", got)
	}
	if !strings.Contains(got, "+3") {
		t.Errorf("MyEventRow missing pending: %q", got)
	}
}

func TestMyEventRow_NoPending(t *testing.T) {
	ts := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	got := MyEventRow("🎉", "Party", ts, 2, 10, 0)
	if strings.Contains(got, "+0") || strings.Contains(got, "\U0001f514") {
		t.Errorf("MyEventRow with 0 pending should omit bell: %q", got)
	}
}

func TestParticipantApproved(t *testing.T) {
	got := ParticipantApproved("Party", "Host", "host_tg", "NYC", []string{"Alice", "Bob"})
	if !strings.Contains(got, "Party") {
		t.Errorf("ParticipantApproved missing event title: %q", got)
	}
	if !strings.Contains(got, "@host_tg") {
		t.Errorf("ParticipantApproved missing host username: %q", got)
	}
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Bob") {
		t.Errorf("ParticipantApproved missing participants: %q", got)
	}
}

func TestParticipantApproved_NoUsername(t *testing.T) {
	got := ParticipantApproved("Party", "Host", "", "NYC", nil)
	if strings.Contains(got, "@") {
		t.Errorf("ParticipantApproved no username should omit @: %q", got)
	}
}
