package events

import (
	"strings"
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

func TestParseEventTime_UsesLocalTimezone(t *testing.T) {
	// The result should be in the server's local timezone, not UTC.
	future := time.Now().AddDate(0, 0, 10)
	input := future.Format("02.01 15:04")

	got, err := parseEventTime(input)
	if err != nil {
		t.Fatalf("parseEventTime(%q) error: %v", input, err)
	}
	if got.Location() != time.Now().Location() {
		t.Errorf("parseEventTime timezone = %v, want %v", got.Location(), time.Now().Location())
	}
}

func TestParseEventTime_WhitespaceTrimmed(t *testing.T) {
	got, err := parseEventTime("  15.03 20:00  ")
	if err != nil {
		t.Fatalf("parseEventTime with whitespace error: %v", err)
	}
	if got.Month() != time.March || got.Day() != 15 {
		t.Errorf("parseEventTime with whitespace = %v, want March 15", got)
	}
}

func TestParseEventTime_MidnightJan1(t *testing.T) {
	got, err := parseEventTime("01.01 00:00")
	if err != nil {
		t.Fatalf("parseEventTime(\"01.01 00:00\") error: %v", err)
	}
	if got.Month() != time.January || got.Day() != 1 {
		t.Errorf("parseEventTime = %v, want Jan 1", got)
	}
	if got.Hour() != 0 || got.Minute() != 0 {
		t.Errorf("parseEventTime time = %02d:%02d, want 00:00", got.Hour(), got.Minute())
	}
}

func TestParseEventTime_Dec31(t *testing.T) {
	got, err := parseEventTime("31.12 23:59")
	if err != nil {
		t.Fatalf("parseEventTime(\"31.12 23:59\") error: %v", err)
	}
	if got.Month() != time.December || got.Day() != 31 {
		t.Errorf("parseEventTime = %v, want Dec 31", got)
	}
}

func TestParseEventTime_InvalidFormats(t *testing.T) {
	bad := []string{
		"",
		"15/03 20:00",  // wrong separator
		"2025-03-15",   // ISO format
		"15.03",        // missing time
		"20:00",        // missing date
		"32.01 20:00",  // invalid day
		"15.13 20:00",  // invalid month
		"abc",
	}
	for _, input := range bad {
		_, err := parseEventTime(input)
		if err == nil {
			t.Errorf("parseEventTime(%q) expected error, got nil", input)
		}
	}
}

func TestParseEventTime_SecondsZeroed(t *testing.T) {
	future := time.Now().AddDate(0, 0, 10)
	input := future.Format("02.01 15:04")

	got, err := parseEventTime(input)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if got.Second() != 0 || got.Nanosecond() != 0 {
		t.Errorf("parseEventTime should zero seconds, got %v", got)
	}
}

func TestParseEventTime_TimeFormatConsistentWithCreateSuccess(t *testing.T) {
	// parseEventTime parses "DD.MM HH:MM" and CreateSuccess formats with "02.01 15:04".
	// Verify roundtrip: format a time, parse it back.
	original := time.Date(2025, 7, 20, 14, 30, 0, 0, time.Now().Location())
	formatted := original.Format("02.01 15:04")

	if !strings.Contains(formatted, "20.07") || !strings.Contains(formatted, "14:30") {
		t.Fatalf("unexpected format: %q", formatted)
	}

	got, err := parseEventTime(formatted)
	if err != nil {
		t.Fatalf("roundtrip parse error: %v", err)
	}
	if got.Month() != original.Month() || got.Day() != original.Day() ||
		got.Hour() != original.Hour() || got.Minute() != original.Minute() {
		t.Errorf("roundtrip mismatch: got %v, want month=%v day=%d %02d:%02d",
			got, original.Month(), original.Day(), original.Hour(), original.Minute())
	}
}
