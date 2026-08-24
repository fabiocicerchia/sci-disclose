package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
)

func defaultPattern(t *testing.T) *regexp.Regexp {
	t.Helper()
	pattern, err := regexp.Compile(DefaultUnitPattern)
	if err != nil {
		t.Fatal(err)
	}
	return pattern
}

func TestScanUnitsAcceptsTheUsualMarkerSpellings(t *testing.T) {
	pattern := defaultPattern(t)
	for _, line := range []string{
		"SCI-UNITS: 5000", "sci_units=5000", "sci units 5000",
		"done. SCI-UNIT: 5000 in 3s", "SCI-Units:5000",
	} {
		count, ok := ScanUnits(line, pattern)
		if !ok || count != 5000 {
			t.Errorf("%q: got %g (%t)", line, count, ok)
		}
	}
}

func TestScanUnitsTakesTheWorkloadsFinalWord(t *testing.T) {
	pattern := defaultPattern(t)
	progress := "SCI-UNITS: 100\nstill going\nSCI-UNITS: 2500\n"
	if count, _ := ScanUnits(progress, pattern); count != 2500 {
		t.Errorf("the last count should win, got %g", count)
	}
}

func TestScanUnitsRejectsNonsense(t *testing.T) {
	pattern := defaultPattern(t)
	for _, text := range []string{"", "no marker here", "SCI-UNITS: 0", "SCI-UNITS: -5"} {
		if _, ok := ScanUnits(text, pattern); ok {
			t.Errorf("%q should not have produced a count", text)
		}
	}
}

func TestUnitsFromFileTakesABareNumberOrAMarker(t *testing.T) {
	dir := t.TempDir()
	pattern := defaultPattern(t)
	bare := writeFile(t, dir, "count.txt", "4200\n")
	if count, err := UnitsFromFile(bare, pattern); err != nil || count != 4200 {
		t.Errorf("bare number: %g (%v)", count, err)
	}
	marked := writeFile(t, dir, "report.log", "finished\nSCI-UNITS: 77\n")
	if count, err := UnitsFromFile(marked, pattern); err != nil || count != 77 {
		t.Errorf("marker: %g (%v)", count, err)
	}
	junk := writeFile(t, dir, "junk.txt", "no number at all\n")
	if _, err := UnitsFromFile(junk, pattern); err == nil {
		t.Error("a file with no count should be an error, not a silent 1")
	}
	if _, err := UnitsFromFile(filepath.Join(dir, "missing"), pattern); err == nil {
		t.Error("a missing file should be an error")
	}
}

func TestUnitsFromCommandReadsStdout(t *testing.T) {
	pattern := defaultPattern(t)
	if count, err := UnitsFromCommand("echo 640", pattern); err != nil || count != 640 {
		t.Errorf("plain command: %g (%v)", count, err)
	}
	// Anything shell-shaped goes through sh, so pipes and redirects work.
	if count, err := UnitsFromCommand("printf 'a\\nb\\n' | wc -l", pattern); err != nil ||
		count != 2 {
		t.Errorf("pipeline: %g (%v)", count, err)
	}
	if _, err := UnitsFromCommand("sh -c 'exit 1'", pattern); err == nil {
		t.Error("a failing command should be an error")
	}
	if _, err := UnitsFromCommand("echo nothing", pattern); err == nil {
		t.Error("output with no count should be an error")
	}
}

func TestUnitsAreReadFromTheWorkloadsOwnOutput(t *testing.T) {
	code, stdout := runCLI(t, "run", "-offline", "-units-from-stdout", "-unit-label",
		"image resized", "-format", "json", "--",
		"sh", "-c", "echo working; echo 'SCI-UNITS: 5000'")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var report Report
	if err := json.Unmarshal([]byte(stdout[strings.Index(stdout, "{"):]), &report); err != nil {
		t.Fatal(err)
	}
	if report.FunctionalUnit.Quantity != 5000 {
		t.Fatalf("R: %+v", report.FunctionalUnit)
	}
	if !strings.Contains(report.FunctionalUnit.Source, "stdout") {
		t.Errorf("the count's origin should be recorded: %q", report.FunctionalUnit.Source)
	}
	// The workload's own output still reaches the terminal.
	if !strings.Contains(stdout, "working") {
		t.Error("tapping stdout should not swallow it")
	}
}

