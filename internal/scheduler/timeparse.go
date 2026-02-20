package scheduler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseNaturalTime parses short-form time expressions relative to now.
// Supported formats:
//   - "in 10m", "in 2h", "in 1d" (relative durations)
//   - "tomorrow", "tomorrow 09:00" (next day, optional time)
//   - "monday", "monday 14:00" (next occurrence of weekday, optional time)
//   - RFC3339 passthrough ("2026-02-20T14:00:00Z")
func ParseNaturalTime(input string, now time.Time, loc *time.Location) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, fmt.Errorf("empty time expression")
	}
	if loc == nil {
		loc = now.Location()
	}
	now = now.In(loc)

	// try RFC3339 first (before lowercasing, since RFC3339 needs uppercase T/Z)
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t.UTC(), nil
	}

	input = strings.ToLower(input)

	// "in <N><unit>" pattern
	if strings.HasPrefix(input, "in ") {
		return parseRelative(input[3:], now)
	}

	// "tomorrow" with optional time
	if strings.HasPrefix(input, "tomorrow") {
		rest := strings.TrimSpace(strings.TrimPrefix(input, "tomorrow"))
		tomorrow := now.AddDate(0, 0, 1)
		return applyTimeOfDay(tomorrow, rest, now)
	}

	// weekday names
	if day, ok := parseWeekday(input); ok {
		parts := strings.Fields(input)
		rest := ""
		if len(parts) > 1 {
			rest = strings.Join(parts[1:], " ")
		}
		next := nextWeekday(now, day)
		return applyTimeOfDay(next, rest, now)
	}

	return time.Time{}, fmt.Errorf("unrecognized time expression: %q", input)
}

var relativePattern = regexp.MustCompile(`^(\d+)\s*(m|min|mins|minutes?|h|hrs?|hours?|d|days?|w|weeks?|s|secs?|seconds?)$`)

func parseRelative(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	match := relativePattern.FindStringSubmatch(s)
	if match == nil {
		return time.Time{}, fmt.Errorf("invalid relative duration: %q", s)
	}

	n, _ := strconv.Atoi(match[1])
	unit := match[2]

	var d time.Duration
	switch {
	case strings.HasPrefix(unit, "s"):
		d = time.Duration(n) * time.Second
	case strings.HasPrefix(unit, "m"):
		d = time.Duration(n) * time.Minute
	case strings.HasPrefix(unit, "h"):
		d = time.Duration(n) * time.Hour
	case strings.HasPrefix(unit, "d"):
		d = time.Duration(n) * 24 * time.Hour
	case strings.HasPrefix(unit, "w"):
		d = time.Duration(n) * 7 * 24 * time.Hour
	default:
		return time.Time{}, fmt.Errorf("unknown unit: %q", unit)
	}

	return now.Add(d).UTC(), nil
}

var timeOfDayPattern = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)

func applyTimeOfDay(day time.Time, timeStr string, now time.Time) (time.Time, error) {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		// keep same time of day as now
		result := time.Date(day.Year(), day.Month(), day.Day(),
			now.Hour(), now.Minute(), 0, 0, now.Location())
		return result.UTC(), nil
	}

	match := timeOfDayPattern.FindStringSubmatch(timeStr)
	if match == nil {
		return time.Time{}, fmt.Errorf("invalid time of day: %q (expected HH:MM)", timeStr)
	}

	hour, _ := strconv.Atoi(match[1])
	minute, _ := strconv.Atoi(match[2])
	if hour > 23 || minute > 59 {
		return time.Time{}, fmt.Errorf("invalid time of day: %q", timeStr)
	}

	result := time.Date(day.Year(), day.Month(), day.Day(),
		hour, minute, 0, 0, now.Location())
	return result.UTC(), nil
}

var weekdays = map[string]time.Weekday{
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
	"sunday":    time.Sunday,
	"mon":       time.Monday,
	"tue":       time.Tuesday,
	"wed":       time.Wednesday,
	"thu":       time.Thursday,
	"fri":       time.Friday,
	"sat":       time.Saturday,
	"sun":       time.Sunday,
}

func parseWeekday(input string) (time.Weekday, bool) {
	first := strings.Fields(input)[0]
	day, ok := weekdays[first]
	return day, ok
}

func nextWeekday(now time.Time, target time.Weekday) time.Time {
	days := int(target - now.Weekday())
	if days <= 0 {
		days += 7
	}
	return now.AddDate(0, 0, days)
}
