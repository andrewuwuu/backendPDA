package timeutil

import (
	"testing"
	"time"
)

func TestJakartaLocation(t *testing.T) {
	loc := JakartaLocation()
	if loc == nil {
		t.Fatal("JakartaLocation() returned nil")
	}

	// The location should represent UTC+7.
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	_, offset := now.Zone()
	if offset != 7*60*60 {
		t.Errorf("expected UTC+7 offset (25200), got %d", offset)
	}
}

func TestJakartaLocationIdempotent(t *testing.T) {
	loc1 := JakartaLocation()
	loc2 := JakartaLocation()
	if loc1 != loc2 {
		t.Error("JakartaLocation() should return the same pointer on repeated calls")
	}
}

func TestNowJakarta(t *testing.T) {
	now := NowJakarta()
	loc := JakartaLocation()
	if now.Location() != loc {
		t.Error("NowJakarta() should return time in Jakarta location")
	}
}

func TestTruncateToHour(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "already at hour boundary",
			input:    time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC),
		},
		{
			name:     "mid-hour",
			input:    time.Date(2024, 6, 15, 14, 30, 45, 123456789, time.UTC),
			expected: time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC),
		},
		{
			name:     "last second of hour",
			input:    time.Date(2024, 6, 15, 14, 59, 59, 999999999, time.UTC),
			expected: time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC),
		},
		{
			name:     "preserves location",
			input:    time.Date(2024, 6, 15, 14, 30, 0, 0, JakartaLocation()),
			expected: time.Date(2024, 6, 15, 14, 0, 0, 0, JakartaLocation()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateToHour(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("TruncateToHour(%v) = %v, want %v", tt.input, result, tt.expected)
			}
			if result.Location() != tt.input.Location() {
				t.Errorf("TruncateToHour should preserve location, got %v want %v",
					result.Location(), tt.input.Location())
			}
		})
	}
}

func TestTruncateToJakartaHour(t *testing.T) {
	input := time.Date(2024, 6, 15, 8, 30, 45, 0, time.UTC)

	result := TruncateToJakartaHour(input)

	expected := time.Date(2024, 6, 15, 15, 0, 0, 0, JakartaLocation())
	if !result.Equal(expected) {
		t.Fatalf("TruncateToJakartaHour(%v) = %v, want %v", input, result, expected)
	}
	if result.Location() != JakartaLocation() {
		t.Fatalf("expected Jakarta location, got %v", result.Location())
	}
}
