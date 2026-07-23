package airasia_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	fixtures "flight-search-aggr-system"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider/airasia"
)

func TestAirAsia_Fetch(t *testing.T) {
	adapter := airasia.New(fixtures.FS)
	flights, err := adapter.Fetch(context.Background(), domain.SearchRequest{
		Origin:      "CGK",
		Destination: "DPS",
		Passengers:  1,
		CabinClass:  domain.Economy,
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(flights) != 4 {
		t.Fatalf("expected 4 flights, got %d", len(flights))
	}
}

// TestAirAsia_QZ7250 covers the one-stop flight and all the nil/empty traps.
func TestAirAsia_QZ7250(t *testing.T) {
	adapter := airasia.New(fixtures.FS)
	flights, _ := adapter.Fetch(context.Background(), domain.SearchRequest{})

	var f *domain.Flight
	for i := range flights {
		if flights[i].FlightNumber == "QZ7250" {
			f = &flights[i]
			break
		}
	}
	if f == nil {
		t.Fatal("QZ7250 not found")
	}

	if f.Duration.TotalMinutes != 260 {
		t.Errorf("Duration = %d min, want 260", f.Duration.TotalMinutes)
	}
	if f.Stops != 1 {
		t.Errorf("Stops = %d, want 1", f.Stops)
	}
	if len(f.Layovers) != 1 || f.Layovers[0].Airport != "SOC" {
		t.Errorf("Layovers = %v, want [{SOC 95}]", f.Layovers)
	}
	if f.Layovers[0].Minutes != 95 {
		t.Errorf("Layover minutes = %d, want 95", f.Layovers[0].Minutes)
	}

	// Aircraft must be nil (absent in provider) — serialises as JSON null.
	if f.Aircraft != nil {
		t.Errorf("Aircraft = %v, want nil", f.Aircraft)
	}

	// Amenities must be non-nil and empty — serialises as JSON [].
	if f.Amenities == nil {
		t.Error("Amenities is nil, want []string{}")
	}
	if len(f.Amenities) != 0 {
		t.Errorf("Amenities = %v, want empty slice", f.Amenities)
	}
}

// TestAirAsia_JSONRoundTrip proves aircraft serialises as null and amenities as [].
func TestAirAsia_JSONRoundTrip(t *testing.T) {
	adapter := airasia.New(fixtures.FS)
	flights, _ := adapter.Fetch(context.Background(), domain.SearchRequest{})

	var qz7250 *domain.Flight
	for i := range flights {
		if flights[i].FlightNumber == "QZ7250" {
			qz7250 = &flights[i]
			break
		}
	}
	if qz7250 == nil {
		t.Fatal("QZ7250 not found")
	}

	// Wrap in a struct that matches the JSON output shape.
	type wire struct {
		Aircraft  *string  `json:"aircraft"`
		Amenities []string `json:"amenities"`
	}
	w := wire{Aircraft: qz7250.Aircraft, Amenities: qz7250.Amenities}

	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	got := string(b)

	if !strings.Contains(got, `"aircraft":null`) {
		t.Errorf("expected aircraft:null in JSON, got: %s", got)
	}
	if !strings.Contains(got, `"amenities":[]`) {
		t.Errorf("expected amenities:[] in JSON, got: %s", got)
	}
}
