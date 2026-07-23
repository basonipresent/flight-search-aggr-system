package aggregator

import (
	"flight-search-aggr-system/internal/domain"
	"fmt"
)

// Deduplicate collapses flights with the same airlines, number, and departure time, keeping the cheapest.
func Deduplicate(flights []domain.Flight) []domain.Flight {
	index := make(map[string]int, len(flights))
	out := make([]domain.Flight, 0, len(flights))

	for _, f := range flights {
		key := fmt.Sprintf("%s|%s|%d", f.Airline.Code, f.FlightNumber, f.Departure.Instant.Unix())
		if i, exists := index[key]; exists {
			if f.Price.Amount < out[i].Price.Amount {
				out[i] = f
			}
			continue
		}
		index[key] = len(out)
		out = append(out, f)
	}
	return out
}
