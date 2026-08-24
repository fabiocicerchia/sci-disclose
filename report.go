package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

// Output. A disclosure is only useful if it shows its working, so every
// renderer prints E, I, M and R next to the score.

// FormatNumber keeps very small and very large figures both readable.
func FormatNumber(value float64) string {
	magnitude := math.Abs(value)
	switch {
	case magnitude == 0:
		return "0"
	case magnitude >= 1000:
		return addThousands(fmt.Sprintf("%.0f", value))
	case magnitude >= 1:
		if value == math.Trunc(value) {
			return fmt.Sprintf("%.0f", value)
		}
		return fmt.Sprintf("%.3f", value)
	case magnitude >= 1e-4:
		return fmt.Sprintf("%.6f", value)
	default:
		return fmt.Sprintf("%.3e", value)
	}
}

func addThousands(digits string) string {
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	var parts []string
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	return sign + strings.Join(append([]string{digits}, parts...), ",")
}

// RenderText is the default human-readable disclosure.
func RenderText(report *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "sci: %s %s\n\n", FormatNumber(report.SCI), report.SCIUnit)
	fmt.Fprintf(&b, "  target      %s\n", report.Target.Description)
	if m := report.Measurement; m != nil {
		fmt.Fprintf(&b, "  ran in      %.2fs wall, %.2fs CPU, %.0f%% of %d reserved vCPU, "+
			"peak RSS %.0f MB\n", m.WallS, m.CPUS, m.Utilisation*100,
			report.Assumptions.ReservedVCPUs, m.PeakRSSGB*1024)
	}
	fmt.Fprintf(&b, "\n  E  energy        %s kWh [%s]\n",
		FormatNumber(report.EnergyKWh), report.EnergySource)
	for _, part := range report.EnergyBreakdown {
		fmt.Fprintf(&b, "       %-22s %s kWh\n", part.Name, FormatNumber(part.KWh))
	}
	if i := report.Intensity; i != nil {
		fmt.Fprintf(&b, "  I  intensity     %.0f gCO2e/kWh\n", i.Value)
		fmt.Fprintf(&b, "       %s\n", i.Source)
		if credit := i.Credit(); credit != "" {
			fmt.Fprintf(&b, "       %s\n", credit)
		}
	}
	fmt.Fprintf(&b, "  O  operational   %s gCO2e   (E x I)\n", FormatNumber(report.Operational))
	fmt.Fprintf(&b, "  M  embodied      %s gCO2e\n", FormatNumber(report.Embodied))
	if d := report.EmbodiedDetail; d != nil {
		fmt.Fprintf(&b, "       %.0f kg over %.0f y, time-share %.3e, resource-share %.3f\n",
			d.TotalEmbodiedKg, d.LifespanYears, d.TimeShare, d.ResourceShare)
	}
	fmt.Fprintf(&b, "  C  total         %s gCO2e   (O + M)\n", FormatNumber(report.Total))
	fmt.Fprintf(&b, "  R  per           %s x %s\n",
		FormatNumber(report.FunctionalUnit.Quantity), report.FunctionalUnit.Label)
	if report.FunctionalUnit.Source != "" {
		fmt.Fprintf(&b, "       %s\n", report.FunctionalUnit.Source)
	}
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 46))
	fmt.Fprintf(&b, "  SCI             %s %s\n", FormatNumber(report.SCI), report.SCIUnit)

	if len(report.Components) > 0 {
		fmt.Fprintf(&b, "\n  components\n")
		for _, row := range report.Components {
			fmt.Fprintf(&b, "    %-28s %s gCO2e (%s kWh)\n", row.Name,
				FormatNumber(row.Total), FormatNumber(row.EnergyKWh))
		}
	}
	if len(report.Notes) > 0 {
		fmt.Fprintln(&b)
		for _, note := range report.Notes {
			fmt.Fprintf(&b, "  ! %s\n", note)
		}
	}
	if report.Budget != nil {
		verdict := "within"
		if !report.Budget.Pass {
			verdict = "OVER"
		}
		fmt.Fprintf(&b, "\n  budget: %s — %s vs %s %s\n", verdict,
			FormatNumber(report.SCI), FormatNumber(report.Budget.Limit), report.SCIUnit)
	}
	return strings.TrimRight(b.String(), "\n")
}

