package cliutils

import (
	"fmt"
	"strings"
	"time"
)

func TimeStrParse(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time value")
	}

	now := time.Now()
	loc := now.Location()

	dateTimeLayouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
	}
	for _, layout := range dateTimeLayouts {
		if parsed, err := time.ParseInLocation(layout, s, loc); err == nil {
			return parsed, nil
		}
	}

	for _, layout := range []string{"15:04:05", "15:04", "15h04", "3:04PM", "3PM"} {
		if parsed, err := time.ParseInLocation(layout, strings.ToUpper(s), loc); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, loc), nil
		}
	}

	for _, layout := range []string{"2006-01-02", "2006/01/02", "02/01/2006", "02-01-2006"} {
		if parsed, err := time.ParseInLocation(layout, s, loc); err == nil {
			return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), loc), nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format %q", s)
}

func TimeToString(t time.Time) string {
	now := time.Now()
	localTime := t.In(now.Location())
	if localTime.Year() == now.Year() &&
		localTime.Month() == now.Month() &&
		localTime.Day() == now.Day() {
		return localTime.Format("15:04")
	}
	return localTime.Format("2006-01-02 15:04")
}
