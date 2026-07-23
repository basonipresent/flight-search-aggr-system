package batik_test

import (
	"context"
	"testing"

	fixtures "flight-search-aggr-system"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider/batik"
)

func TestBatik_Fetch(t *testing.T) {
	adapter := batik.New(fixtures.FS)
	flights, err := adapter.Fetch(context.Background(), domain.SearchRequest{
		Origin:      "CGK",
		Destination: "DPS",
		Passengers:  1,
		CabinClass:  domain.Economy,
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(flights) != 3 {
		t.Fatalf("expected 3 flights, got %d", len(flights))
	}

	tests := []struct {
		flightNumber string
		wantMins     int
		wantPrice    int64
		wantStops    int
	}{
		// ID6514: price is totalPrice (1_100_000), not basePrice (980_000).
		{"ID6514", 105, 1_100_000, 0},
		{"ID6520", 110, 1_180_000, 0},
		// ID7042: travelTime claims "3h 5m" (185 min) but timestamps give 245 min.
		{"ID7042", 245, 950_000, 1},
	}

	byNumber := make(map[string]int)
	for i, f := range flights {
		byNumber[f.FlightNumber] = i
	}

	for _, tt := range tests {
		t.Run(tt.flightNumber, func(t *testing.T) {
			idx, ok := byNumber[tt.flightNumber]
			if !ok {
				t.Fatalf("flight %s not found", tt.flightNumber)
			}
			f := flights[idx]
			if f.Duration.TotalMinutes != tt.wantMins {
				t.Errorf("Duration = %d min, want %d", f.Duration.TotalMinutes, tt.wantMins)
			}
			if f.Price.Amount != tt.wantPrice {
				t.Errorf("Price = %d, want %d", f.Price.Amount, tt.wantPrice)
			}
			if f.Stops != tt.wantStops {
				t.Errorf("Stops = %d, want %d", f.Stops, tt.wantStops)
			}
		})
	}
}

// TestBatik_ID7042_IgnoresStatedTravelTime is the named test from the spec.
// travelTime: "3h 5m" is simply wrong; timestamps give 245 min.
func TestBatik_ID7042_IgnoresStatedTravelTime(t *testing.T) {
	adapter := batik.New(fixtures.FS)
	flights, _ := adapter.Fetch(context.Background(), domain.SearchRequest{})

	for _, f := range flights {
		if f.FlightNumber == "ID7042" {
			// 185 min would mean we used travelTime; 245 means we used timestamps.
			if f.Duration.TotalMinutes == 185 {
				t.Error("ID7042: used travelTime field (185 min) instead of computing from timestamps (245 min)")
			}
			if f.Duration.TotalMinutes != 245 {
				t.Errorf("ID7042: Duration = %d min, want 245", f.Duration.TotalMinutes)
			}
			return
		}
	}
	t.Fatal("ID7042 not found in results")
}
