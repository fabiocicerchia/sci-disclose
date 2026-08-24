package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const demoManifest = `
name: demo
functional-unit:
  label: request
  quantity: 1000
defaults:
  provider: aws
  intensity: 100
  period-hours: 10
components:
  - name: api
    type: compute
    vcpus: 2
    replicas: 2
    utilisation: 1.0
    memory-gb: 4
    embodied:
      device-kg: 1000
      lifespan-years: 4
      total-vcpus: 64
  - name: bucket
    type: storage
    storage-gb: 1024
  - name: egress
    type: network
    network-gb: 10
`

func loadDemo(t *testing.T) Manifest {
	t.Helper()
	path := writeFile(t, t.TempDir(), "sci.yaml", demoManifest)
	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func offlineConfig() Config {
	cfg := NewConfig()
	cfg.Offline = true
	return cfg
}

func TestManifestTotalsAreTheSumOfTheComponents(t *testing.T) {
	report, err := EstimateManifest(loadDemo(t), offlineConfig(), "sci.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Components) != 3 {
		t.Fatalf("components: %d", len(report.Components))
	}
	var energy float64
	for _, row := range report.Components {
		energy += row.EnergyKWh
	}
	approx(t, report.EnergyKWh, energy, 1e-12, "total energy")
	approx(t, report.SCI, report.Total/1000, 1e-12, "SCI per request")
	if report.SCIUnit != "gCO2e per request" {
		t.Errorf("unit: %s", report.SCIUnit)
	}
}

func TestManifestEmbodiedUsesVCPUsOverTotalVCPUs(t *testing.T) {
	report, err := EstimateManifest(loadDemo(t), offlineConfig(), "")
	if err != nil {
		t.Fatal(err)
	}
	expected := 1_000_000 * (10 / (4 * HoursPerYear)) * (2.0 / 64) * 2 // x2 replicas
	approx(t, report.Components[0].Embodied, expected, 1e-9, "M for the api component")
}

func TestManifestComputeMatchesTheModelUsedForMeasuredRuns(t *testing.T) {
	report, err := EstimateManifest(loadDemo(t), offlineConfig(), "")
	if err != nil {
		t.Fatal(err)
	}
	profile := CPUProfiles["aws"]
	// Four vCPUs (2 x 2 replicas) at 100% for ten hours, plus 8 GB of memory.
	cpu := 4 * profile.MaxW * 10 / 1000
	memory := MemorykWh(8, 10)
	approx(t, report.Components[0].EnergyKWh, (cpu+memory)*profile.PUE, 1e-9, "api energy")
}

func TestEndUserDevicesCarryNoDatacentreOverhead(t *testing.T) {
	manifest := Manifest{
		FunctionalUnit: UnitSpec{Label: "session", Quantity: 1},
		Defaults:       Defaults{Intensity: 100, PeriodHours: 2},
		Components: []Component{
			{Name: "phones", Type: "device", Watts: 5, Replicas: 10},
		},
	}
	report, err := EstimateManifest(manifest, offlineConfig(), "")
	if err != nil {
		t.Fatal(err)
	}
	row := report.Components[0]
	approx(t, row.EnergyKWh, 5*2*10/1000.0, 1e-12, "device energy")
	for _, part := range row.EnergyBreakdown {
		if part.Name == "datacentre_overhead" {
			t.Error("end-user devices are on the grid, not in a datacentre")
		}
	}
}

func TestManifestAcceptsUnderscoresAndBothSpellingsOfUtilisation(t *testing.T) {
	path := writeFile(t, t.TempDir(), "sci.json", `{
	  "functional_unit": {"label": "job", "quantity": 2},
	  "defaults": {"intensity": 100, "period_hours": 1},
	  "components": [{"type": "compute", "vcpus": 1, "utilization": 1.0,
	                  "memory_gb": 2}]
	}`)
	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.FunctionalUnit.Quantity != 2 {
		t.Fatalf("functional unit: %+v", manifest.FunctionalUnit)
	}
	component := manifest.Components[0]
	if component.Utilisation == nil || *component.Utilisation != 1.0 || component.MemoryGB != 2 {
		t.Fatalf("component: %+v", component)
	}
}

func TestManifestErrorsAreExplicit(t *testing.T) {
	if _, err := EstimateManifest(Manifest{Name: "empty"}, offlineConfig(), ""); err == nil {
		t.Error("a manifest with no components should be refused")
	}
	unknown := Manifest{Components: []Component{{Name: "x", Type: "quantum"}}}
	if _, err := EstimateManifest(unknown, offlineConfig(), ""); err == nil {
		t.Error("an unknown component type should be refused")
	}
	medium := Manifest{Components: []Component{{Type: "storage", Medium: "tape"}}}
	if _, err := EstimateManifest(medium, offlineConfig(), ""); err == nil {
		t.Error("an unknown storage medium should be refused")
	}
	if _, err := LoadManifest(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("a missing manifest should be refused")
	}
	broken := writeFile(t, t.TempDir(), "sci.yaml", "components: [oops\n")
	if _, err := LoadManifest(broken); err == nil {
		t.Error("invalid YAML should be refused")
	}
}

func TestComponentOverridesBeatTheDefaults(t *testing.T) {
	manifest := Manifest{
		FunctionalUnit: UnitSpec{Label: "run", Quantity: 1},
		Defaults:       Defaults{Intensity: 100, PeriodHours: 1, Provider: "aws"},
		Components: []Component{
			{Name: "elsewhere", Type: "compute", VCPUs: 1, Intensity: 900},
			{Name: "here", Type: "compute", VCPUs: 1},
		},
	}
	report, err := EstimateManifest(manifest, offlineConfig(), "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Components[0].Intensity.Value != 900 || report.Components[1].Intensity.Value != 100 {
		t.Fatalf("intensities: %g, %g", report.Components[0].Intensity.Value,
			report.Components[1].Intensity.Value)
	}
	// Two different grids means no single headline figure.
	if report.Intensity != nil {
		t.Error("a mixed-grid manifest should not claim one intensity")
	}
}

func TestScaffoldedManifestIsValidInputForEstimate(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "sci.yaml", RenderManifest("demo", nil, nil))
	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	report, err := EstimateManifest(manifest, offlineConfig(), path)
	if err != nil {
		t.Fatal(err)
	}
	if report.SCI <= 0 || report.FunctionalUnit.Label != "request" {
		t.Fatalf("%+v", report.FunctionalUnit)
	}
	if !strings.Contains(RenderManifest("demo", nil, nil), "# SCI manifest for demo") {
		t.Error("the scaffold should explain itself")
	}
}
