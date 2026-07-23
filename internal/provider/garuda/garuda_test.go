package garuda_test

import (
	"context"
	"testing"

	fixtures "flight-search-aggr-system"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider/garuda"
)

func TestGaruda_Fetch(t *testing.T) {
	adapter := garuda.New(fixtures.FS)
	flights, err := adapter.Fetch(context.Background(), domain.SearchRequest{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(flights) != 3 {
		t.Fatalf("expected 3 flights, got %d", len(flights))
	}

	tests := []struct {
		flightNumber    string
		wantMins        int
		wantDestAirport string
		wantStops       int
	}{
		// GA400 and GA410: no segments, top-level is correct.
		{"GA400", 110, "DPS", 0},
		{"GA410", 115, "DPS", 0},
		// GA315: segments override top-level (which says SUB, 0 stops, 90 min).
		// Correct values: DPS, 1 stop, 225 min.
		{"GA315", 225, "DPS", 1},
	}

	byNumber := make(map[string]int)
	for i, f := range flights {
		byNumber[f.FlightNumber] = i
	}

	for _, tt := range tests {
		t.Run(tt.flightNumber, func(t *testing.T) {
			idx, ok := byNumber[tt.flightNumber]
			if !ok {
				t.Fatalf("flight %s not found in results", tt.flightNumber)
			}
			f := flights[idx]
			if f.Duration.TotalMinutes != tt.wantMins {
				t.Errorf("Duration.TotalMinutes = %d, want %d", f.Duration.TotalMinutes, tt.wantMins)
			}
			if f.Arrival.Airport != tt.wantDestAirport {
				t.Errorf("Arrival.Airport = %q, want %q", f.Arrival.Airport, tt.wantDestAirport)
			}
			if f.Stops != tt.wantStops {
				t.Errorf("Stops = %d, want %d", f.Stops, tt.wantStops)
			}
		})
	}
}

// TestGaruda_GA315_SegmentOverridesTopLevel is the named test from the spec.
// It documents the exact trap: top-level says SUB/0 stops/90 min; segments say DPS/1 stop/225 min.
func TestGaruda_GA315_SegmentOverridesTopLevel(t *testing.T) {
	adapter := garuda.New(fixtures.FS)
	flights, err := adapter.Fetch(context.Background(), domain.SearchRequest{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	var ga315 *struct {
		airport string
		stops   int
		mins    int
		layover int
	}
	for _, f := range flights {
		if f.FlightNumber == "GA315" {
			ga315 = &struct {
				airport string
				stops   int
				mins    int
				layover int
			}{
				airport: f.Arrival.Airport,
				stops:   f.Stops,
				mins:    f.Duration.TotalMinutes,
			}
			if len(f.Layovers) > 0 {
				ga315.layover = f.Layovers[0].Minutes
			}
			break
		}
	}
	if ga315 == nil {
		t.Fatal("GA315 not found")
	}

	// These assert the segment override worked correctly.
	if ga315.airport != "DPS" {
		t.Errorf("arrival airport = %q, want DPS (top-level wrongly says SUB)", ga315.airport)
	}
	if ga315.stops != 1 {
		t.Errorf("stops = %d, want 1 (top-level wrongly says 0)", ga315.stops)
	}
	if ga315.mins != 225 {
		t.Errorf("duration = %d min, want 225 (top-level wrongly says 90)", ga315.mins)
	}
	// Layover computed from inter-segment gap, not from layover_minutes field.
	if ga315.layover != 105 {
		t.Errorf("layover at SUB = %d min, want 105", ga315.layover)
	}
}
