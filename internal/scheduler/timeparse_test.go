package scheduler

import (
	"testing"
	"time"
)

func TestParseNaturalTime_Relative(t *testing.T) {
	now := time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		input string
		want  time.Time
	}{
		{"in 10m", now.Add(10 * time.Minute)},
		{"in 2h", now.Add(2 * time.Hour)},
		{"in 1d", now.Add(24 * time.Hour)},
		{"in 30s", now.Add(30 * time.Second)},
		{"in 1w", now.Add(7 * 24 * time.Hour)},
		{"in 15 minutes", now.Add(15 * time.Minute)},
		{"in 3 hours", now.Add(3 * time.Hour)},
	}

	for _, tt := range tests {
		got, err := ParseNaturalTime(tt.input, now, time.UTC)
		if err != nil {
			t.Errorf("ParseNaturalTime(%q): %v", tt.input, err)
			continue
		}
		if !got.Equal(tt.want) {
			t.Errorf("ParseNaturalTime(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestParseNaturalTime_Tomorrow(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Vienna")
	now := time.Date(2026, 2, 20, 14, 30, 0, 0, loc)

	// tomorrow without time → same time of day
	got, err := ParseNaturalTime("tomorrow", now, loc)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 2, 21, 14, 30, 0, 0, loc).UTC()
	if !got.Equal(want) {
		t.Errorf("tomorrow = %s, want %s", got, want)
	}

	// tomorrow with time
	got, err = ParseNaturalTime("tomorrow 09:00", now, loc)
	if err != nil {
		t.Fatal(err)
	}
	want = time.Date(2026, 2, 21, 9, 0, 0, 0, loc).UTC()
	if !got.Equal(want) {
		t.Errorf("tomorrow 09:00 = %s, want %s", got, want)
	}
}

func TestParseNaturalTime_Weekday(t *testing.T) {
	loc := time.UTC
	// Friday 2026-02-20
	now := time.Date(2026, 2, 20, 10, 0, 0, 0, loc)

	got, err := ParseNaturalTime("monday 14:00", now, loc)
	if err != nil {
		t.Fatal(err)
	}
	// next monday is 2026-02-23
	want := time.Date(2026, 2, 23, 14, 0, 0, 0, loc).UTC()
	if !got.Equal(want) {
		t.Errorf("monday 14:00 = %s, want %s", got, want)
	}

	// same weekday → next week
	got, err = ParseNaturalTime("friday", now, loc)
	if err != nil {
		t.Fatal(err)
	}
	want = time.Date(2026, 2, 27, 10, 0, 0, 0, loc).UTC()
	if !got.Equal(want) {
		t.Errorf("friday = %s, want %s", got, want)
	}
}

func TestParseNaturalTime_RFC3339(t *testing.T) {
	now := time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)
	input := "2026-03-01T09:00:00Z"
	got, err := ParseNaturalTime(input, now, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := time.Parse(time.RFC3339, input)
	if !got.Equal(want) {
		t.Errorf("RFC3339 = %s, want %s", got, want)
	}
}

func TestParseNaturalTime_InvalidInputs(t *testing.T) {
	now := time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)

	invalids := []string{
		"",
		"yesterday",
		"in abc",
		"tomorrow 25:00",
		"in 10x",
	}
	for _, input := range invalids {
		_, err := ParseNaturalTime(input, now, time.UTC)
		if err == nil {
			t.Errorf("ParseNaturalTime(%q): expected error", input)
		}
	}
}

func TestParseNaturalTime_ShortWeekday(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 2, 20, 10, 0, 0, 0, loc) // Friday

	got, err := ParseNaturalTime("mon 08:00", now, loc)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 2, 23, 8, 0, 0, 0, loc).UTC()
	if !got.Equal(want) {
		t.Errorf("mon 08:00 = %s, want %s", got, want)
	}
}
