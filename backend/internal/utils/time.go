package utils

import (
	"strings"
	"time"
)

var ParisLocation *time.Location

func init() {
	var err error
	ParisLocation, err = time.LoadLocation("Europe/Paris")
	if err != nil {
		ParisLocation = time.FixedZone("Europe/Paris", 3600)
	}
}

func NowInParis() time.Time {
	return time.Now().In(ParisLocation)
}

func InParis(t time.Time) time.Time {
	if t.IsZero() {
		return NowInParis()
	}
	return t.In(ParisLocation)
}

func ParseCustomTimeInParis(val string) (time.Time, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339, val); err == nil {
		return parsed.In(ParisLocation).Truncate(time.Second), true
	}
	if parsed, err := time.ParseInLocation("2006-01-02T15:04:05", val, ParisLocation); err == nil {
		return parsed.Truncate(time.Second), true
	}
	if parsed, err := time.ParseInLocation("2006-01-02T15:04", val, ParisLocation); err == nil {
		return parsed.Truncate(time.Second), true
	}
	if parsed, err := time.ParseInLocation("15:04", val, ParisLocation); err == nil {
		now := NowInParis()
		return time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, ParisLocation), true
	}
	return time.Time{}, false
}
