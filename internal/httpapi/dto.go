package httpapi

import "flight-search-aggr-system/internal/domain"

// SearchRequest is the decoded HTTP request body.
type SearchRequest struct {
	Origin        string            `json:"origin"`
	Destination   string            `json:"destination"`
	DepartureDate string            `json:"departureDate"`
	ReturnDate    *string           `json:"returnDate"`
	Passengers    int               `json:"passengers"`
	CabinClass    string            `json:"cabinClass"`
	Filters       FiltersDTO        `json:"filters"`
	Sort          string            `json:"sort"`
}

type FiltersDTO struct {
	MaxStops           *int     `json:"maxStops"`
	PriceMin           *int64   `json:"priceMin"`
	PriceMax           *int64   `json:"priceMax"`
	DepartureAfter     string   `json:"departureAfter"`
	DepartureBefore    string   `json:"departureBefore"`
	ArrivalAfter       string   `json:"arrivalAfter"`
	ArrivalBefore      string   `json:"arrivalBefore"`
	Airlines           []string `json:"airlines"`
	MaxDurationMinutes *int     `json:"maxDurationMinutes"`
}

// SearchResponse is the HTTP response body.
type SearchResponse struct {
	Metadata MetadataDTO `json:"metadata"`
	Flights  []FlightDTO `json:"flights"`
}

type MetadataDTO struct {
	TotalResults    int      `json:"total_results"`
	CacheHit        bool     `json:"cache_hit"`
	ProvidersFailed int      `json:"providers_failed"`
	Warnings        []string `json:"warnings"`
}

type FlightDTO struct {
	ID           string       `json:"id"`
	Provider     string       `json:"provider"`
	Airline      AirlineDTO   `json:"airline"`
	FlightNumber string       `json:"flight_number"`
	Departure    EndpointDTO  `json:"departure"`
	Arrival      EndpointDTO  `json:"arrival"`
	Duration     DurationDTO  `json:"duration"`
	Stops        int          `json:"stops"`
	Layovers     []LayoverDTO `json:"layovers"`
	Price        PriceDTO     `json:"price"`
	Seats        int          `json:"available_seats"`
	CabinClass   string       `json:"cabin_class"`
	Aircraft     *string      `json:"aircraft"`  // no omitempty - null when absent
	Amenities    []string     `json:"amenities"` // no omitempty - [] when absent
	Baggage      BaggageDTO   `json:"baggage"`
}

type AirlineDTO struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type EndpointDTO struct {
	Airport  string  `json:"airport"`
	City     string  `json:"city"`
	Time     string  `json:"time"`
	Offset   string  `json:"offset"`
	Terminal *string `json:"terminal"`
}

type DurationDTO struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type LayoverDTO struct {
	Airport string `json:"airport"`
	Minutes int    `json:"minutes"`
}

type PriceDTO struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type BaggageDTO struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

// toFlightDTO converts a domain.Flight to its wire representation.
func toFlightDTO(f domain.Flight) FlightDTO {
	layovers := make([]LayoverDTO, len(f.Layovers))
	for i, layover := range f.Layovers {
		layovers[i] = LayoverDTO{
			Airport: layover.Airport,
			Minutes: layover.Minutes,
		}
	}
	return FlightDTO{
		ID:       f.ID,
		Provider: f.Provider,
		Airline: AirlineDTO{
			Code: f.Airline.Code,
			Name: f.Airline.Name,
		},
		FlightNumber: f.FlightNumber,
		Departure: EndpointDTO{
			Airport:  f.Departure.Airport,
			City:     f.Departure.City,
			Time:     f.Departure.Instant.Format("2006-01-02T15:04:05") + f.Departure.Offset,
			Offset:   f.Departure.Offset,
			Terminal: f.Departure.Terminal,
		},
		Arrival: EndpointDTO{
			Airport:  f.Arrival.Airport,
			City:     f.Arrival.City,
			Time:     f.Arrival.Instant.Format("2006-01-02T15:04:05") + f.Arrival.Offset,
			Offset:   f.Arrival.Offset,
			Terminal: f.Arrival.Terminal,
		},
		Duration: DurationDTO{
			TotalMinutes: f.Duration.TotalMinutes,
			Formatted:    f.Duration.Formatted,
		},
		Stops:    f.Stops,
		Layovers: layovers,
		Price: PriceDTO{
			Amount:   f.Price.Amount,
			Currency: f.Price.Currency,
		},
		Seats:      f.AvailableSeats,
		CabinClass: string(f.CabinClass),
		Aircraft:   f.Aircraft,
		Amenities:  f.Amenities,
		Baggage: BaggageDTO{
			CarryOn: f.Baggage.CarryOn,
			Checked: f.Baggage.Checked,
		},
	}
}
