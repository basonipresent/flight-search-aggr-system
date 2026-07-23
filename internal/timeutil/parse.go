// Package timeutil provides timezone-aware time parsing for the four provider formats.
package timeutil

import (
	"fmt"
	"sync"
	"time"
)

// batikLayout matches Batik Air's offset format (+0700, no colon).
// time.RFC3339 rejects this input.
const batikLayout = "2006-01-02T15:04:05-0700"

var locationCache sync.Map

// ParseRFC3339 parses a standard RFC3339 timestamp (Garuda, AirAsia).
func ParseRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("ParseRFC3339: %w", err)
	}
	return t, nil
}

// ParseCompactOffset parses a timestamp with a colon-less UTC offset (Batik Air).
// Example: "2025-12-15T07:15:00+0700"
func ParseCompactOffset(s string) (time.Time, error) {
	t, err := time.Parse(batikLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("ParseCompactOffset: %w", err)
	}
	return t, nil
}

// ParseInZone parses a naive wall-clock timestamp anchored to an IANA zone (Lion Air).
// Example: ParseInZone("2025-12-15T05:30:00", "Asia/Jakarta")
func ParseInZone(s, ianaZone string) (time.Time, error) {
	loc, err := loadLocation(ianaZone)
	if err != nil {
		return time.Time{}, fmt.Errorf("ParseInZone: load %q: %w", ianaZone, err)
	}
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("ParseInZone: parse %q: %w", s, err)
	}
	return t, nil
}

func loadLocation(name string) (*time.Location, error) {
	if v, ok := locationCache.Load(name); ok {
		return v.(*time.Location), nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	locationCache.Store(name, loc)
	return loc, nil
}
