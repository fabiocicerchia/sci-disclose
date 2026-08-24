package main

import (
	"math"
	"testing"
)

func approx(t *testing.T, got, want, tolerance float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s: got %g, want %g (±%g)", what, got, want, tolerance)
	}
}

func testConfig(mutate func(*Config)) Config {
	cfg := NewConfig()
	cfg.VCPUs, cfg.TotalVCPUs = 1, 1
	cfg.Intensity = 500
	cfg.Offline = true
	cfg.EnergySource = "model"
	if mutate != nil {
		mutate(&cfg)
	}
	return cfg
}

func TestEmbodiedIsTotalTimesTimeShareTimesResourceShare(t *testing.T) {
	cfg := testConfig(func(c *Config) {
		c.VCPUs, c.TotalVCPUs = 2, 8
		c.EmbodiedKg, c.LifespanYears = 1000, 4
	})
	m := EmbodiedGCO2e(cfg, HoursPerYear) // one year of one machine
	approx(t, m.TimeShare, 0.25, 1e-9, "time share")
	approx(t, m.ResourceShare, 0.25, 1e-9, "resource share")
	approx(t, m.GCO2e, 1_000_000*0.25*0.25, 1e-6, "M")
}

func TestSCIIsTotalCarbonDividedByTheFunctionalUnit(t *testing.T) {
	cfg := testConfig(func(c *Config) {
		c.Provider, c.Units, c.UnitLabel = "aws", 100, "request"
	})
	sample := Sample{WallS: 3600, CPUS: 3600, PeakRSSGB: 1}
	report, err := SCIReport(Target{Kind: "test"}, sample, cfg, 0, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	approx(t, report.Total, report.Operational+report.Embodied, 1e-9, "C = O + M")
	approx(t, report.SCI, report.Total/100, 1e-12, "SCI = C / R")
	approx(t, report.Operational, report.EnergyKWh*500, 1e-9, "O = E x I")
	if report.SCIUnit != "gCO2e per request" {
		t.Fatalf("unit label: %q", report.SCIUnit)
	}
}

func TestConfigValidationRejectsUnknownInputs(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"provider": func(c *Config) { c.Provider = "hetzner" },
		"hardware": func(c *Config) { c.HardwareName = "toaster" },
		"medium":   func(c *Config) { c.StorageMedium = "tape" },
		"energy":   func(c *Config) { c.EnergySource = "vibes" },
	} {
		if err := testConfig(mutate).Validate(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if err := testConfig(nil).Validate(); err != nil {
		t.Errorf("defaults should validate: %v", err)
	}
}

func TestOverridesBeatThePresets(t *testing.T) {
	cfg := testConfig(func(c *Config) {
		c.Provider, c.PUE = "aws", 2.5
		c.HardwareName, c.EmbodiedKg, c.LifespanYears = "laptop", 400, 6
	})
	if got := cfg.Profile().PUE; got != 2.5 {
		t.Errorf("PUE override: %g", got)
	}
	if got := cfg.Profile().MinW; got != CPUProfiles["aws"].MinW {
		t.Errorf("profile should stay AWS: %g", got)
	}
	device := cfg.DeviceSpec()
	if device.EmbodiedKg != 400 || device.LifespanYears != 6 {
		t.Errorf("embodied overrides: %+v", device)
	}
}

func TestBudgetVerdictIsAttachedAndReturned(t *testing.T) {
	cfg := testConfig(nil)
	sample := Sample{WallS: 60, CPUS: 30, PeakRSSGB: 0.5}
	report, err := SCIReport(Target{Kind: "test"}, sample, cfg, 0, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ApplyBudget(report, report.SCI*2, true) || !report.Budget.Pass {
		t.Error("a generous budget should pass")
	}
	if ApplyBudget(report, report.SCI/2, true) || report.Budget.Pass {
		t.Error("a tight budget should fail")
	}
	report.Budget = nil
	if !ApplyBudget(report, 0, false) || report.Budget != nil {
		t.Error("no budget means no verdict")
	}
}