// RenderMarkdown is the disclosure as a PR comment or README block.
func RenderMarkdown(report *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### SCI: %s %s\n\n", FormatNumber(report.SCI), report.SCIUnit)
	fmt.Fprintf(&b, "`%s`\n\n", report.Target.Description)
	fmt.Fprintf(&b, "| Term | Value | Source |\n| --- | --- | --- |\n")
	fmt.Fprintf(&b, "| E energy | %s kWh | %s |\n",
		FormatNumber(report.EnergyKWh), report.EnergySource)
	if i := report.Intensity; i != nil {
		fmt.Fprintf(&b, "| I intensity | %.0f gCO2e/kWh | %s |\n", i.Value, i.Source)
	}
	fmt.Fprintf(&b, "| O operational | %s gCO2e | E x I |\n", FormatNumber(report.Operational))
	fmt.Fprintf(&b, "| M embodied | %s gCO2e | amortised hardware |\n",
		FormatNumber(report.Embodied))
	fmt.Fprintf(&b, "| C total | %s gCO2e | O + M |\n", FormatNumber(report.Total))
	fmt.Fprintf(&b, "| R functional unit | %s x %s | declared |\n\n",
		FormatNumber(report.FunctionalUnit.Quantity), report.FunctionalUnit.Label)
	fmt.Fprintf(&b, "<details><summary>Software boundary and assumptions</summary>\n\n")
	for _, item := range report.Boundary {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	for _, note := range report.Notes {
		fmt.Fprintf(&b, "- _%s_\n", note)
	}
	if i := report.Intensity; i != nil {
		if credit := i.Credit(); credit != "" {
			fmt.Fprintf(&b, "- intensity data: %s\n", credit)
		}
		if i.Methodology != "" {
			fmt.Fprintf(&b, "- method: %s\n", i.Methodology)
		}
	}
	fmt.Fprintf(&b, "\n</details>\n\n_Measured with sci-disclose %s; SCI per ISO/IEC 21031:2024._",
		report.Version)
	return b.String()
}

// Emit writes the report in the requested format, to a file or to the writer.
func Emit(report *Report, format, output string, out io.Writer) error {
	var text string
	switch format {
	case "json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		text = string(data)
	case "markdown":
		text = RenderMarkdown(report)
	default:
		text = RenderText(report)
	}
	if output != "" {
		if err := os.WriteFile(output, []byte(text+"\n"), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "sci: wrote %s\n", output)
		return nil
	}
	fmt.Fprintln(out, text)
	return nil
}

// Comparison is the delta between two disclosures.
type Comparison struct {
	Before          float64 `json:"before"`
	After           float64 `json:"after"`
	Delta           float64 `json:"delta"`
	DeltaPct        float64 `json:"delta_pct"`
	Unit            string  `json:"unit"`
	Regression      bool    `json:"regression"`
	TolerancePct    float64 `json:"tolerance_pct"`
	IntensityMoved  bool    `json:"intensity_moved"`
	IntensityBefore float64 `json:"intensity_before,omitempty"`
	IntensityAfter  float64 `json:"intensity_after,omitempty"`
}

// CompareReports diffs two disclosures and flags a regression outside tolerance.
func CompareReports(before, after *Report, tolerance float64) Comparison {
	delta := after.SCI - before.SCI
	pct := math.Inf(1)
	if before.SCI != 0 {
		pct = delta / before.SCI * 100
	}
	comparison := Comparison{
		Before: before.SCI, After: after.SCI, Delta: delta, DeltaPct: pct,
		Unit: firstNonEmpty(after.SCIUnit, before.SCIUnit, "gCO2e"),
		// A tolerance of exactly 0 still means "no increase allowed".
		Regression: pct > tolerance, TolerancePct: tolerance,
	}
	if before.Intensity != nil && after.Intensity != nil {
		comparison.IntensityBefore = before.Intensity.Value
		comparison.IntensityAfter = after.Intensity.Value
		comparison.IntensityMoved = before.Intensity.Value != after.Intensity.Value
	}
	return comparison
}

// RenderComparison prints the delta, and warns when the grid moved underneath
// it: with a live intensity, two runs of identical code do not tie.
func RenderComparison(c Comparison) string {
	arrow := ""
	if c.Delta > 0 {
		arrow = "+"
	}
	verdict := "ok"
	if c.Regression {
		verdict = "REGRESSION"
	}
	lines := []string{
		fmt.Sprintf("before   %s %s", FormatNumber(c.Before), c.Unit),
		fmt.Sprintf("after    %s %s", FormatNumber(c.After), c.Unit),
		fmt.Sprintf("delta    %s%s (%s%.1f%%)  [%s]", arrow, FormatNumber(c.Delta),
			arrow, c.DeltaPct, verdict),
	}
	if c.IntensityMoved {
		lines = append(lines, fmt.Sprintf(
			"warning  I differs between the two runs (%.0f vs %.0f gCO2e/kWh): part of "+
				"this delta is the grid, not the code. Pin it with --intensity to "+
				"compare code alone.", c.IntensityBefore, c.IntensityAfter))
	}
	return strings.Join(lines, "\n")
}
