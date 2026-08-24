package units

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/fabiocicerchia/sci-disclose/internal/testutil"
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
	bare := testutil.WriteFile(t, dir, "count.txt", "4200\n")
	if count, err := UnitsFromFile(bare, pattern); err != nil || count != 4200 {
		t.Errorf("bare number: %g (%v)", count, err)
	}
	marked := testutil.WriteFile(t, dir, "report.log", "finished\nSCI-UNITS: 77\n")
	if count, err := UnitsFromFile(marked, pattern); err != nil || count != 77 {
		t.Errorf("marker: %g (%v)", count, err)
	}
	junk := testutil.WriteFile(t, dir, "junk.txt", "no number at all\n")
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

func TestScrapeCounterSumsEverySeriesOfTheMetric(t *testing.T) {
	body := `# HELP http_requests_total total requests
# TYPE http_requests_total counter
http_requests_total{method="get",code="200"} 900
http_requests_total{method="post",code="200"} 100
other_metric 5
`
	server, _, _ := testutil.IntensityServer(t, body, 200)
	count, err := ScrapeCounter(server.URL, "http_requests_total")
	if err != nil || count != 1000 {
		t.Fatalf("got %g (%v)", count, err)
	}
	if _, err := ScrapeCounter(server.URL, "absent_total"); err == nil {
		t.Error("a metric that is not published should be an error")
	}
}

func TestScrapeCounterRejectsABadEndpoint(t *testing.T) {
	server, _, _ := testutil.IntensityServer(t, "nope", 500)
	if _, err := ScrapeCounter(server.URL, "http_requests_total"); err == nil {
		t.Error("HTTP 500 should be an error")
	}
}
