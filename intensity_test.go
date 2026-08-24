package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A real /v1/last-hour/IE response, kept verbatim: the live API carries fields
// the documentation page does not show (zone, data_year, estimated,
// methodology, attribution), and the client must survive all of them.
func readingJSON(basis, generatedAt string) string {
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
func zoneReadingJSON(generatedAt string) string {
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
func intensityServer(t *testing.T, body string, status int) (*httptest.Server, *atomic.Int64, *[]string) {
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

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func TestExplicitIntensityBeatsEveryLookup(t *testing.T) {
	server, hits, _ := intensityServer(t, readingJSON("measured", now()), http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Country, cfg.Intensity = server.URL, "DE", 42
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if intensity.Value != 42 || hits.Load() != 0 {
		t.Fatalf("got %v after %d requests", intensity, hits.Load())
	}
}

func TestLastHourReadingIsUsedWithItsProvenance(t *testing.T) {
	server, _, paths := intensityServer(t, readingJSON("measured", now()), http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Region = server.URL, "eu-west-1" // Dublin -> IE
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if (*paths)[0] != "/v1/last-hour/IE" {
		t.Errorf("path: %s", (*paths)[0])
	}
	// consumption_lifecycle by default: the figure to report a footprint with.
	if intensity.Value != 339 || intensity.Basis != "consumption_lifecycle" {
		t.Errorf("value %g basis %q", intensity.Value, intensity.Basis)
	}
	if !intensity.Measured || intensity.Stale || intensity.Estimated {
		t.Errorf("a fresh measured reading is neither modelled nor stale: %+v", intensity)
	}
	if intensity.RequestedBasis != "" {
		t.Errorf("nothing fell back: %q", intensity.RequestedBasis)
	}
	// The window is the operator's settlement period, not always a whole hour.
	if intensity.Window != "2026-08-21T14:30:00Z to 2026-08-21T15:00:00Z" {
		t.Errorf("window: %q", intensity.Window)
	}
	if intensity.DataYear != 2025 || intensity.Methodology == "" {
		t.Errorf("the reading's own metadata was dropped: %+v", intensity)
	}
	for _, want := range []string{"Ireland", "ENTSO-E", "14:30:00Z", "measured"} {
		if !strings.Contains(intensity.Source, want) {
			t.Errorf("provenance %q is missing %q", intensity.Source, want)
		}
	}
	// The API is AGPL and asks to be credited; a disclosure quoting it must say so.
	credit := intensity.Credit()
	for _, want := range []string{"Carbon Intensity API", "AGPL-3.0-or-later",
		"ENTSO-E", "smartgriddashboard"} {
		if !strings.Contains(credit, want) {
			t.Errorf("credit %q is missing %q", credit, want)
		}
	}
}

func TestIntensityBasisIsSelectable(t *testing.T) {
	server, _, _ := intensityServer(t, readingJSON("measured", now()), http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Country, cfg.IntensityBasis = server.URL, "IE", "direct"
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if intensity.Value != 249 {
		t.Fatalf("direct: %g", intensity.Value)
	}
}

func TestZoneReadingsFallBackThroughTheBases(t *testing.T) {
	server, _, paths := intensityServer(t, zoneReadingJSON(now()), http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Zone = server.URL, "IT/SICI"
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if (*paths)[0] != "/v1/zones/IT/SICI" {
		t.Errorf("path: %s", (*paths)[0])
	}
	// SICI publishes no consumption figures, so the default basis cannot apply.
	if intensity.Value != 279 || intensity.Basis != "lifecycle" {
		t.Errorf("expected the lifecycle figure, got %g via %q", intensity.Value, intensity.Basis)
	}
	if intensity.RequestedBasis != "consumption_lifecycle" {
		t.Errorf("the substitution should be recorded: %q", intensity.RequestedBasis)
	}
	if !strings.Contains(intensity.Source, "not published for this zone") {
		t.Errorf("the substitution should be visible: %q", intensity.Source)
	}
	// The place is the zone, not just the country it sits in.
	if !strings.Contains(intensity.Source, "Italy/SICI") {
		t.Errorf("provenance should name the zone: %q", intensity.Source)
	}
}

func TestAFallenBackBasisIsNotedInTheReport(t *testing.T) {
	server, _, _ := intensityServer(t, zoneReadingJSON(now()), http.StatusOK)
	cfg := testConfig(func(c *Config) {
		c.Intensity, c.Offline = 0, false
		c.IntensityAPI, c.Zone = server.URL, "IT/SICI"
	})
	report, err := SCIReport(Target{Kind: "test"}, Sample{WallS: 1, CPUS: 1}, cfg, 0, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	var noted bool
	for _, note := range report.Notes {
		if strings.Contains(note, "basis fell back to lifecycle") {
			noted = true
		}
	}
	if !noted {
		t.Errorf("the basis substitution should reach the report: %v", report.Notes)
	}
	if !strings.Contains(RenderText(report), "AGPL-3.0-or-later") {
		t.Error("the text disclosure should carry the attribution")
	}
}

func TestStalenessFollowsTheAPIsOwnRule(t *testing.T) {
	fresh := Reading{Basis: "measured", GeneratedAt: now()}
	if fresh.Stale(time.Now()) {
		t.Error("a reading generated now is not stale")
	}
	old := Reading{Basis: "measured",
		GeneratedAt: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)}
	if !old.Stale(time.Now()) {
		t.Error("a measured reading older than 65 minutes means a refresh was missed")
	}
	annual := Reading{Basis: "annual-average", GeneratedAt: now()}
	if !annual.Stale(time.Now()) {
		t.Error("an annual average never describes the hour you asked for")
	}
}

func TestAnnualAverageIsUsedButFlagged(t *testing.T) {
	server, _, _ := intensityServer(t, readingJSON("annual-average", now()), http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Country = server.URL, "IE"
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if intensity.Value != 339 || intensity.Measured || !intensity.Stale {
		t.Fatalf("%+v", intensity)
	}
}

func TestAnUnreachableAPIFallsBackToTheBundledTable(t *testing.T) {
	server, _, _ := intensityServer(t, "rate limited", http.StatusTooManyRequests)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Region = server.URL, "eu-west-1"
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if intensity.Value != GridZones["IE"] {
		t.Errorf("value: %g", intensity.Value)
	}
	if !strings.Contains(intensity.Source, "unreachable") {
		t.Errorf("the substitution should be visible: %q", intensity.Source)
	}
}

func TestReadingsAreCachedSoTheRateLimitIsNotHammered(t *testing.T) {
	server, hits, _ := intensityServer(t, readingJSON("measured", now()), http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Country = server.URL, "IE"
	for range 3 {
		if _, err := ResolveIntensity(cfg); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("%d requests for three resolutions; the hourly cache did not hold",
			hits.Load())
	}
	second, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(second.Source, "cached") {
		t.Errorf("a cache hit should say so: %q", second.Source)
	}
}

func TestOfflineNeverCallsOut(t *testing.T) {
	server, hits, _ := intensityServer(t, readingJSON("measured", now()), http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Region, cfg.Offline = server.URL, "eu-north-1", true
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 0 {
		t.Error("-offline made a request anyway")
	}
	if intensity.Value != GridZones["SE"] || strings.Contains(intensity.Source, "unreachable") {
		t.Errorf("%+v", intensity)
	}
}

func TestUnknownRegionIsRefusedRatherThanGuessed(t *testing.T) {
	cfg := NewConfig()
	cfg.Offline, cfg.Region = true, "atlantis-1"
	if _, err := ResolveIntensity(cfg); err == nil {
		t.Fatal("expected an error for an unknown region")
	}
}

func TestNoLocationFallsBackToTheWorldAverage(t *testing.T) {
	cfg := NewConfig()
	cfg.Offline = true
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if intensity.Value != DefaultIntensity || !strings.Contains(intensity.Source, "world") {
		t.Fatalf("%+v", intensity)
	}
}

func TestEveryCloudRegionResolvesOfflineToo(t *testing.T) {
	for region := range RegionCountry {
		cfg := NewConfig()
		cfg.Offline, cfg.Region = true, region
		if _, err := ResolveIntensity(cfg); err != nil {
			t.Errorf("%s: %v", region, err)
		}
	}
}

func TestCacheFreshnessFollowsTheReadingNotTheFile(t *testing.T) {
	clock := time.Now()
	stamp := func(ago time.Duration) string {
		return clock.Add(-ago).UTC().Format(time.RFC3339)
	}
	cases := []struct {
		name    string
		reading Reading
		age     time.Duration
		fresh   bool
	}{
		{"measured, generated ten minutes ago",
			Reading{Basis: "measured", GeneratedAt: stamp(10 * time.Minute)},
			10 * time.Minute, true},
		{"measured, already stale when cached",
			Reading{Basis: "measured", GeneratedAt: stamp(2 * time.Hour)},
			time.Minute, false},
		{"annual average, three hours old",
			Reading{Basis: "annual-average", GeneratedAt: stamp(3 * time.Hour)},
			3 * time.Hour, true},
		{"annual average, two days old",
			Reading{Basis: "annual-average"}, 48 * time.Hour, false},
	}
	for _, testCase := range cases {
		if got := cacheFresh(testCase.reading, testCase.age, clock); got != testCase.fresh {
			t.Errorf("%s: fresh=%t, want %t", testCase.name, got, testCase.fresh)
		}
	}
}

func TestAStaleCachedReadingIsRefetched(t *testing.T) {
	// Generated over an hour ago: the hourly refresh has happened since.
	body := readingJSON("measured", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339))
	server, hits, _ := intensityServer(t, body, http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Country = server.URL, "IE"
	for range 2 {
		if _, err := ResolveIntensity(cfg); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 2 {
		t.Fatalf("%d requests: a stale cache entry should be replaced, not reused",
			hits.Load())
	}
}

func TestALiveFailureFallsBackToTheCacheBeforeTheBundledTable(t *testing.T) {
	server, hits, _ := intensityServer(t, readingJSON("measured", now()), http.StatusOK)
	cfg := NewConfig()
	cfg.IntensityAPI, cfg.Country = server.URL, "IE"
	if _, err := ResolveIntensity(cfg); err != nil { // populate the cache
		t.Fatal(err)
	}
	server.Close() // the API goes away, but the reading is still the best we have
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if intensity.Value != 339 {
		t.Fatalf("expected the cached reading, got %g (%s)", intensity.Value, intensity.Source)
	}
	if hits.Load() != 1 {
		t.Errorf("requests: %d", hits.Load())
	}
}
