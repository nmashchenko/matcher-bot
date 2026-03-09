package util

import "testing"

func TestStr(t *testing.T) {
	// Empty string returns nil.
	if got := Str(""); got != nil {
		t.Errorf("Str(\"\") = %v, want nil", got)
	}

	// Non-empty string returns pointer to value.
	got := Str("hello")
	if got == nil {
		t.Fatal("Str(\"hello\") = nil, want non-nil")
	}
	if *got != "hello" {
		t.Errorf("*Str(\"hello\") = %q, want %q", *got, "hello")
	}
}

func TestDeref(t *testing.T) {
	// Nil returns empty string.
	if got := Deref(nil); got != "" {
		t.Errorf("Deref(nil) = %q, want \"\"", got)
	}

	// Non-nil returns pointed value.
	s := "world"
	if got := Deref(&s); got != "world" {
		t.Errorf("Deref(&\"world\") = %q, want %q", got, "world")
	}
}
