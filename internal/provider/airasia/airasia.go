// Package airasia implements the provider.Adapter for AirAsia.
package airasia

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"

	"flight-search-aggr-system/internal/airport"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider"
	"flight-search-aggr-system/internal/timeutil"
)

const fixturePath = "testdata/airasia_search_response.json"

// Adapter reads from the embedded fixture and returns normalised AirAsia flights.
type Adapter struct {
	fs fs.ReadFileFS
}

// New creates an AirAsia Adapter.
func New(fsys fs.ReadFileFS) *Adapter {
	return &Adapter{fs: fsys}
}

// Name identifies this provider.
func (a *Adapter) Name() string { return "airasia" }

// Fetch parses the AirAsia fixture and returns normalised, validated flights.
func (a *Adapter) Fetch(_ context.Context, _ domain.SearchRequest) ([]domain.Flight, error) {
	data, err := a.fs.ReadFile(fixturePath)
	if err != nil {
		return nil, &provider.ProviderError{Provider: a.Name(), Kind: provider.KindTransport, Err: err}
	}

	var resp aaResponse
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

type aaResponse struct {
	Flights []aaFlight `json:"flights"`
}

type aaFlight struct {
	FlightCode  string  `json:"flight_code"`
	Airline     string  `json:"airline"`
	FromAirport string  `json:"from_airport"`
	ToAirport   string  `json:"to_airport"`
	DepartTime  string  `json:"depart_time"`
	ArriveTime  string  `json:"arrive_time"`
	DurationHrs  float64 `json:"duration_hours"`
	DirectFlight bool    `json:"direct_flight"`
	Stops        []struct {
		Airport     string `json:"airport"`
		WaitMinutes int    `json:"wait_time_minutes"`
	} `json:"stops"`
	PriceIDR    int64  `json:"price_idr"`
	Seats       int    `json:"seats"`
	CabinClass  string `json:"cabin_class"`
	BaggageNote string `json:"baggage_note"`
}

var baggageNotes = map[string]domain.Baggage{
	"Cabin baggage only, checked bags additional fee": {
		CarryOn: "Cabin baggage only",
		Checked: "Additional fee",
	},
}

func normalize(f aaFlight) (domain.Flight, error) {
	depTime, err := timeutil.ParseRFC3339(f.DepartTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("airasia dep: %w", err)
	}
	arrTime, err := timeutil.ParseRFC3339(f.ArriveTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("airasia arr: %w", err)
	}

	depAP, _ := airport.Lookup(f.FromAirport)
	arrAP, _ := airport.Lookup(f.ToAirport)

	fl := domain.NewFlight()
	fl.ID = f.FlightCode + "_airasia"
	fl.Provider = "airasia"
	fl.Airline = domain.Airline{
		Code: airlineCodeFromFlightCode(f.FlightCode),
		Name: f.Airline,
	}
	fl.FlightNumber = f.FlightCode
	fl.Departure = domain.Endpoint{
		Airport: f.FromAirport,
		City:    depAP.City,
		Instant: depTime.UTC(),
		Offset:  depTime.Format("-07:00"),
	}
	fl.Arrival = domain.Endpoint{
		Airport: f.ToAirport,
		City:    arrAP.City,
		Instant: arrTime.UTC(),
		Offset:  arrTime.Format("-07:00"),
	}
	fl.Duration = domain.NewDuration(fl.Departure.Instant, fl.Arrival.Instant)

	if f.DirectFlight {
		fl.Stops = 0
	} else {
		fl.Stops = len(f.Stops)
		fl.Layovers = make([]domain.Layover, 0, len(f.Stops))
		for _, s := range f.Stops {
			fl.Layovers = append(fl.Layovers, domain.Layover{
				Airport: s.Airport,
				Minutes: s.WaitMinutes,
			})
		}
	}

	fl.Baggage = parseBaggageNote(f.BaggageNote)
	fl.Price = domain.Money{Amount: f.PriceIDR, Currency: "IDR"}
	fl.AvailableSeats = f.Seats

	cc, err := domain.ParseCabinClass(f.CabinClass)
	if err != nil {
		cc = domain.Economy
	}
	fl.CabinClass = cc

	return fl, nil
}

func airlineCodeFromFlightCode(code string) string {
	for i, c := range code {
		if c >= '0' && c <= '9' {
			return code[:i]
		}
	}
	return code
}

func parseBaggageNote(note string) domain.Baggage {
	if b, ok := baggageNotes[note]; ok {
		return b
	}
	return domain.Baggage{CarryOn: note}
}
