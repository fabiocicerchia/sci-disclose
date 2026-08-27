package grid

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
	"github.com/fabiocicerchia/sci-disclose/internal/config"
	"github.com/fabiocicerchia/sci-disclose/internal/fetch"
)

// I — carbon intensity.
//
// The default source is the Carbon Intensity API, which publishes the last
// hour's grid intensity per country and per bidding zone. It is rate limited
// to one request per ten seconds per IP and its readings only move hourly, so
// responses are cached on disk for an hour and a failure degrades to the
// bundled yearly averages rather than failing the run.

// DataSource is the grid operator a reading was computed from.
type DataSource struct {
	Name     string `json:"name"`
	URL      string `json:"url,omitempty"`
	Realtime bool   `json:"realtime"`
	Status   string `json:"status,omitempty"`
}

// Attribution is what the API asks to be carried with its figures. It is
// AGPL-3.0 and computes intensities the operators do not themselves publish,
// so a disclosure that quotes the number carries this with it.
type Attribution struct {
	Name       string `json:"name"`
	Author     string `json:"author,omitempty"`
	URL        string `json:"url,omitempty"`
	Repository string `json:"repository,omitempty"`
	License    string `json:"license,omitempty"`
}

// Reading is one response from /v1/last-hour/<CODE> or /v1/zones/<CODE>/<ZONE>.
// The four intensities are pointers because zone readings publish only the
// production-based pair.
type Reading struct {
	Country              string      `json:"country"`
	CountryCode          string      `json:"country_code"`
	Zone                 string      `json:"zone"`
	HourStart            *string     `json:"hour_start"`
	HourEnd              *string     `json:"hour_end"`
	Unit                 string      `json:"unit"`
	Direct               *float64    `json:"direct"`
	Lifecycle            *float64    `json:"lifecycle"`
	ConsumptionDirect    *float64    `json:"consumption_direct"`
	ConsumptionLifecycle *float64    `json:"consumption_lifecycle"`
	Basis                string      `json:"basis"`
	Estimated            bool        `json:"estimated"`
	DataYear             int         `json:"data_year"`
	DataSource           DataSource  `json:"data_source"`
	Methodology          string      `json:"methodology"`
	Attribution          Attribution `json:"attribution"`
	GeneratedAt          string      `json:"generated_at"`
}

// Where names the reading's location the way the API reports it: the country,
// narrowed to the bidding zone when the reading is for one.
func (r Reading) Where(fallback string) string {
	where := r.Country
	if where == "" {
		where = fallback
	}
	if r.Zone != "" && r.Zone != r.CountryCode {
		where += "/" + r.Zone
	}
	return where
}

// Window is the interval the reading covers. It is not always an hour: zone
// readings can carry the operator's finer settlement period.
func (r Reading) Window() string {
	if r.HourStart == nil || r.HourEnd == nil {
		return "" // an annual average describes no particular hour
	}
	return *r.HourStart + " to " + *r.HourEnd
}

// Value returns the named intensity, or the first one present in fallback
// order when the reading omits it, along with which field was used.
func (r Reading) Value(preferred string) (float64, string, bool) {
	fields := map[string]*float64{
		"direct":                r.Direct,
		"lifecycle":             r.Lifecycle,
		"consumption_direct":    r.ConsumptionDirect,
		"consumption_lifecycle": r.ConsumptionLifecycle,
	}
	if value, ok := fields[preferred]; ok && value != nil {
		return *value, preferred, true
	}
	for _, name := range coefficients.IntensityBases {
		if value := fields[name]; value != nil {
			return *value, name, true
		}
	}
	return 0, "", false
}

// Stale applies the API's own freshness rule: an annual average never
// describes the hour you asked for, and a measured reading more than 65
// minutes old means an hourly refresh was missed.
func (r Reading) Stale(now time.Time) bool {
	if r.Basis != "measured" {
		return true
	}
	generated, err := time.Parse(time.RFC3339, r.GeneratedAt)
	if err != nil {
		return true
	}
	return now.Sub(generated) > 65*time.Minute
}

