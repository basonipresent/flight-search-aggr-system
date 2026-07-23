package pipeline

import (
	"flight-search-aggr-system/internal/domain"
	"sort"
)

// Sort orders flights by the given SortKey using stable sort so equals keys produce deterministics output.
func Sort(flights []domain.Flight, key domain.SortKey) {
	sort.SliceStable(flights, comparator(flights, key))
}

func comparator(flights []domain.Flight, key domain.SortKey) func(i, j int) bool {
	switch key {
	case domain.SortPriceAsc:
		return func(i, j int) bool {
			return flights[i].Price.Amount < flights[j].Price.Amount
		}
	case domain.SortPriceDesc:
		return func(i, j int) bool {
			return flights[i].Price.Amount > flights[j].Price.Amount
		}
	case domain.SortDurationAsc:
		return func(i, j int) bool {
			return flights[i].Duration.TotalMinutes < flights[j].Duration.TotalMinutes
		}
	case domain.SortDurationDesc:
		return func(i, j int) bool {
			return flights[i].Duration.TotalMinutes > flights[j].Duration.TotalMinutes
		}
	case domain.SortDepartureAsc:
		return func(i, j int) bool {
			return flights[i].Departure.Instant.Before(flights[j].Departure.Instant)
		}
	case domain.SortDepartureDesc:
		return func(i, j int) bool {
			return flights[i].Departure.Instant.After(flights[j].Departure.Instant)
		}
	case domain.SortArrivalAsc:
		return func(i, j int) bool {
			return flights[i].Arrival.Instant.Before(flights[j].Arrival.Instant)
		}
	case domain.SortArrivalDesc:
		return func(i, j int) bool {
			return flights[i].Arrival.Instant.After(flights[j].Arrival.Instant)
		}
	default:
		return func(i, j int) bool {
			return flights[i].Price.Amount < flights[j].Price.Amount
		}
	}
}
