package lion_test

import (
	"context"
	fixtures "flight-search-aggr-system"
	"testing"

	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider/lion"
)

func TestLion_Fetch(t *testing.T) {
	adapter := lion.New(fixtures.FS)
	flights, err := adapter.Fetch(context.Background(), domain.SearchRequest{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(flights) != 3 {
		t.Fatalf("expected 3 flights, got %d", len(flights))
	}

	tests := []struct {
		id          string
		wantMins    int
		wantStops   int
		wantLayover int // 0 if direct
	}{
		{"JT740", 105, 0, 0},
		{"JT742", 110, 0, 0},
		// JT650: 1 stop with 75-min layover at SUB.
		{"JT650", 230, 1, 75},
	}

	byID := make(map[string]int)
	for i, f := range flights {
		byID[f.FlightNumber] = i
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			idx, ok := byID[tt.id]
			if !ok {
				t.Fatalf("flight %s not found", tt.id)
			}
			f := flights[idx]

			if f.Duration.TotalMinutes != tt.wantMins {
				t.Errorf("Duration = %d min, want %d", f.Duration.TotalMinutes, tt.wantMins)
			}
			if f.Stops != tt.wantStops {
				t.Errorf("Stops = %d, want %d", f.Stops, tt.wantStops)
			}
			if tt.wantLayover > 0 {
				if len(f.Layovers) == 0 {
					t.Fatalf("expected layover, got none")
				}
				if f.Layovers[0].Minutes != tt.wantLayover {
					t.Errorf("Layover = %d min, want %d", f.Layovers[0].Minutes, tt.wantLayover)
				}
			}
		})
	}
}
