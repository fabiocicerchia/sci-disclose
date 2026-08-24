package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// Fixtures for the carbon-intensity API: real responses, kept verbatim, and a
// server that serves one of them while counting what was asked for.

// A real /v1/last-hour/IE response, kept verbatim: the live API carries fields
// the documentation page does not show (zone, data_year, estimated,
// methodology, attribution), and the client must survive all of them.
func ReadingJSON(basis, generatedAt string) string {
	return fmt.Sprintf(`{
	  "generated_at": %q,
	  "country": "Ireland",
	  "country_code": "IE",
	  "zone": "IE",
	  "hour_start": "2026-08-21T14:30:00Z",
	  "hour_end": "2026-08-21T15:00:00Z",
	  "unit": "gCO2eq/kWh",
	  "direct": 249,
	  "lifecycle": 294,
	  "consumption_direct": 294,
	  "consumption_lifecycle": 339,
	  "basis": %q,
	  "data_source": {
	    "name": "ENTSO-E",
	    "url": "https://www.smartgriddashboard.com/",
	    "realtime": true,
	    "status": "operational",
	    "ref": null
	  },
	  "data_year": 2025,
	  "estimated": false,
	  "methodology": "Intensity computed by Carbon Intensity API from the data_source's published generation mix; IPCC AR6 lifecycle factors; ECON-PowerCI consumption accounting.",
	  "attribution": {
	    "name": "Carbon Intensity API",
	    "author": "Fabio Cicerchia",
	    "url": "https://ci-api.fabiocicerchia.it",
	    "repository": "https://github.com/fabiocicerchia/carbon-intensity-api",
	    "license": "AGPL-3.0-or-later"
	  }
	}`, generatedAt, basis)
}

// A real /v1/zones/IT/SICI response. Zone readings publish only the
// production-based pair: no consumption figures at all.
func ZoneReadingJSON(generatedAt string) string {
	return fmt.Sprintf(`{
	  "generated_at": %q,
	  "country": "Italy",
	  "country_code": "IT",
	  "zone": "SICI",
	  "hour_start": "2026-08-21T15:45:00Z",
	  "hour_end": "2026-08-21T16:00:00Z",
	  "unit": "gCO2eq/kWh",
	  "direct": 234,
	  "lifecycle": 279,
	  "basis": "measured",
	  "data_source": {"name": "ENTSO-E", "url": "https://transparency.entsoe.eu/",
	                  "realtime": true, "status": "operational", "ref": null},
	  "data_year": 2025,
	  "estimated": false,
	  "attribution": {"name": "Carbon Intensity API", "license": "AGPL-3.0-or-later"}
	}`, generatedAt)
}

// intensityServer serves one body and counts requests and paths.
func IntensityServer(t *testing.T, body string, status int) (*httptest.Server, *atomic.Int64, *[]string) {
	t.Helper()
	var hits atomic.Int64
	paths := &[]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		*paths = append(*paths, r.URL.Path)
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)
	t.Setenv("XDG_CACHE_HOME", t.TempDir()) // never touch the developer's real cache
	return server, &hits, paths
}

func Now() string { return time.Now().UTC().Format(time.RFC3339) }
