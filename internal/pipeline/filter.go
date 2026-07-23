package pipeline

import (
	"flight-search-aggr-system/internal/domain"
	"time"
)

// Predicate is a filter function over a single flight.
type Predicate func(domain.Flight) bool

// Apply runs all predicates in a single pass, returning only flights that satisfy every predicate.
func Apply(flights []domain.Flight, predicates []Predicate) []domain.Flight {
	out := make([]domain.Flight, 0, len(flights))
	for _, f := range flights {
		pass := true
		for _, p := range predicates {
			if !p(f) {
				pass = false
				break
			}
		}
		if pass {
			out = append(out, f)
		}
	}
	return out
}

// BuildPredicates converts a FIlters struct into a slice of Predicate.
func BuildPredicates(f domain.Filters) []Predicate {
	var ps []Predicate

	if f.PriceMin > 0 {
		min := f.PriceMin
		ps = append(ps, func(fl domain.Flight) bool {
			return fl.Price.Amount >= min
		})
	}
	if f.PriceMax > 0 {
		max := f.PriceMax
		ps = append(ps, func(fl domain.Flight) bool {
			return fl.Price.Amount <= max
		})
	}
	if f.MaxStops > 0 && f.MaxStops != -1 {
		ms := f.MaxStops
		ps = append(ps, func(fl domain.Flight) bool {
			return fl.Stops <= ms
		})
	}
	if f.MaxDurationMinutes > 0 {
		md := f.MaxDurationMinutes
		ps = append(ps, func(fl domain.Flight) bool {
			return fl.Duration.TotalMinutes <= md
		})
	}
	if len(f.Airlines) > 0 {
		set := make(map[string]struct{}, len(f.Airlines))
		for _, a := range f.Airlines {
			set[a] = struct{}{}
		}
		ps = append(ps, func(fl domain.Flight) bool {
			_, ok := set[fl.Airline.Code]
			return ok
		})
	}
	if f.DepartureAfter != "" {
		after := f.DepartureAfter
		ps = append(ps, func(fl domain.Flight) bool {
			return localWalkClock(fl.Departure) >= after
		})
	}
	if f.DepartureBefore != "" {
		before := f.DepartureBefore
		ps = append(ps, func(fl domain.Flight) bool {
			return localWalkClock(fl.Departure) <= before
		})
	}
	if f.ArrivalAfter != "" {
		after := f.ArrivalAfter
		ps = append(ps, func(fl domain.Flight) bool {
			return localWalkClock(fl.Arrival) >= after
		})
	}
	if f.ArrivalBefore != "" {
		before := f.ArrivalBefore
		ps = append(ps, func(fl domain.Flight) bool {
			return localWalkClock(fl.Arrival) <= before
		})
	}

	return ps
}

// localWalkClock returns "HH:MM" in the endpoint's local timezone.
func localWalkClock(ep domain.Endpoint) string {
	loc := time.FixedZone("", offsetSeconds(ep.Offset))
	local := ep.Instant.In(loc)
	return local.Format("15:04")
}

// offsetSeconds converts "+07:00" or "-05:30" to a seconds integer.
func offsetSeconds(offset string) int {
	if len(offset) != 6 {
		return 0
	}
	sign := 1
	if offset[0] == '-' {
		sign = -1
	}
	h := int(offset[1]-'0')*10 + int(offset[2]-'0')
	m := int(offset[4]-'0')*10 + int(offset[5]-'0')
	return sign * (h*3600 + m*60)
}
