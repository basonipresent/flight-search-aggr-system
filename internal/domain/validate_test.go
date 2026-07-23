package domain

import (
	"testing"
	"time"
)

func TestFlight_Validate(t *testing.T) {
	base := time.Date(2025, 12, 15, 6, 0, 0, 0, time.UTC)

	validFlight := func() Flight {
		f := NewFlight()
		f.Departure = Endpoint{Instant: base}
		f.Arrival = Endpoint{Instant: base.Add(2 * time.Hour)}
		f.Price = Money{Amount: 500_000, Currency: "IDR"}
		f.AvailableSeats = 5
		f.CabinClass = Economy
		return f
	}

	tests := []struct {
		name    string
		mutate  func(*Flight)
		wantErr bool
		errField string
	}{
		{
			name:    "valid flight passes",
			mutate:  func(f *Flight) {},
			wantErr: false,
		},
		{
			name: "arrival before departure is rejected",
			mutate: func(f *Flight) {
				// Swap dep and arr to create an invalid flight.
				f.Arrival.Instant = base.Add(-1 * time.Hour)
			},
			wantErr:  true,
			errField: "arrival",
		},
		{
			name: "arrival equal to departure is rejected",
			mutate: func(f *Flight) {
				f.Arrival.Instant = base
			},
			wantErr:  true,
			errField: "arrival",
		},
		{
			name: "zero price is rejected",
			mutate: func(f *Flight) {
				f.Price.Amount = 0
			},
			wantErr:  true,
			errField: "price",
		},
		{
			name: "negative price is rejected",
			mutate: func(f *Flight) {
				f.Price.Amount = -1
			},
			wantErr:  true,
			errField: "price",
		},
		{
			name: "negative seats are rejected",
			mutate: func(f *Flight) {
				f.AvailableSeats = -1
			},
			wantErr:  true,
			errField: "available_seats",
		},
		{
			name: "unknown cabin class is rejected",
			mutate: func(f *Flight) {
				f.CabinClass = "premium" // not a valid enum value
			},
			wantErr:  true,
			errField: "cabin_class",
		},
		{
			name: "stops corrected to match layovers",
			mutate: func(f *Flight) {
				f.Stops = 0
				f.Layovers = []Layover{{Airport: "SUB", Minutes: 60}}
			},
			wantErr: false, // self-corrects, does not reject
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := validFlight()
			tt.mutate(&f)

			err := f.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errField != "" {
				ve, ok := err.(*ValidationError)
				if !ok {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				if ve.Field != tt.errField {
					t.Errorf("expected field %q, got %q", tt.errField, ve.Field)
				}
			}
		})
	}
}
