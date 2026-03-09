package ptr

// Str returns a pointer to s, or nil if s is empty.
func Str(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Deref returns the value pointed to by s, or "" if s is nil.
func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
