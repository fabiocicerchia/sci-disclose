// Package testutil holds helpers shared by the tests of several packages.
// It is under internal/ so it never reaches a consumer of this module.
package testutil

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/fabiocicerchia/sci-disclose/internal/config"
)

// WriteFile writes content to dir/name, creating parents, and returns the path.
// Any failure fails the test rather than being returned: a fixture that cannot
// be written is a broken test, not a case under test.
func WriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Approx fails the test when got and want differ by more than tolerance.
// Every number this tool produces is a float derived from a measurement, so an
// exact comparison would be testing the FPU rather than the arithmetic.
func Approx(t *testing.T, got, want, tolerance float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s: got %g, want %g (±%g)", what, got, want, tolerance)
	}
}

// Config is a deterministic Config for tests: one vCPU, a pinned intensity, no
// network, and the modelled energy backend — so a result depends only on the
// code under test and not on the host or the grid.
func Config(mutate func(*config.Config)) config.Config {
	cfg := config.NewConfig()
	cfg.VCPUs, cfg.TotalVCPUs = 1, 1
	cfg.Intensity = 500
	cfg.Offline = true
	cfg.EnergySource = "model"
	if mutate != nil {
		mutate(&cfg)
	}
	return cfg
}