// Intensity is I, with enough provenance to argue with.
type Intensity struct {
	Value          float64      `json:"gco2e_per_kwh"`
	Source         string       `json:"source"`
	Basis          string       `json:"basis,omitempty"`
	RequestedBasis string       `json:"requested_basis,omitempty"`
	Window         string       `json:"window,omitempty"`
	Measured       bool         `json:"measured"`
	Estimated      bool         `json:"estimated,omitempty"`
	Stale          bool         `json:"stale,omitempty"`
	DataYear       int          `json:"data_year,omitempty"`
	DataSource     *DataSource  `json:"data_source,omitempty"`
	Methodology    string       `json:"methodology,omitempty"`
	Attribution    *Attribution `json:"attribution,omitempty"`
}

// Credit is the one-line attribution shown under the intensity in a report.
func (i Intensity) Credit() string {
	if i.Attribution == nil {
		return ""
	}
	credit := i.Attribution.Name
	if i.Attribution.License != "" {
		credit += " (" + i.Attribution.License + ")"
	}
	if i.DataSource != nil && i.DataSource.Name != "" {
		credit += " · grid data from " + i.DataSource.Name
		if i.DataSource.URL != "" {
			credit += " " + i.DataSource.URL
		}
	}
	return credit
}

// ResolveIntensity picks I in this order: an explicit figure, the live API for
// a known country or zone, the bundled yearly table, then the world average.
func ResolveIntensity(cfg config.Config) (Intensity, error) {
	if cfg.Intensity > 0 {
		return Intensity{Value: cfg.Intensity, Source: "--intensity (user supplied)"}, nil
	}

	path, label := apiTarget(cfg)
	if path != "" && !cfg.Offline {
		if reading, from, err := fetchReading(cfg, path); err == nil {
			if value, basis, ok := reading.Value(cfg.IntensityBasis); ok {
				intensity := Intensity{
					Value:       value,
					Source:      describeReading(reading, basis, cfg.IntensityBasis, label, from),
					Basis:       basis,
					Window:      reading.Window(),
					Measured:    reading.Basis == "measured",
					Estimated:   reading.Estimated,
					Stale:       reading.Stale(time.Now()),
					DataYear:    reading.DataYear,
					Methodology: reading.Methodology,
				}
				if basis != cfg.IntensityBasis {
					intensity.RequestedBasis = cfg.IntensityBasis
				}
				if reading.DataSource.Name != "" {
					source := reading.DataSource
					intensity.DataSource = &source
				}
				if reading.Attribution.Name != "" {
					attribution := reading.Attribution
					intensity.Attribution = &attribution
				}
				return intensity, nil
			}
		}
	}

	if bundled, zone, ok := bundledIntensity(cfg); ok {
		reason := "bundled yearly average, zone " + zone
		if path != "" && !cfg.Offline {
			reason += " (the intensity API was unreachable)"
		}
		return Intensity{Value: bundled, Source: reason}, nil
	}
	if cfg.Region != "" || cfg.Country != "" || cfg.Zone != "" {
		return Intensity{}, fmt.Errorf("unknown region or zone %q; pass --intensity, "+
			"or see `sci coefficients`", cmp.Or(cfg.Zone, cfg.Country, cfg.Region))
	}
	return Intensity{
		Value:  coefficients.DefaultIntensity,
		Source: "world average (no --region, --country or --zone given)",
	}, nil
}

// apiTarget builds the API path for the configured location, if any.
func apiTarget(cfg config.Config) (path, label string) {
	if cfg.Zone != "" {
		parts := strings.SplitN(strings.ToUpper(cfg.Zone), "/", 2)
		if len(parts) == 2 {
			return "/v1/zones/" + parts[0] + "/" + parts[1], parts[0] + "/" + parts[1]
		}
	}
	country := strings.ToUpper(cfg.Country)
	if country == "" {
		country = coefficients.RegionCountry[cfg.Region]
	}
	if country == "" {
		return "", ""
	}
	return "/v1/last-hour/" + country, country
}

func describeReading(reading Reading, basis, requested, label, from string) string {
	when := reading.Window()
	if when == "" {
		// An annual average carries no hour_start: it describes no hour at all.
		when = "no particular hour"
	}
	source := reading.DataSource.Name
	if source == "" {
		source = "Carbon Intensity API"
	}
	basisNote := reading.Basis
	if basisNote == "" {
		basisNote = "unknown basis"
	}
	described := fmt.Sprintf("Carbon Intensity API: %s %s [%s], %s, via %s (%s)",
		reading.Where(label), basis, basisNote, when, source, from)
	if basis != requested {
		described += fmt.Sprintf(" — %s is not published for this zone", requested)
	}
	return described
}

