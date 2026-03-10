package events

import "testing"

func TestParseEventUser_Valid(t *testing.T) {
	eventID, tgID, err := parseEventUser("abc-uuid:123")
	if err != nil {
		t.Fatalf("parseEventUser(\"abc-uuid:123\") error: %v", err)
	}
	if eventID != "abc-uuid" {
		t.Errorf("eventID = %q, want %q", eventID, "abc-uuid")
	}
	if tgID != 123 {
		t.Errorf("tgID = %d, want %d", tgID, 123)
	}
}

func TestParseEventUser_MissingColon(t *testing.T) {
	_, _, err := parseEventUser("nocolon")
	if err == nil {
		t.Error("parseEventUser(\"nocolon\") expected error, got nil")
	}
}

func TestParseEventUser_NonNumericID(t *testing.T) {
	_, _, err := parseEventUser("uuid:abc")
	if err == nil {
		t.Error("parseEventUser(\"uuid:abc\") expected error, got nil")
	}
}

func TestParseEventUser_EmptyString(t *testing.T) {
	_, _, err := parseEventUser("")
	if err == nil {
		t.Error("parseEventUser(\"\") expected error, got nil")
	}
}
