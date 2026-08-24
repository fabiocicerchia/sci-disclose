package grid

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
	"github.com/fabiocicerchia/sci-disclose/internal/config"
	"github.com/fabiocicerchia/sci-disclose/internal/testutil"
)

func TestExplicitIntensityBeatsEveryLookup(t *testing.T) {
	server, hits, _ := testutil.IntensityServer(t, testutil.ReadingJSON("measured", testutil.Now()), http.StatusOK)
	cfg := config.NewConfig()
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
	server, _, paths := testutil.IntensityServer(t, testutil.ReadingJSON("measured", testutil.Now()), http.StatusOK)
	cfg := config.NewConfig()
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
	// "AGPL-3.0-or-later" is an SPDX licence id, not a key -- it just has the
	// entropy of one, and gitleaks scores it as generic-api-key.
	for _, want := range []string{"Carbon Intensity API", "AGPL-3.0-or-later", // gitleaks:allow
		"ENTSO-E", "smartgriddashboard"} {
		if !strings.Contains(credit, want) {
			t.Errorf("credit %q is missing %q", credit, want)
		}
	}
}

func TestIntensityBasisIsSelectable(t *testing.T) {
	server, _, _ := testutil.IntensityServer(t, testutil.ReadingJSON("measured", testutil.Now()), http.StatusOK)
	cfg := config.NewConfig()
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
	server, _, paths := testutil.IntensityServer(t, testutil.ZoneReadingJSON(testutil.Now()), http.StatusOK)
	cfg := config.NewConfig()
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

func TestStalenessFollowsTheAPIsOwnRule(t *testing.T) {
	fresh := Reading{Basis: "measured", GeneratedAt: testutil.Now()}
	if fresh.Stale(time.Now()) {
		t.Error("a reading generated now is not stale")
	}
	old := Reading{Basis: "measured",
		GeneratedAt: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)}
	if !old.Stale(time.Now()) {
		t.Error("a measured reading older than 65 minutes means a refresh was missed")
	}
	annual := Reading{Basis: "annual-average", GeneratedAt: testutil.Now()}
	if !annual.Stale(time.Now()) {
		t.Error("an annual average never describes the hour you asked for")
	}
}

func TestAnnualAverageIsUsedButFlagged(t *testing.T) {
	server, _, _ := testutil.IntensityServer(t, testutil.ReadingJSON("annual-average", testutil.Now()), http.StatusOK)
	cfg := config.NewConfig()
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
	server, _, _ := testutil.IntensityServer(t, "rate limited", http.StatusTooManyRequests)
	cfg := config.NewConfig()
	cfg.IntensityAPI, cfg.Region = server.URL, "eu-west-1"
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if intensity.Value != coefficients.GridZones["IE"] {
		t.Errorf("value: %g", intensity.Value)
	}
	if !strings.Contains(intensity.Source, "unreachable") {
		t.Errorf("the substitution should be visible: %q", intensity.Source)
	}
}

func TestReadingsAreCachedSoTheRateLimitIsNotHammered(t *testing.T) {
	server, hits, _ := testutil.IntensityServer(t, testutil.ReadingJSON("measured", testutil.Now()), http.StatusOK)
	cfg := config.NewConfig()
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
	server, hits, _ := testutil.IntensityServer(t, testutil.ReadingJSON("measured", testutil.Now()), http.StatusOK)
	cfg := config.NewConfig()
	cfg.IntensityAPI, cfg.Region, cfg.Offline = server.URL, "eu-north-1", true
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 0 {
		t.Error("-offline made a request anyway")
	}
	if intensity.Value != coefficients.GridZones["SE"] || strings.Contains(intensity.Source, "unreachable") {
		t.Errorf("%+v", intensity)
	}
}

func TestUnknownRegionIsRefusedRatherThanGuessed(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Offline, cfg.Region = true, "atlantis-1"
	if _, err := ResolveIntensity(cfg); err == nil {
		t.Fatal("expected an error for an unknown region")
	}
}

func TestNoLocationFallsBackToTheWorldAverage(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Offline = true
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if intensity.Value != coefficients.DefaultIntensity || !strings.Contains(intensity.Source, "world") {
		t.Fatalf("%+v", intensity)
	}
}

func TestEveryCloudRegionResolvesOfflineToo(t *testing.T) {
	for region := range coefficients.RegionCountry {
		cfg := config.NewConfig()
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
	body := testutil.ReadingJSON("measured", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339))
	server, hits, _ := testutil.IntensityServer(t, body, http.StatusOK)
	cfg := config.NewConfig()
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
	server, hits, _ := testutil.IntensityServer(t, testutil.ReadingJSON("measured", testutil.Now()), http.StatusOK)
	cfg := config.NewConfig()
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
