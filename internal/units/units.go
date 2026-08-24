package units

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/fabiocicerchia/sci-disclose/internal/fetch"
)

// Counting R.
//
// SCI is a rate, so something has to count the work. Typing --units by hand is
// fine for a batch of known size and useless for anything else, so the count
// can instead come from the workload's own output, from a file it wrote, or
// from a command run afterwards. Whichever is used is named in the report:
// a functional unit nobody can trace is not a disclosure.

// DefaultUnitPattern matches the marker a workload prints to report its own
// unit count: `SCI-UNITS: 5000`, `sci_units=5000`, `sci units 5000`.
const DefaultUnitPattern = `(?i)sci[-_ ]?units?[=:\s]+([0-9]+(?:\.[0-9]+)?)`

// UnitSource says where a count came from, for the report.
type UnitSource struct {
	Count  float64
	Origin string
}

// ScanUnits returns the last number the pattern captures. Last, not first:
// a workload that reports progress should be taken at its final word.
func ScanUnits(text string, pattern *regexp.Regexp) (float64, bool) {
	matches := pattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0, false
	}
	last := matches[len(matches)-1]
	if len(last) < 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(last[1], 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

// UnitsFromFile reads a count from a file the workload wrote: either a bare
// number, or any line carrying the marker.
func UnitsFromFile(path string, pattern *regexp.Regexp) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("cannot read the unit count: %w", err)
	}
	text := strings.TrimSpace(string(data))
	if value, err := strconv.ParseFloat(text, 64); err == nil && value > 0 {
		return value, nil
	}
	if value, ok := ScanUnits(text, pattern); ok {
		return value, nil
	}
	return 0, fmt.Errorf("%s holds no unit count: expected a bare number or a line "+
		"matching %s", path, pattern)
}

// UnitsFromCommand runs a command after the workload has finished and reads the
// count from its output — `wc -l < out.csv`, a jq over a load-test summary, a
// curl of a metrics endpoint. It runs outside the measured window, so it costs
// the measurement nothing.
func UnitsFromCommand(commandLine string, pattern *regexp.Regexp) (float64, error) {
	fields := strings.Fields(commandLine)
	if len(fields) == 0 {
		return 0, fmt.Errorf("--units-cmd is empty")
	}
	var command *exec.Cmd
	if strings.ContainsAny(commandLine, "|<>$*") {
		// Let a shell handle anything that clearly wants one.
		command = exec.Command("sh", "-c", commandLine)
	} else {
		command = exec.Command(fields[0], fields[1:]...)
	}
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		return 0, fmt.Errorf("--units-cmd failed: %w", err)
	}
	text := strings.TrimSpace(string(output))
	if value, err := strconv.ParseFloat(text, 64); err == nil && value > 0 {
		return value, nil
	}
	if value, ok := ScanUnits(text, pattern); ok {
		return value, nil
	}
	return 0, fmt.Errorf("--units-cmd printed no unit count: %q", text)
}

// ScrapeCounter reads a Prometheus-style counter from a metrics endpoint. A
// service's units accrue over time, so the honest count for a measured window
// is the delta across it: scrape before, scrape after, subtract.
func ScrapeCounter(url, metric string) (float64, error) {
	body, status, err := fetch.Get(url)
	if err != nil {
		return 0, fmt.Errorf("cannot scrape %s: %w", url, err)
	}
	if status != 200 {
		return 0, fmt.Errorf("cannot scrape %s: HTTP %d", url, status)
	}
	total, found := 0.0, false
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		if base, _, _ := strings.Cut(name, "{"); base != metric {
			continue
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			continue
		}
		total += parsed // every series of the metric, summed
		found = true
	}
	if !found {
		return 0, fmt.Errorf("%s publishes no counter called %q", url, metric)
	}
	return total, nil
}
