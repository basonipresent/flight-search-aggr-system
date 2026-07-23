package timeutil

import (
	"testing"
	"time"
)

// TestParseCompactOffset_RFC3339Fails proves that time.RFC3339 cannot parse
// Batik Air's offset format, and that ParseCompactOffset succeeds where it fails.
func TestParseCompactOffset_RFC3339Fails(t *testing.T) {
	batikTimestamp := "2025-12-15T07:15:00+0700"

	// Prove the problem: RFC3339 must fail on this input.
	_, err := time.Parse(time.RFC3339, batikTimestamp)
	if err == nil {
		t.Fatal("expected time.RFC3339 to fail on '+0700' (no colon), but it succeeded — test assumption is wrong")
	}

	// Prove the fix: ParseCompactOffset must succeed.
	got, err := ParseCompactOffset(batikTimestamp)
	if err != nil {
		t.Fatalf("ParseCompactOffset(%q) unexpected error: %v", batikTimestamp, err)
	}

	// The parsed instant in UTC should be 00:15 (07:15 minus +07:00).
	want := time.Date(2025, 12, 15, 0, 15, 0, 0, time.UTC)
	if !got.UTC().Equal(want) {
		t.Errorf("UTC instant: got %v, want %v", got.UTC(), want)
	}
}

// TestParseInZone_NaiveTimeParsedAsUTCIsWrong proves that using time.Parse
// (not ParseInLocation) on Lion Air's naive timestamps shifts the result by
// seven hours. This is the highest-severity silent failure in the assignment.
func TestParseInZone_NaiveTimeParsedAsUTCIsWrong(t *testing.T) {
	lionTimestamp := "2025-12-15T05:30:00" // naive, no offset
	ianaZone := "Asia/Jakarta"             // WIB = UTC+7

	// Prove the problem: time.Parse treats naive strings as UTC.
	wrongTime, err := time.Parse("2006-01-02T15:04:05", lionTimestamp)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	// wrongTime.UTC() == 2025-12-15T05:30:00Z — seven hours off

	// Prove the fix: ParseInZone anchors to the correct zone.
	correct, err := ParseInZone(lionTimestamp, ianaZone)
	if err != nil {
		t.Fatalf("ParseInZone(%q, %q) unexpected error: %v", lionTimestamp, ianaZone, err)
	}
	// correct.UTC() == 2025-12-14T22:30:00Z (05:30 WIB = 22:30 UTC previous day)
	wantUTC := time.Date(2025, 12, 14, 22, 30, 0, 0, time.UTC)
	if !correct.UTC().Equal(wantUTC) {
		t.Errorf("correct UTC: got %v, want %v", correct.UTC(), wantUTC)
	}

	// The two results must differ by exactly 7 hours.
	diff := correct.UTC().Sub(wrongTime.UTC())
	wantDiff := -7 * time.Hour
	if diff != wantDiff {
		t.Errorf("expected %v difference between wrong and correct parse, got %v", wantDiff, diff)
	}
}

func TestParseRFC3339(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantUTC time.Time
		wantErr bool
	}{
		{
			name:    "Garuda style with colon offset",
			input:   "2025-12-15T06:00:00+07:00",
			wantUTC: time.Date(2025, 12, 14, 23, 0, 0, 0, time.UTC),
		},
		{
			name:    "AirAsia style UTC",
			input:   "2025-12-15T00:00:00Z",
			wantUTC: time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "garbage input returns error",
			input:   "not-a-date",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRFC3339(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseRFC3339(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && !got.UTC().Equal(tt.wantUTC) {
				t.Errorf("UTC: got %v, want %v", got.UTC(), tt.wantUTC)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		mins int
		want string
	}{
		{260, "4h 20m"},
		{105, "1h 45m"},
		{45, "45m"},
		{60, "1h 0m"},
		{225, "3h 45m"},
		{0, "0m"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.mins)
			if got != tt.want {
				t.Errorf("FormatDuration(%d) = %q, want %q", tt.mins, got, tt.want)
			}
		})
	}
}
