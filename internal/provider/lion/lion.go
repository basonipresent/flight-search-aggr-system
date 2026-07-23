// Package lion implements the provider.Adapter for Lion Air.
package lion

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"

	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/provider"
	"flight-search-aggr-system/internal/timeutil"
)

const fixturePath = "testdata/lion_air_search_response.json"

// Adapter reads from the embedded fixture and returns normalised Lion Air flights.
type Adapter struct {
	fs fs.ReadFileFS
}

// New creates a Lion Air Adapter.
func New(fsys fs.ReadFileFS) *Adapter {
	return &Adapter{fs: fsys}
}

// Name identifies this provider.
func (a *Adapter) Name() string { return "lion" }

// Fetch parses the Lion Air fixture and returns normalised, validated flights.
func (a *Adapter) Fetch(_ context.Context, _ domain.SearchRequest) ([]domain.Flight, error) {
	data, err := a.fs.ReadFile(fixturePath)
	if err != nil {
		return nil, &provider.ProviderError{Provider: a.Name(), Kind: provider.KindTransport, Err: err}
	}

	var resp lionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, &provider.ProviderError{Provider: a.Name(), Kind: provider.KindDecode, Err: err}
	}

	flights := make([]domain.Flight, 0, len(resp.Data.Flights))
	for _, f := range resp.Data.Flights {
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

type lionResponse struct {
	Data struct {
		Flights []lionFlight `json:"available_flights"`
	} `json:"data"`
}

type lionFlight struct {
	ID      string `json:"id"`
	Carrier struct {
		Name string `json:"name"`
		IATA string `json:"iata"`
	} `json:"carrier"`
	Route struct {
		From lionRouteEnd `json:"from"`
		To   lionRouteEnd `json:"to"`
	} `json:"route"`
	Schedule struct {
		Departure        string `json:"departure"`
		DepartureTZ      string `json:"departure_timezone"`
		Arrival          string `json:"arrival"`
		ArrivalTZ        string `json:"arrival_timezone"`
	} `json:"schedule"`
	FlightTime int  `json:"flight_time"`
	IsDirect   bool `json:"is_direct"`
	StopCount  int  `json:"stop_count"`
	Layovers   []struct {
		Airport string `json:"airport"`
		Minutes int    `json:"duration_minutes"`
	} `json:"layovers"`
	Pricing struct {
		Total    int64  `json:"total"`
		Currency string `json:"currency"`
		FareType string `json:"fare_type"`
	} `json:"pricing"`
	SeatsLeft int    `json:"seats_left"`
	PlaneType string `json:"plane_type"`
	Services  struct {
		WifiAvailable  bool `json:"wifi_available"`
		MealsIncluded  bool `json:"meals_included"`
		BaggageAllow   struct {
			Cabin string `json:"cabin"`
			Hold  string `json:"hold"`
		} `json:"baggage_allowance"`
	} `json:"services"`
}

type lionRouteEnd struct {
	Code string `json:"code"`
	City string `json:"city"`
}

func normalize(f lionFlight) (domain.Flight, error) {
	depTime, err := timeutil.ParseInZone(f.Schedule.Departure, f.Schedule.DepartureTZ)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("lion dep: %w", err)
	}
	arrTime, err := timeutil.ParseInZone(f.Schedule.Arrival, f.Schedule.ArrivalTZ)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("lion arr: %w", err)
	}

	fl := domain.NewFlight()
	fl.ID = f.ID + "_lion"
	fl.Provider = "lion"
	fl.Airline = domain.Airline{Code: f.Carrier.IATA, Name: f.Carrier.Name}
	fl.FlightNumber = f.ID
	fl.Departure = domain.Endpoint{
		Airport: f.Route.From.Code,
		City:    f.Route.From.City,
		Instant: depTime.UTC(),
		Offset:  depTime.Format("-07:00"),
	}
	fl.Arrival = domain.Endpoint{
		Airport: f.Route.To.Code,
		City:    f.Route.To.City,
		Instant: arrTime.UTC(),
		Offset:  arrTime.Format("-07:00"),
	}
	fl.Duration = domain.NewDuration(fl.Departure.Instant, fl.Arrival.Instant)

	if f.IsDirect {
		fl.Stops = 0
	} else {
		fl.Stops = f.StopCount
		fl.Layovers = make([]domain.Layover, 0, len(f.Layovers))
		for _, l := range f.Layovers {
			fl.Layovers = append(fl.Layovers, domain.Layover{
				Airport: l.Airport,
				Minutes: l.Minutes,
			})
		}
	}

	pl := f.PlaneType
	fl.Aircraft = &pl

	if f.Services.WifiAvailable {
		fl.Amenities = append(fl.Amenities, "wifi")
	}
	if f.Services.MealsIncluded {
		fl.Amenities = append(fl.Amenities, "meal")
	}

	fl.Baggage = domain.Baggage{
		CarryOn: f.Services.BaggageAllow.Cabin,
		Checked: f.Services.BaggageAllow.Hold,
	}

	fl.Price = domain.Money{Amount: f.Pricing.Total, Currency: f.Pricing.Currency}
	fl.AvailableSeats = f.SeatsLeft

	fl.CabinClass = parseFareType(f.Pricing.FareType)

	return fl, nil
}

func parseFareType(s string) domain.CabinClass {
	mapping := map[string]domain.CabinClass{
		"ECONOMY":         domain.Economy,
		"PREMIUM_ECONOMY": domain.PremiumEconomy,
		"BUSINESS":        domain.Business,
		"FIRST":           domain.First,
	}
	if cc, ok := mapping[s]; ok {
		return cc
	}
	return domain.Economy
}
