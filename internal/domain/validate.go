package domain

import "fmt"

// Validate checks domain invariants and returns a *ValidationError on the first violation.
func (f *Flight) Validate() error {
	if !f.Arrival.Instant.After(f.Departure.Instant) {
		return &ValidationError{Field: "arrival", Reason: "must be after departure"}
	}
	if f.Price.Amount <= 0 {
		return &ValidationError{Field: "price", Reason: fmt.Sprintf("must be positive, got %d", f.Price.Amount)}
	}
	if f.AvailableSeats < 0 {
		return &ValidationError{Field: "available_seats", Reason: fmt.Sprintf("cannot be negative, got %d", f.AvailableSeats)}
	}
	if _, err := ParseCabinClass(string(f.CabinClass)); err != nil {
		return &ValidationError{Field: "cabin_class", Reason: "unknown value: " + string(f.CabinClass)}
	}
	// When layover data is present, trust it over the top-level stops count.
	if len(f.Layovers) > 0 && f.Stops != len(f.Layovers) {
		f.Stops = len(f.Layovers)
	}
	return nil
}
