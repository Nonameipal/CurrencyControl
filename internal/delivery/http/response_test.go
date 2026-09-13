package http

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"13.09.2026", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"01.05.2025", time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)},
		{"5.9.2026", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)},
		{"2026-09-13", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"2026.09.13", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"13/09/2026", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"13.09.2026 14:30:00", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"2026-09-13T10:00:00Z", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			parsed := parseDate(tc.input)
			if parsed == nil {
				t.Fatalf("expected parsed date for %q, got nil", tc.input)
			}
			if parsed.Year() != tc.expected.Year() || parsed.Month() != tc.expected.Month() || parsed.Day() != tc.expected.Day() {
				t.Errorf("for input %q expected %v, got %v", tc.input, tc.expected, *parsed)
			}
		})
	}

	if parseDate("") != nil {
		t.Errorf("expected empty string to return nil")
	}
	if parseDate("not-a-date") != nil {
		t.Errorf("expected invalid string to return nil")
	}
}