// bundledIntensity is the offline fallback: the yearly zone averages shipped
// with the tool.
func bundledIntensity(cfg config.Config) (float64, string, bool) {
	candidates := []string{}
	if cfg.Region != "" {
		if zone, ok := coefficients.RegionZone[cfg.Region]; ok {
			candidates = append(candidates, zone)
		}
		candidates = append(candidates, strings.ToUpper(cfg.Region))
	}
	if cfg.Country != "" {
		candidates = append(candidates, strings.ToUpper(cfg.Country))
	}
	if cfg.Zone != "" {
		upper := strings.ToUpper(cfg.Zone)
		candidates = append(candidates, upper)
		if country, _, found := strings.Cut(upper, "/"); found {
			candidates = append(candidates, country)
		}
	}
	for _, candidate := range candidates {
		if value, ok := coefficients.GridZones[candidate]; ok {
			return value, candidate, true
		}
	}
	return 0, "", false
}

// fetchReading returns a reading from the on-disk cache when it is fresh, and
// from the API otherwise. A failed request falls back to an expired cache entry
// before giving up, so a rate limit or an outage never fails a measurement.
func fetchReading(cfg config.Config, path string) (Reading, string, error) {
	endpoint := strings.TrimSuffix(cfg.IntensityAPI, "/") + path
	cachePath := cacheFile(endpoint)

	if body, age, err := readCache(cachePath); err == nil {
		var reading Reading
		if json.Unmarshal(body, &reading) == nil && cacheFresh(reading, age, time.Now()) {
			return reading, fmt.Sprintf("cached %s ago", age.Round(time.Minute)), nil
		}
	}

	body, err := getJSON(endpoint)
	if err != nil {
		if cached, age, cacheErr := readCache(cachePath); cacheErr == nil {
			var reading Reading
			if json.Unmarshal(cached, &reading) == nil {
				return reading, fmt.Sprintf("stale cache, %s old, live fetch failed: %v",
					age.Round(time.Minute), err), nil
			}
		}
		return Reading{}, "", err
	}
	var reading Reading
	if err := json.Unmarshal(body, &reading); err != nil {
		return Reading{}, "", fmt.Errorf("unreadable response from %s: %w", endpoint, err)
	}
	writeCache(cachePath, body)
	return reading, "fetched", nil
}

// cacheFresh decides whether a cached reading still describes now. Measured
// readings refresh hourly, so one is worth keeping until it goes stale by the
// API's own rule; annual averages are rewritten weekly and cannot go out of
// date in an hour.
func cacheFresh(reading Reading, age time.Duration, now time.Time) bool {
	if reading.Basis != "measured" {
		return age < 24*time.Hour
	}
	if generated, err := time.Parse(time.RFC3339, reading.GeneratedAt); err == nil {
		return now.Sub(generated) < 65*time.Minute
	}
	return age < time.Hour
}

func getJSON(endpoint string) ([]byte, error) {
	body, status, err := fetch.Get(endpoint)
	if err != nil {
		return nil, err
	}
	switch {
	case status == http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited (1 request per 10s per IP)")
	case status == http.StatusNotFound:
		return nil, fmt.Errorf("no reading published for that country or zone")
	case status != http.StatusOK:
		return nil, fmt.Errorf("HTTP %d", status)
	}
	return body, nil
}

func cacheFile(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "sci-disclose", hex.EncodeToString(sum[:8])+".json")
}

func readCache(path string) ([]byte, time.Duration, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	return body, time.Since(info.ModTime()), nil
}

// writeCache stores one API response for the cache window.
//
// 0700/0600, not 0755/0644: cacheFile falls back to os.TempDir() when
// os.UserCacheDir() fails, and /tmp is shared and world-writable. A
// group-readable directory there lets any other local user enumerate which
// grids this host has queried, and — the part that actually bites — a
// directory they can write to is one they can plant a symlink in, which
// os.WriteFile would then follow. Nothing but this process reads the cache, so
// there is nothing to trade away. gosec G301/G306.
func writeCache(path string, body []byte) {
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	_ = os.WriteFile(path, body, 0o600)
}
