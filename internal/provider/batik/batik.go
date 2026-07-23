// Package batik implements the provider.Adapter for Batik Air.
package batik

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"flight-search-aggr-system/internal/airport"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider"
	"flight-search-aggr-system/internal/timeutil"
)

const fixturePath = "testdata/batik_air_search_response.json"

// Adapter reads from the embedded fixture and returns normalised Batik Air flights.
type Adapter struct {
	fs fs.ReadFileFS
}

// New creates a Batik Air Adapter.
func New(fsys fs.ReadFileFS) *Adapter {
	return &Adapter{fs: fsys}
}

// Name identifies this provider.
func (a *Adapter) Name() string { return "batik" }

// Fetch parses the Batik Air fixture and returns normalised, validated flights.
func (a *Adapter) Fetch(_ context.Context, _ domain.SearchRequest) ([]domain.Flight, error) {
	data, err := a.fs.ReadFile(fixturePath)
	if err != nil {
		return nil, &provider.ProviderError{Provider: a.Name(), Kind: provider.KindTransport, Err: err}
	}

	var resp batikResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, &provider.ProviderError{Provider: a.Name(), Kind: provider.KindDecode, Err: err}
	}

	flights := make([]domain.Flight, 0, len(resp.Results))
	for _, f := range resp.Results {
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

// ---- JSON structs ---------------------------------------------------------

type batikResponse struct {
	Results []batikFlight `json:"results"`
}

type batikFlight struct {
	FlightNumber    string `json:"flightNumber"`
	AirlineName     string `json:"airlineName"`
	AirlineIATA     string `json:"airlineIATA"`
	Origin          string `json:"origin"`
	Destination     string `json:"destination"`
	DepartureDateTime string `json:"departureDateTime"`
	ArrivalDateTime   string `json:"arrivalDateTime"`
	TravelTime        string `json:"travelTime"`
	NumberOfStops     int    `json:"numberOfStops"`
	Connections       []struct {
		StopAirport  string `json:"stopAirport"`
		StopDuration string `json:"stopDuration"` // e.g. "55m"
	} `json:"connections"`
	Fare struct {
		BasePrice  int64  `json:"basePrice"`
		Taxes      int64  `json:"taxes"`
		TotalPrice int64  `json:"totalPrice"`
		Currency   string `json:"currencyCode"`
		Class      string `json:"class"`
	} `json:"fare"`
	SeatsAvailable int    `json:"seatsAvailable"`
	AircraftModel  string `json:"aircraftModel"`
	BaggageInfo    string `json:"baggageInfo"`
	OnboardServices []string `json:"onboardServices"`
}

func normalize(f batikFlight) (domain.Flight, error) {
	depTime, err := timeutil.ParseCompactOffset(f.DepartureDateTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("batik dep: %w", err)
	}
	arrTime, err := timeutil.ParseCompactOffset(f.ArrivalDateTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("batik arr: %w", err)
	}

	depAP, _ := airport.Lookup(f.Origin)
	arrAP, _ := airport.Lookup(f.Destination)

	fl := domain.NewFlight()
	fl.ID = f.FlightNumber + "_batik"
	fl.Provider = "batik"
	fl.Airline = domain.Airline{Code: f.AirlineIATA, Name: f.AirlineName}
	fl.FlightNumber = f.FlightNumber
	fl.Departure = domain.Endpoint{
		Airport: f.Origin,
		City:    depAP.City,
		Instant: depTime.UTC(),
		Offset:  depTime.Format("-07:00"),
	}
	fl.Arrival = domain.Endpoint{
		Airport: f.Destination,
		City:    arrAP.City,
		Instant: arrTime.UTC(),
		Offset:  arrTime.Format("-07:00"),
	}
	fl.Duration = domain.NewDuration(fl.Departure.Instant, fl.Arrival.Instant)

	fl.Stops = f.NumberOfStops
	fl.Layovers = make([]domain.Layover, 0, len(f.Connections))
	for _, c := range f.Connections {
		mins, err := parseDurationString(c.StopDuration)
		if err != nil {
			continue
		}
		fl.Layovers = append(fl.Layovers, domain.Layover{
			Airport: c.StopAirport,
			Minutes: mins,
		})
	}

	fl.Price = domain.Money{Amount: f.Fare.TotalPrice, Currency: f.Fare.Currency}
	fl.AvailableSeats = f.SeatsAvailable

	ac := f.AircraftModel
	fl.Aircraft = &ac

	fl.Amenities = make([]string, len(f.OnboardServices))
	copy(fl.Amenities, f.OnboardServices)

	fl.Baggage = parseBaggageInfo(f.BaggageInfo)

	fl.CabinClass = parseBatikClass(f.Fare.Class)

	return fl, nil
}

func parseDurationString(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("parseDurationString %q: %w", s, err)
	}
	return int(d.Minutes()), nil
}

func parseBaggageInfo(s string) domain.Baggage {
	parts := strings.SplitN(s, ", ", 2)
	if len(parts) == 2 {
		return domain.Baggage{CarryOn: parts[0], Checked: parts[1]}
	}
	return domain.Baggage{CarryOn: s}
}

// parseBatikClass maps IATA booking class codes to CabinClass.
// "Y" is full-fare economy; "C"/"J" are business; "F" is first.
func parseBatikClass(s string) domain.CabinClass {
	switch s {
	case "Y", "B", "M", "H", "K", "Q", "V", "W", "G", "S", "L", "A", "T", "E":
		return domain.Economy
	case "C", "D", "I", "J", "Z":
		return domain.Business
	case "F", "P":
		return domain.First
	default:
		return domain.Economy
	}
}
