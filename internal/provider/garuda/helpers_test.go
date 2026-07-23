package garuda_test

import "flight-search-aggr-system/internal/domain"

func testRequest() domain.SearchRequest {
	return domain.SearchRequest{
		Origin:      "CGK",
		Destination: "DPS",
		Passengers:  1,
		CabinClass:  domain.Economy,
	}
}
