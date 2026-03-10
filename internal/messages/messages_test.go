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
	got := CreateConfirm("🎉", "Вечеринка", "Test", "A description", "NYC", ts, 10, "")
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
	got := CreateConfirm("🎉", "Вечеринка", "Test", "", "NYC", ts, 5, "")
	if strings.Contains(got, "\U0001f4ac") {
		t.Errorf("CreateConfirm with empty desc should omit desc line: %q", got)
	}
}

func TestCreateConfirm_WithAge(t *testing.T) {
	ts := time.Date(2025, 3, 15, 20, 0, 0, 0, time.UTC)
	got := CreateConfirm("🎉", "Вечеринка", "Test", "", "NYC", ts, 5, "18-30")
	if !strings.Contains(got, "18-30") {
		t.Errorf("CreateConfirm with age should show age restriction: %q", got)
	}
}

func TestEventCard(t *testing.T) {
	ts := time.Date(2025, 6, 1, 18, 30, 0, 0, time.UTC)
	got := EventCard("⚽", "Спорт", "Football", "Friendly match", "LA", ts, 3, 10, "")
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

func TestEventCard_WithAge(t *testing.T) {
	ts := time.Date(2025, 6, 1, 18, 30, 0, 0, time.UTC)
	got := EventCard("⚽", "Спорт", "Football", "", "LA", ts, 3, 10, "21-35")
	if !strings.Contains(got, "21-35") {
		t.Errorf("EventCard with age should show age restriction: %q", got)
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

func TestParticipantApproved_NoParticipants(t *testing.T) {
	got := ParticipantApproved("Party", "Host", "h", "NYC", nil)
	if strings.Contains(got, "Участники") {
		t.Errorf("ParticipantApproved with no participants should omit list: %q", got)
	}
}

func TestAskAge(t *testing.T) {
	got := AskAge("Boston", "MA")
	if !strings.Contains(got, "Boston") || !strings.Contains(got, "MA") {
		t.Errorf("AskAge missing city/state: %q", got)
	}
	if !strings.Contains(got, "лет") {
		t.Errorf("AskAge missing age prompt: %q", got)
	}
}

func TestAgeFromTelegram(t *testing.T) {
	got := AgeFromTelegram(24, "NYC", "NY")
	if !strings.Contains(got, "24") {
		t.Errorf("AgeFromTelegram missing age: %q", got)
	}
	if !strings.Contains(got, "NYC") || !strings.Contains(got, "NY") {
		t.Errorf("AgeFromTelegram missing city/state: %q", got)
	}
}

func TestOnboardingComplete(t *testing.T) {
	got := OnboardingComplete("Nikita", 24, "Boston", "MA")
	if !strings.Contains(got, "Nikita") {
		t.Errorf("OnboardingComplete missing name: %q", got)
	}
	if !strings.Contains(got, "24") {
		t.Errorf("OnboardingComplete missing age: %q", got)
	}
	if !strings.Contains(got, "Boston") || !strings.Contains(got, "MA") {
		t.Errorf("OnboardingComplete missing city/state: %q", got)
	}
	for _, cmd := range []string{"/events", "/create", "/myevents", "/settings"} {
		if !strings.Contains(got, cmd) {
			t.Errorf("OnboardingComplete missing command %s: %q", cmd, got)
		}
	}
}

func TestMainMenu(t *testing.T) {
	got := MainMenu("LA", "CA")
	if !strings.Contains(got, "LA") || !strings.Contains(got, "CA") {
		t.Errorf("MainMenu missing city/state: %q", got)
	}
	for _, cmd := range []string{"/events", "/create", "/myevents", "/settings"} {
		if !strings.Contains(got, cmd) {
			t.Errorf("MainMenu missing command %s: %q", cmd, got)
		}
	}
}

func TestCreateSuccess(t *testing.T) {
	ts := time.Date(2025, 7, 20, 14, 30, 0, 0, time.UTC)
	got := CreateSuccess("Beach Party", "Miami", ts, "")
	if !strings.Contains(got, "Beach Party") {
		t.Errorf("CreateSuccess missing title: %q", got)
	}
	if !strings.Contains(got, "Miami") {
		t.Errorf("CreateSuccess missing city: %q", got)
	}
	if !strings.Contains(got, "20.07 14:30") {
		t.Errorf("CreateSuccess missing formatted time: %q", got)
	}
}

func TestCreateSuccess_WithAge(t *testing.T) {
	ts := time.Date(2025, 7, 20, 14, 30, 0, 0, time.UTC)
	got := CreateSuccess("Beach Party", "Miami", ts, "18-30")
	if !strings.Contains(got, "18-30") {
		t.Errorf("CreateSuccess with age should show age restriction: %q", got)
	}
}

func TestBrowseFilterAll(t *testing.T) {
	got := BrowseFilterAll("Chicago")
	if !strings.Contains(got, "Chicago") {
		t.Errorf("BrowseFilterAll missing city: %q", got)
	}
	if !strings.Contains(got, "все типы") {
		t.Errorf("BrowseFilterAll missing 'все типы': %q", got)
	}
}

func TestParticipantRejected(t *testing.T) {
	got := ParticipantRejected("Game Night")
	if !strings.Contains(got, "Game Night") {
		t.Errorf("ParticipantRejected missing title: %q", got)
	}
}

func TestEventCancelled(t *testing.T) {
	got := EventCancelled("Beach Party")
	if !strings.Contains(got, "Beach Party") {
		t.Errorf("EventCancelled missing title: %q", got)
	}
	if !strings.Contains(got, "отменено") {
		t.Errorf("EventCancelled missing cancel text: %q", got)
	}
}

func TestParticipantRemoved(t *testing.T) {
	got := ParticipantRemoved("Concert")
	if !strings.Contains(got, "Concert") {
		t.Errorf("ParticipantRemoved missing title: %q", got)
	}
}

func TestHostNotifyLeft(t *testing.T) {
	got := HostNotifyLeft("Party", "Alice")
	if !strings.Contains(got, "Party") {
		t.Errorf("HostNotifyLeft missing title: %q", got)
	}
	if !strings.Contains(got, "Alice") {
		t.Errorf("HostNotifyLeft missing name: %q", got)
	}
}

func TestHostApprovedConfirm(t *testing.T) {
	got := HostApprovedConfirm("Bob", "bob_tg", "Sports")
	if !strings.Contains(got, "Bob") || !strings.Contains(got, "Sports") {
		t.Errorf("HostApprovedConfirm missing name/title: %q", got)
	}
	if !strings.Contains(got, "@bob_tg") {
		t.Errorf("HostApprovedConfirm missing username: %q", got)
	}
}

func TestHostApprovedConfirm_NoUsername(t *testing.T) {
	got := HostApprovedConfirm("Bob", "", "Sports")
	if strings.Contains(got, "@") {
		t.Errorf("HostApprovedConfirm with no username should omit @: %q", got)
	}
}

func TestHostRemovedConfirm(t *testing.T) {
	got := HostRemovedConfirm("Eve", "Hangout")
	if !strings.Contains(got, "Eve") || !strings.Contains(got, "Hangout") {
		t.Errorf("HostRemovedConfirm missing name/title: %q", got)
	}
}

func TestEventCancelledConfirm(t *testing.T) {
	got := EventCancelledConfirm("Date Night")
	if !strings.Contains(got, "Date Night") {
		t.Errorf("EventCancelledConfirm missing title: %q", got)
	}
	if !strings.Contains(got, "уведомлены") {
		t.Errorf("EventCancelledConfirm missing notify text: %q", got)
	}
}

func TestEventCard_NoDesc(t *testing.T) {
	ts := time.Date(2025, 6, 1, 18, 30, 0, 0, time.UTC)
	got := EventCard("⚽", "Спорт", "Football", "", "LA", ts, 3, 10, "")
	if strings.Contains(got, "\U0001f4ac") {
		t.Errorf("EventCard with empty desc should omit desc line: %q", got)
	}
}

func TestEventCard_TimeFormat(t *testing.T) {
	ts := time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC)
	got := EventCard("🎉", "Party", "NYE", "", "NYC", ts, 0, 50, "")
	if !strings.Contains(got, "31.12 23:59") {
		t.Errorf("EventCard time format wrong: %q", got)
	}
}
