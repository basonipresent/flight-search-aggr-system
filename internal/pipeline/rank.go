package pipeline

import "flight-search-aggr-system/internal/domain"

// Score attaches a best-value score to each flight.
// Scores are in [0, 1] where 1.0 is the best value in this result set.
type Score struct {
	Flight domain.Flight
	Value  float64
}

// Rank computes a weighted best-value score for each flight.
// Weights must sum to 1.0; defaults: price=0.5, duration=0.3, stops=0.2.
func Rank(flights []domain.Flight, wPrice, wDuration, wStops float64) []Score {
	if len(flights) == 0 {
		return nil
	}

	minPrice, maxPrice := flights[0].Price.Amount, flights[0].Price.Amount
	minDur, maxDur := flights[0].Duration.TotalMinutes, flights[0].Duration.TotalMinutes
	minStops, maxStops := flights[0].Stops, flights[0].Stops

	for _, f := range flights[1:] {
		if f.Price.Amount < minPrice {
			minPrice = f.Price.Amount
		}
		if f.Price.Amount > maxPrice {
			maxPrice = f.Price.Amount
		}
		if f.Duration.TotalMinutes < minDur {
			minDur = f.Duration.TotalMinutes
		}
		if f.Duration.TotalMinutes > maxDur {
			maxDur = f.Duration.TotalMinutes
		}
		if f.Stops < minStops {
			minStops = f.Stops
		}
		if f.Stops > maxStops {
			maxStops = f.Stops
		}
	}

	scores := make([]Score, len(flights))
	for i, f := range flights {
		scores[i] = Score{
			Flight: f,
			Value: wPrice*normalize(float64(f.Price.Amount), float64(minPrice), float64(maxPrice)) +
				wDuration*normalize(float64(f.Duration.TotalMinutes), float64(minDur), float64(maxDur)) +
				wStops*normalize(float64(f.Stops), float64(minStops), float64(maxStops)),
		}
	}
	return scores
}

// normalize return for the minimum value for the minimum value (best) and 0.0 for the maximum value (worst)
// Returns 0.5 when all values are equal (neutral)
func normalize(v float64, min, max float64) float64 {
	if max == min {
		return 0.5
	}
	return 1 - (v-min)/(max-min)
}
