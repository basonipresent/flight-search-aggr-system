// Package airport provides IATA code lookups for city names and IANA timezones.
// It is used by adapters whose providers omit the city field (Batik Air, AirAsia).
package airport

// Airport holds the static metadata for an airport known to the system.
type Airport struct {
	IATA     string
	City     string
	Timezone string // IANA zone name, e.g. "Asia/Jakarta"
}

// airports is the seed table. Extend here when adding new routes.
var airports = map[string]Airport{
	"CGK": {IATA: "CGK", City: "Jakarta", Timezone: "Asia/Jakarta"},
	"DPS": {IATA: "DPS", City: "Denpasar", Timezone: "Asia/Makassar"},
	"SUB": {IATA: "SUB", City: "Surabaya", Timezone: "Asia/Jakarta"},
	"UPG": {IATA: "UPG", City: "Makassar", Timezone: "Asia/Makassar"},
	"SOC": {IATA: "SOC", City: "Solo", Timezone: "Asia/Jakarta"},
	"DJJ": {IATA: "DJJ", City: "Jayapura", Timezone: "Asia/Jayapura"},
	"NDA": {IATA: "NDA", City: "Ambon", Timezone: "Asia/Jayapura"},
}

// Lookup returns the Airport for the given IATA code.
// The second return value is false when the code is unknown;
// callers should degrade to an empty city string rather than erroring.
func Lookup(iata string) (Airport, bool) {
	ap, ok := airports[iata]
	return ap, ok
}
