package events

import (
	"testing"
	"time"
)

func TestParseEventTime_ValidFormat(t *testing.T) {
	got, err := parseEventTime("15.03 20:00")
	if err != nil {
		t.Fatalf("parseEventTime(\"15.03 20:00\") error: %v", err)
	}
	if got.Month() != time.March || got.Day() != 15 || got.Hour() != 20 || got.Minute() != 0 {
		t.Errorf("parseEventTime(\"15.03 20:00\") = %v, want March 15 20:00", got)
	}
}

func TestParseEventTime_InvalidFormat(t *testing.T) {
	_, err := parseEventTime("not-a-date")
	if err == nil {
		t.Error("parseEventTime(\"not-a-date\") expected error, got nil")
	}
}

func TestParseEventTime_YearRollover(t *testing.T) {
	// Use a date that has already passed this year — it should roll to next year.
	past := time.Now().AddDate(0, 0, -1)
	input := past.Format("02.01 15:04")

	got, err := parseEventTime(input)
	if err != nil {
		t.Fatalf("parseEventTime(%q) error: %v", input, err)
	}
	if !got.After(time.Now()) {
		t.Errorf("parseEventTime(%q) = %v, expected future date", input, got)
	}
}

func TestParseEventTime_FutureDate(t *testing.T) {
	// Use a date well in the future this year — should stay in current year.
	future := time.Now().AddDate(0, 0, 30)
	input := future.Format("02.01 15:04")

	got, err := parseEventTime(input)
	if err != nil {
		t.Fatalf("parseEventTime(%q) error: %v", input, err)
	}
	if got.Year() != time.Now().Year() {
		t.Errorf("parseEventTime(%q).Year() = %d, want %d", input, got.Year(), time.Now().Year())
	}
}
