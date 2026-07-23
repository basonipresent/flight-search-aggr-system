package airport

import "testing"

func TestLookup(t *testing.T) {
	tests := []struct {
		iata     string
		wantCity string
		wantTZ   string
		wantOK   bool
	}{
		{"CGK", "Jakarta", "Asia/Jakarta", true},
		{"DPS", "Denpasar", "Asia/Makassar", true}, // WITA, not WIB — different from CGK
		{"SUB", "Surabaya", "Asia/Jakarta", true},
		{iata: "UPG", wantCity: "Makassar", wantTZ: "Asia/Makassar", wantOK: true},
		{"SOC", "Solo", "Asia/Jakarta", true},
		{"DJJ", "Jayapura", "Asia/Jayapura", true},
		{"NDA", "Ambon", "Asia/Jayapura", true},
		{"XXX", "", "", false}, // unknown — must not error
		{"", "", "", false},    // empty string
	}

	for _, tt := range tests {
		t.Run(tt.iata, func(t *testing.T) {
			got, ok := Lookup(tt.iata)
			if ok != tt.wantOK {
				t.Fatalf("Lookup(%q) ok = %v, want %v", tt.iata, ok, tt.wantOK)
			}
			if !ok {
				return // zero-value Airport is fine for unknown codes
			}
			if got.City != tt.wantCity {
				t.Errorf("City = %q, want %q", got.City, tt.wantCity)
			}
			if got.Timezone != tt.wantTZ {
				t.Errorf("Timezone = %q, want %q", got.Timezone, tt.wantTZ)
			}
		})
	}
}
