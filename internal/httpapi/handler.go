package httpapi

import (
	"encoding/json"
	"errors"
	"flight-search-aggr-system/internal/aggregator"
	"flight-search-aggr-system/internal/cache"
	"flight-search-aggr-system/internal/domain"
	"flight-search-aggr-system/internal/pipeline"
	"flight-search-aggr-system/internal/provider"
	"net/http"
	"time"
)

// Handler handles POST /api/v1/flights/search.
type Handler struct {
	agg   *aggregator.Aggregator
	group *cache.SearchGroup
	cfg   Config
}

// Config holds handler-level configuration.
type Config struct {
	ScoreWeightPrice    float64
	ScoreWeightDuration float64
	ScoreWeightStops    float64
}

// NewHandler creates a new Handler.
func NewHandler(agg *aggregator.Aggregator, sg *cache.SearchGroup, cfg Config) *Handler {
	return &Handler{
		agg:   agg,
		group: sg,
		cfg:   cfg,
	}
}

// ServeHTTP handles the search endpoint.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	domainReq, err := toDomainRequest(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	flights, cacheHit, err := h.group.Do(domainReq, func() ([]domain.Flight, error) {
		res, err := h.agg.Search(r.Context(), domainReq)
		if err != nil {
			return nil, err
		}
		return res.Flights, nil
	})
	if err != nil {
		if errors.Is(err, provider.ErrAllProvidersFailed) {
			writeError(w, http.StatusBadGateway, "all providers failed")
			return
		}
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	// Dedupe -> filter -> sort -> rank
	flights = aggregator.Deduplicate(flights)

	preds := pipeline.BuildPredicates(domainReq.Filters)
	flights = pipeline.Apply(flights, preds)

	sortKey := domainReq.Sort
	if sortKey == "" || sortKey == domain.SortBestValue {
		scores := pipeline.Rank(flights, h.cfg.ScoreWeightPrice, h.cfg.ScoreWeightDuration, h.cfg.ScoreWeightStops)
		ranked := make([]domain.Flight, len(scores))
		for i, s := range scores {
			ranked[i] = s.Flight
		}
		flights = ranked
	} else {
		pipeline.Sort(flights, sortKey)
	}

	flightDTOs := make([]FlightDTO, len(flights))
	for i, f := range flights {
		flightDTOs[i] = toFlightDTO(f)
	}

	writeJSON(w, http.StatusOK, SearchResponse{
		Metadata: MetadataDTO{
			TotalResults: len(flights),
			CacheHit:     cacheHit,
		},
		Flights: flightDTOs,
	})
}

func toDomainRequest(r SearchRequest) (domain.SearchRequest, error) {
	if len(r.Origin) != 3 || len(r.Destination) != 3 {
		return domain.SearchRequest{}, errors.New("origin and destination must be 3-letter IATA codes")
	}
	if r.Origin == r.Destination {
		return domain.SearchRequest{}, errors.New("origin and destination must differ")
	}
	if r.Passengers < 1 || r.Passengers > 9 {
		return domain.SearchRequest{}, errors.New("passengers must be between 1 and 9")
	}

	date, err := time.Parse("2006-01-02", r.DepartureDate)
	if err != nil {
		return domain.SearchRequest{}, errors.New("departureDate must be YYYY-MM-DD")
	}
	if date.Before(time.Now().Truncate(24 * time.Hour)) {
		return domain.SearchRequest{}, errors.New("departureDate must not be in the past")
	}

	cc, err := domain.ParseCabinClass(r.CabinClass)
	if err != nil {
		return domain.SearchRequest{}, errors.New("unknown cabinClass")
	}

	f := domain.Filters{MaxStops: -1}
	if r.Filters.MaxStops != nil {
		f.MaxStops = *r.Filters.MaxStops
	}
	if r.Filters.PriceMin != nil {
		f.PriceMin = *r.Filters.PriceMin
	}
	if r.Filters.PriceMax != nil {
		f.PriceMax = *r.Filters.PriceMax
	}
	if r.Filters.MaxDurationMinutes != nil {
		f.MaxDurationMinutes = *r.Filters.MaxDurationMinutes
	}
	f.DepartureAfter = r.Filters.DepartureAfter
	f.DepartureBefore = r.Filters.DepartureBefore
	f.ArrivalAfter = r.Filters.ArrivalAfter
	f.ArrivalBefore = r.Filters.ArrivalBefore
	f.Airlines = r.Filters.Airlines

	sortKey := domain.SortBestValue
	if r.Sort != "" {
		sortKey = domain.SortKey(r.Sort)
	}

	return domain.SearchRequest{
		Origin:        r.Origin,
		Destination:   r.Destination,
		DepartureDate: date,
		Passengers:    r.Passengers,
		CabinClass:    cc,
		Filters:       f,
		Sort:          sortKey,
	}, nil
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
