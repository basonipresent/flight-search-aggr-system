package timeutil

import "strconv"

// FormatDuration converts a total number of minutes to a human-readable string.
// Examples: 260 → "4h 20m", 105 → "1h 45m", 45 → "45m", 60 → "1h 0m".
func FormatDuration(mins int) string {
	h := mins / 60
	m := mins % 60
	if h == 0 {
		return strconv.Itoa(m) + "m"
	}
	return strconv.Itoa(h) + "h " + strconv.Itoa(m) + "m"
}
