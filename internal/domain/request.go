package domain

import "time"

// SearchRequest is the validated, decoded search parameters from the HTTP handler.
type SearchRequest struct {
	Origin        string
	Destination   string
	DepartureDate time.Time
	ReturnDate    *time.Time // nil for one-way
	Passengers    int
	CabinClass    CabinClass
	Filters       Filters
	Sort          SortKey
}

// Filters holds the optional post-aggregation filter criteria.
// Zero values mean "no constraint".
type Filters struct {
	PriceMin           int64
	PriceMax           int64    // 0 = unbounded
	MaxStops           int      // -1 = unbounded
	DepartureAfter     string   // "HH:MM" local wall-clock
	DepartureBefore    string   // "HH:MM" local wall-clock
	ArrivalAfter       string
	ArrivalBefore      string
	Airlines           []string // empty = all airlines
	MaxDurationMinutes int      // 0 = unbounded
}

// SortKey selects the comparator used when ordering results.
type SortKey string

const (
	SortPriceAsc      SortKey = "price_asc"
	SortPriceDesc     SortKey = "price_desc"
	SortDurationAsc   SortKey = "duration_asc"
	SortDurationDesc  SortKey = "duration_desc"
	SortDepartureAsc  SortKey = "departure_asc"
	SortDepartureDesc SortKey = "departure_desc"
	SortArrivalAsc    SortKey = "arrival_asc"
	SortArrivalDesc   SortKey = "arrival_desc"
	SortBestValue     SortKey = "best_value"
)
