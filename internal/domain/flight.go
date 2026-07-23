// Package domain defines the canonical flight model shared across all providers.
package domain

import (
	"strconv"
	"time"
)

// Flight is the canonical, normalized representation of a single flight option.
type Flight struct {
	ID             string     // "<flightNumber>_<provider>"
	Provider       string
	Airline        Airline
	FlightNumber   string
	Departure      Endpoint
	Arrival        Endpoint
	Duration       Duration
	Stops          int
	Layovers       []Layover
	Price          Money
	AvailableSeats int
	CabinClass     CabinClass
	Aircraft       *string  // nil → JSON null
	Amenities      []string // non-nil, possibly empty → JSON []
	Baggage        Baggage
}

// Endpoint is one end (departure or arrival) of a flight.
// Instant is always UTC. Offset preserves the original wall-clock timezone
// so filters and display can use local time without converting back from UTC.
type Endpoint struct {
	Airport  string
	City     string
	Instant  time.Time // canonical UTC instant
	Offset   string    // e.g. "+07:00" — for rendering local wall-clock time
	Terminal *string
}

// Duration holds a flight's elapsed time.
// Always constructed via NewDuration — never set directly.
type Duration struct {
	TotalMinutes int
	Formatted    string
}

// NewDuration computes a Duration from two UTC instants.
// Provider-supplied duration fields are never used; three fixtures contain
// values that contradict their own timestamps.
func NewDuration(dep, arr time.Time) Duration {
	mins := int(arr.Sub(dep).Minutes())
	return Duration{
		TotalMinutes: mins,
		Formatted:    formatDuration(mins),
	}
}

func formatDuration(mins int) string {
	h := mins / 60
	m := mins % 60
	if h == 0 {
		return strconv.Itoa(m) + "m"
	}
	return strconv.Itoa(h) + "h " + strconv.Itoa(m) + "m"
}

// Layover represents a stop between two flight segments.
type Layover struct {
	Airport string
	Minutes int
}

// Money represents a monetary amount in integer rupiah.
// float64 is banned from the money path — IDR has no minor unit and
// floating-point arithmetic introduces rounding errors.
type Money struct {
	Amount   int64  // integer rupiah
	Currency string // always "IDR"
}

// Baggage describes the allowance for a single passenger.
type Baggage struct {
	CarryOn string
	Checked string
}

// Airline identifies the operating carrier.
type Airline struct {
	Code string // IATA airline code, e.g. "GA"
	Name string
}

// CabinClass is the service class on the aircraft.
type CabinClass string

const (
	Economy        CabinClass = "economy"
	PremiumEconomy CabinClass = "premium_economy"
	Business       CabinClass = "business"
	First          CabinClass = "first"
)

// ParseCabinClass maps a raw provider string to a CabinClass.
// Unknown values return an error; callers should default to Economy and log.
func ParseCabinClass(s string) (CabinClass, error) {
	switch CabinClass(s) {
	case Economy, PremiumEconomy, Business, First:
		return CabinClass(s), nil
	default:
		return "", &ValidationError{Field: "cabin_class", Reason: "unknown value: " + s}
	}
}

// NewFlight returns a Flight with Amenities pre-set to a non-nil empty slice,
// which serializes as JSON [] rather than null.
func NewFlight() Flight {
	return Flight{Amenities: []string{}}
}
