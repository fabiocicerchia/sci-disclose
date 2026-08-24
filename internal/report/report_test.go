package report

import (
	"strings"
	"testing"

	"github.com/fabiocicerchia/sci-disclose/internal/config"
	"github.com/fabiocicerchia/sci-disclose/internal/energy"
	"github.com/fabiocicerchia/sci-disclose/internal/grid"
	"github.com/fabiocicerchia/sci-disclose/internal/sci"
	"github.com/fabiocicerchia/sci-disclose/internal/testutil"
)

func sampleReport(t *testing.T) *sci.Report {
	t.Helper()
	cfg := testutil.Config(func(c *config.Config) {
		c.VCPUs, c.TotalVCPUs = 1, 4
		c.Intensity, c.Units, c.UnitLabel = 300, 10, "request"
	})
	report, err := sci.SCIReport(sci.Target{Kind: "test", Description: "demo"},
		energy.Sample{WallS: 60, CPUS: 30, PeakRSSGB: 0.5}, cfg, 0, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func TestTextAndMarkdownShowTheWorking(t *testing.T) {
	report := sampleReport(t)
	text := RenderText(report)
	for _, want := range []string{"E  energy", "I  intensity", "M  embodied",
		"R  per", "SCI"} {
		if !strings.Contains(text, want) {
			t.Errorf("text output is missing %q", want)
		}
	}
	markdown := RenderMarkdown(report)
	if !strings.HasPrefix(markdown, "### SCI:") ||
		!strings.Contains(markdown, "| R functional unit |") {
		t.Errorf("markdown output:\n%s", markdown)
	}
}

func TestNumberFormattingStaysReadable(t *testing.T) {
	cases := map[float64]string{
		0: "0", 200: "200", 1000: "1,000", 1234567: "1,234,567", 1.5: "1.500",
		-2500: "-2,500",
	}
	for value, want := range cases {
		if got := FormatNumber(value); got != want {
			t.Errorf("FormatNumber(%g) = %q, want %q", value, got, want)
		}
	}
	if got := FormatNumber(0.000000123); !strings.Contains(got, "e-") {
		t.Errorf("tiny numbers should go scientific: %q", got)
	}
}

func TestCompareFlagsARegressionOutsideTolerance(t *testing.T) {
	before := &sci.Report{SCI: 1.0, SCIUnit: "gCO2e per run"}
	after := &sci.Report{SCI: 1.2, SCIUnit: "gCO2e per run"}
	result := CompareReports(before, after, 0)
	testutil.Approx(t, result.DeltaPct, 20, 1e-9, "delta percent")
	if !result.Regression {
		t.Error("a 20% increase is a regression at zero tolerance")
	}
	if CompareReports(before, after, 25).Regression {
		t.Error("25% tolerance should absorb a 20% increase")
	}
	if !strings.Contains(RenderComparison(result), "REGRESSION") {
		t.Error("the verdict should be visible")
	}
}

func TestCompareWarnsWhenTheGridMovedUnderneathTheCode(t *testing.T) {
	before := &sci.Report{SCI: 1.0, SCIUnit: "gCO2e per run",
		Intensity: &grid.Intensity{Value: 100}}
	after := &sci.Report{SCI: 2.0, SCIUnit: "gCO2e per run",
		Intensity: &grid.Intensity{Value: 200}}
	result := CompareReports(before, after, 0)
	if !result.IntensityMoved {
		t.Fatal("a changed intensity should be detected")
	}
	rendered := RenderComparison(result)
	if !strings.Contains(rendered, "part of this delta is the grid") {
		t.Errorf("the warning is missing:\n%s", rendered)
	}
	same := CompareReports(before, &sci.Report{SCI: 2.0, Intensity: &grid.Intensity{Value: 100}}, 0)
	if same.IntensityMoved || strings.Contains(RenderComparison(same), "warning") {
		t.Error("an unchanged intensity needs no warning")
	}
}
