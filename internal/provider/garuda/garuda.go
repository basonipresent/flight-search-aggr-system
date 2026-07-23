// Package garuda implements the provider.Adapter for Garuda Indonesia.
package garuda

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"

	"flight-search-aggr-system/internal/airport"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider"
	"flight-search-aggr-system/internal/timeutil"
)

const fixturePath = "testdata/garuda_indonesia_search_response.json"

// Adapter reads from the embedded fixture and returns normalised Garuda flights.
type Adapter struct {
	fs fs.ReadFileFS
}

// New creates a Garuda Adapter reading from the given filesystem.
func New(fsys fs.ReadFileFS) *Adapter {
	return &Adapter{fs: fsys}
}

// Name identifies this provider.
func (a *Adapter) Name() string { return "garuda" }

// Fetch parses the Garuda fixture and returns normalised, validated flights.
func (a *Adapter) Fetch(_ context.Context, _ domain.SearchRequest) ([]domain.Flight, error) {
	data, err := a.fs.ReadFile(fixturePath)
	if err != nil {
		return nil, &provider.ProviderError{Provider: a.Name(), Kind: provider.KindTransport, Err: err}
	}

	var resp garudaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, &provider.ProviderError{Provider: a.Name(), Kind: provider.KindDecode, Err: err}
	}

	flights := make([]domain.Flight, 0, len(resp.Flights))
	for _, f := range resp.Flights {
		fl, err := normalize(f)
		if err != nil {
			continue
		}
		if err := fl.Validate(); err != nil {
			continue
		}
		flights = append(flights, fl)
	}
	return flights, nil
}

type garudaResponse struct {
	Flights []garudaFlight `json:"flights"`
}

type garudaFlight struct {
	FlightID        string          `json:"flight_id"`
	Airline         string          `json:"airline"`
	AirlineCode     string          `json:"airline_code"`
	Departure       garudaEndpoint  `json:"departure"`
	Arrival         garudaEndpoint  `json:"arrival"`
	DurationMinutes int             `json:"duration_minutes"`
	Stops           int             `json:"stops"`
	Aircraft        string          `json:"aircraft"`
	Price           garudaPrice     `json:"price"`
	AvailableSeats  int             `json:"available_seats"`
	FareClass       string          `json:"fare_class"`
	Baggage         garudaBaggage   `json:"baggage"`
	Amenities       []string        `json:"amenities"`
	Segments        []garudaSegment `json:"segments"`
}

type garudaEndpoint struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type garudaPrice struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type garudaBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}

type garudaSegment struct {
	Departure garudaSegEndpoint `json:"departure"`
	Arrival   garudaSegEndpoint `json:"arrival"`
}

type garudaSegEndpoint struct {
	Airport string `json:"airport"`
	Time    string `json:"time"`
}

func normalize(f garudaFlight) (domain.Flight, error) {
	fl := domain.NewFlight()
	fl.ID = f.FlightID + "_garuda"
	fl.Provider = "garuda"
	fl.Airline = domain.Airline{Code: f.AirlineCode, Name: f.Airline}
	fl.FlightNumber = f.FlightID
	fl.Price = domain.Money{Amount: f.Price.Amount, Currency: f.Price.Currency}
	fl.AvailableSeats = f.AvailableSeats

	if f.Aircraft != "" {
		ac := f.Aircraft
		fl.Aircraft = &ac
	}

	if len(f.Amenities) > 0 {
		fl.Amenities = f.Amenities
	}

	fl.Baggage = domain.Baggage{
		CarryOn: strconv.Itoa(f.Baggage.CarryOn) + " piece",
		Checked: strconv.Itoa(f.Baggage.Checked) + " pieces",
	}

	cc, err := domain.ParseCabinClass(f.FareClass)
	if err != nil {
		cc = domain.Economy
	}
	fl.CabinClass = cc

	if len(f.Segments) > 0 {
		return normalizeWithSegments(fl, f)
	}
	return normalizeTopLevel(fl, f)
}

// normalizeWithSegments uses the segments array as the authoritative source.
// GA315's top-level arrival, stops, and duration_minutes describe only the
// first leg and are incorrect for the full journey.
func normalizeWithSegments(fl domain.Flight, f garudaFlight) (domain.Flight, error) {
	first := f.Segments[0]
	last := f.Segments[len(f.Segments)-1]

	depTime, err := timeutil.ParseRFC3339(first.Departure.Time)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("garuda segment[0] dep: %w", err)
	}
	arrTime, err := timeutil.ParseRFC3339(last.Arrival.Time)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("garuda segment[last] arr: %w", err)
	}

	depAP, _ := airport.Lookup(first.Departure.Airport)
	arrAP, _ := airport.Lookup(last.Arrival.Airport)

	fl.Departure = domain.Endpoint{
		Airport: first.Departure.Airport,
		City:    depAP.City,
		Instant: depTime.UTC(),
		Offset:  depTime.Format("-07:00"),
	}
	fl.Arrival = domain.Endpoint{
		Airport: last.Arrival.Airport,
		City:    arrAP.City,
		Instant: arrTime.UTC(),
		Offset:  arrTime.Format("-07:00"),
	}
	fl.Stops = len(f.Segments) - 1

	// Layover duration = gap between segment[i-1].arrival and segment[i].departure.
	fl.Layovers = make([]domain.Layover, 0, len(f.Segments)-1)
	for i := 1; i < len(f.Segments); i++ {
		prevArr, err := timeutil.ParseRFC3339(f.Segments[i-1].Arrival.Time)
		if err != nil {
			return domain.Flight{}, fmt.Errorf("garuda segment[%d] arr: %w", i-1, err)
		}
		currDep, err := timeutil.ParseRFC3339(f.Segments[i].Departure.Time)
		if err != nil {
			return domain.Flight{}, fmt.Errorf("garuda segment[%d] dep: %w", i, err)
		}
		fl.Layovers = append(fl.Layovers, domain.Layover{
			Airport: f.Segments[i-1].Arrival.Airport,
			Minutes: int(currDep.Sub(prevArr).Minutes()),
		})
	}

	fl.Duration = domain.NewDuration(fl.Departure.Instant, fl.Arrival.Instant)
	return fl, nil
}

func normalizeTopLevel(fl domain.Flight, f garudaFlight) (domain.Flight, error) {
	depTime, err := timeutil.ParseRFC3339(f.Departure.Time)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("garuda dep: %w", err)
	}
	arrTime, err := timeutil.ParseRFC3339(f.Arrival.Time)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("garuda arr: %w", err)
	}

	var depTerm, arrTerm *string
	if f.Departure.Terminal != "" {
		t := f.Departure.Terminal
		depTerm = &t
	}
	if f.Arrival.Terminal != "" {
		t := f.Arrival.Terminal
		arrTerm = &t
	}

	fl.Departure = domain.Endpoint{
		Airport:  f.Departure.Airport,
		City:     f.Departure.City,
		Instant:  depTime.UTC(),
		Offset:   depTime.Format("-07:00"),
		Terminal: depTerm,
	}
	fl.Arrival = domain.Endpoint{
		Airport:  f.Arrival.Airport,
		City:     f.Arrival.City,
		Instant:  arrTime.UTC(),
		Offset:   arrTime.Format("-07:00"),
		Terminal: arrTerm,
	}
	fl.Stops = f.Stops
	fl.Duration = domain.NewDuration(fl.Departure.Instant, fl.Arrival.Instant)
	return fl, nil
}