func TestAMissingMarkerIsAnErrorNotASilentOne(t *testing.T) {
	code, out := runCLI(t, "run", "-offline", "-units-from-stdout", "--",
		"sh", "-c", "echo nothing")
	if code != 2 || !strings.Contains(out, "printed no unit count") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestExplicitUnitsBeatTheCounters(t *testing.T) {
	code, stdout := runCLI(t, "run", "-offline", "-units", "10", "-units-from-stdout",
		"-format", "json", "--", "sh", "-c", "echo 'SCI-UNITS: 5000'")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var report Report
	if err := json.Unmarshal([]byte(stdout[strings.Index(stdout, "{"):]), &report); err != nil {
		t.Fatal(err)
	}
	if report.FunctionalUnit.Quantity != 10 {
		t.Fatalf("--units should win: %+v", report.FunctionalUnit)
	}
	var noted bool
	for _, note := range report.Notes {
		if strings.Contains(note, "not read from the workload") {
			noted = true
		}
	}
	if !noted {
		t.Errorf("the ignored counter should be noted: %v", report.Notes)
	}
}

func TestUnitCountingIsOnlyOfferedWhereItMeansSomething(t *testing.T) {
	// `func` counts calls and `estimate` declares the quantity, so the flag has
	// no meaning there and should be rejected rather than silently ignored.
	if code, _ := runCLI(t, "estimate", "-units-from-stdout", "-f", "sci.yaml"); code != 2 {
		t.Error("estimate should reject -units-from-stdout")
	}
	if code, _ := runCLI(t, "func", "-units-from-stdout", "mod:fn"); code != 2 {
		t.Error("func should reject -units-from-stdout")
	}
}

func TestScrapeCounterSumsEverySeriesOfTheMetric(t *testing.T) {
	body := `# HELP http_requests_total total requests
# TYPE http_requests_total counter
http_requests_total{method="get",code="200"} 900
http_requests_total{method="post",code="200"} 100
other_metric 5
`
	server, _, _ := intensityServer(t, body, 200)
	count, err := ScrapeCounter(server.URL, "http_requests_total")
	if err != nil || count != 1000 {
		t.Fatalf("got %g (%v)", count, err)
	}
	if _, err := ScrapeCounter(server.URL, "absent_total"); err == nil {
		t.Error("a metric that is not published should be an error")
	}
}

func TestScrapeCounterRejectsABadEndpoint(t *testing.T) {
	server, _, _ := intensityServer(t, "nope", 500)
	if _, err := ScrapeCounter(server.URL, "http_requests_total"); err == nil {
		t.Error("HTTP 500 should be an error")
	}
}

func TestCounterDeltaAcrossTheRunBecomesR(t *testing.T) {
	var served atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The counter advances between the two scrapes, as a live service's would.
		fmt.Fprintf(w, "http_requests_total{code=\"200\"} %d\n", 1000+served.Add(250))
	}))
	t.Cleanup(server.Close)

	code, stdout := runCLI(t, "run", "-offline", "-units-metric", "http_requests_total",
		"-units-url", server.URL, "-unit-label", "request", "-format", "json",
		"--", "sh", "-c", "true")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var report Report
	if err := json.Unmarshal([]byte(stdout[strings.Index(stdout, "{"):]), &report); err != nil {
		t.Fatal(err)
	}
	if report.FunctionalUnit.Quantity != 250 {
		t.Fatalf("R should be the delta, not the total: %+v", report.FunctionalUnit)
	}
	if !strings.Contains(report.FunctionalUnit.Source, "advanced by") {
		t.Errorf("origin: %q", report.FunctionalUnit.Source)
	}
}

func TestACounterThatDoesNotMoveIsRefused(t *testing.T) {
	server, _, _ := intensityServer(t, "http_requests_total 1000\n", 200)
	code, out := runCLI(t, "run", "-offline", "-units-metric", "http_requests_total",
		"-units-url", server.URL, "--", "sh", "-c", "true")
	if code != 2 || !strings.Contains(out, "did not advance") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestUnitsMetricNeedsAURL(t *testing.T) {
	code, out := runCLI(t, "run", "-offline", "-units-metric", "x", "--", "sh", "-c", "true")
	if code != 2 || !strings.Contains(out, "needs -units-url") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestUnitsReconcilesAMeasurementWithALaterCount(t *testing.T) {
	dir := t.TempDir()
	measured := filepath.Join(dir, "measured.json")
	code, out := runCLI(t, "run", "-offline", "-vcpus", "1", "-format", "json",
		"-o", measured, "--", "sh", "-c", "true")
	if code != 0 {
		t.Fatalf("measure: exit %d: %s", code, out)
	}
	before := readReport(t, measured)

	reconciled := filepath.Join(dir, "final.json")
	code, out = runCLI(t, "units", "-units", "4300000", "-unit-label", "checkout request",
		"-format", "json", "-o", reconciled, measured)
	if code != 0 {
		t.Fatalf("units: exit %d: %s", code, out)
	}
	after := readReport(t, reconciled)

	// The carbon is whatever was measured; only the denominator arrived late.
	approx(t, after.Total, before.Total, 1e-12, "C is unchanged")
	approx(t, after.EnergyKWh, before.EnergyKWh, 1e-12, "E is unchanged")
	approx(t, after.SCI, before.Total/4300000, 1e-15, "SCI is C over the later count")
	if after.SCIUnit != "gCO2e per checkout request" {
		t.Errorf("unit: %q", after.SCIUnit)
	}
	if !strings.Contains(after.FunctionalUnit.Source, "after the measurement") {
		t.Errorf("the late arrival should be recorded: %q", after.FunctionalUnit.Source)
	}
	var noted bool
	for _, note := range after.Notes {
		if strings.Contains(note, "reconciled after the fact") {
			noted = true
		}
	}
	if !noted {
		t.Errorf("notes: %v", after.Notes)
	}
}

func TestUnitsGatesOnBudgetToo(t *testing.T) {
	dir := t.TempDir()
	report := writeFile(t, dir, "measured.json",
		`{"total_gco2e": 100, "sci": 100, "sci_unit": "gCO2e per run",
		  "functional_unit": {"label": "run", "quantity": 1}}`)
	if code, _ := runCLI(t, "units", "-units", "1000", "-budget", "1", report); code != 0 {
		t.Error("0.1 per unit is within a budget of 1")
	}
	if code, _ := runCLI(t, "units", "-units", "10", "-budget", "1", report); code != 1 {
		t.Error("10 per unit is over a budget of 1")
	}
}

func TestUnitsRefusesWhatItCannotDivide(t *testing.T) {
	dir := t.TempDir()
	empty := writeFile(t, dir, "empty.json", `{"sci": 0, "total_gco2e": 0}`)
	for _, args := range [][]string{
		{"units", "-units", "10"},                 // no report
		{"units", "-units", "0", empty},           // no count
		{"units", "-units", "10", empty},          // nothing to divide
		{"units", "-units", "10", "missing.json"}, // no such file
	} {
		if code, out := runCLI(t, args...); code != 2 {
			t.Errorf("%v: exit %d\n%s", args, code, out)
		}
	}
}
