package main

// The unit-counting flags end to end. The counting itself is tested in
// internal/units; these drive it through the CLI, which is the only place
// the flags, the workload and the disclosure meet.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/fabiocicerchia/sci-disclose/internal/sci"
	"github.com/fabiocicerchia/sci-disclose/internal/testutil"
)

func TestUnitsAreReadFromTheWorkloadsOwnOutput(t *testing.T) {
	code, stdout := runCLI(t, "run", "-offline", "-units-from-stdout", "-unit-label",
		"image resized", "-format", "json", "--",
		"sh", "-c", "echo working; echo 'SCI-UNITS: 5000'")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var report sci.Report
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
	var report sci.Report
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
	var report sci.Report
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
	server, _, _ := testutil.IntensityServer(t, "http_requests_total 1000\n", 200)
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
	testutil.Approx(t, after.Total, before.Total, 1e-12, "C is unchanged")
	testutil.Approx(t, after.EnergyKWh, before.EnergyKWh, 1e-12, "E is unchanged")
	testutil.Approx(t, after.SCI, before.Total/4300000, 1e-15, "SCI is C over the later count")
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
	report := testutil.WriteFile(t, dir, "measured.json",
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
	empty := testutil.WriteFile(t, dir, "empty.json", `{"sci": 0, "total_gco2e": 0}`)
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
