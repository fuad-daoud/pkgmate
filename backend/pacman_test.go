//go:build pacman || all_backends

package backend

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
		wantTime time.Time
	}{
		{
			name:     "layout_two_digit_day",
			input:    "Thu 28 Nov 2024 10:30:45 AM UTC",
			wantZero: false,
			wantTime: time.Date(2024, time.November, 28, 10, 30, 45, 0, time.UTC),
		},
		{
			name:     "layout_single_digit_day",
			input:    "Thu 7 Nov 2024 10:30:45 AM UTC",
			wantZero: false,
			wantTime: time.Date(2024, time.November, 7, 10, 30, 45, 0, time.UTC),
		},
		{
			name:     "layout_ansic",
			input:    "Thu Nov 28 10:30:45 2024",
			wantZero: false,
			wantTime: time.Date(2024, time.November, 28, 10, 30, 45, 0, time.UTC),
		},
		{
			name:     "garbage_random_string",
			input:    "not a valid date",
			wantZero: true,
		},
		{
			name:     "garbage_empty_string",
			input:    "",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDate(tt.input)
			if tt.wantZero {
				if !got.IsZero() {
					t.Fatalf("parseDate(%q) = %v; want zero time", tt.input, got)
				}
			} else {
				if got.IsZero() {
					t.Fatalf("parseDate(%q) returned zero time; want valid time", tt.input)
				}
				if tt.name == "layout_ansic" {
					// time.ANSIC does not carry timezone, so compare year, month, day, hour, min, sec
					if got.Year() != tt.wantTime.Year() || got.Month() != tt.wantTime.Month() || got.Day() != tt.wantTime.Day() || got.Hour() != tt.wantTime.Hour() || got.Minute() != tt.wantTime.Minute() || got.Second() != tt.wantTime.Second() {
						t.Fatalf("parseDate(%q) = %v; want %v", tt.input, got, tt.wantTime)
					}
				} else {
					if !got.Equal(tt.wantTime) {
						t.Fatalf("parseDate(%q) = %v; want %v", tt.input, got, tt.wantTime)
					}
				}
			}
		})
	}
}
