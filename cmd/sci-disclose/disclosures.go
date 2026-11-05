package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
	"github.com/fabiocicerchia/sci-disclose/internal/discover"
	"github.com/fabiocicerchia/sci-disclose/internal/energy"
	"github.com/fabiocicerchia/sci-disclose/internal/manifest"
	"github.com/fabiocicerchia/sci-disclose/internal/report"
	"github.com/fabiocicerchia/sci-disclose/internal/sci"
)

func cmdEstimate(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("estimate", flag.ContinueOnError)
	file := fs.String("f", "sci.yaml", "declared to read")
	fs.StringVar(file, "file", "sci.yaml", "declared to read")
	opts := addFlags(fs)
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	if err := opts.finish(fs, 1, "run"); err != nil {
		return fail(out, err)
	}
	declared, err := manifest.LoadManifest(*file)
	if err != nil {
		return fail(out, err)
	}
	disclosure, err := manifest.EstimateManifest(declared, *opts.cfg, *file)
	if err != nil {
		return fail(out, err)
	}
	within := sci.ApplyBudget(disclosure, opts.budget, opts.budgetSet)
	if err := report.Emit(disclosure, opts.format, opts.output, out); err != nil {
		return fail(out, err)
	}
	if !within {
		return 1
	}
	return 0
}

func cmdInit(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	output := fs.String("o", "sci.yaml", "where to write the declared")
	fs.StringVar(output, "output", "sci.yaml", "where to write the declared")
	force := fs.Bool("force", false, "overwrite an existing declared")
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	root := "."
	if rest := fs.Args(); len(rest) > 0 {
		root = rest[0]
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return fail(out, err)
	}
	if _, err := os.Stat(*output); err == nil && !*force {
		return fail(out, fmt.Errorf("%s already exists (use -force to overwrite)", *output))
	}

	components, notes := discover.ScanRepo(absolute)
	text := discover.RenderManifest(filepath.Base(absolute), components, notes)
	// 0644 like any other scaffolded config: sci.yaml is written to be edited
	// and committed, not kept private. gosec G306.
	//nolint:gosec // scaffolded config, meant to be committed
	if err := os.WriteFile(*output, []byte(text), 0o644); err != nil {
		return fail(out, err)
	}
	found := "nothing discovered — the scaffold has a placeholder component"
	if len(components) > 0 {
		found = fmt.Sprintf("%d component(s) discovered", len(components))
	}
	emitf(out, "sci: wrote %s (%s)\n", *output, found)
	emitf(out, "sci: fill in the functional unit and utilisation, then run "+
		"`sci-disclose estimate -f %s`\n", *output)
	return 0
}

func cmdCompare(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	tolerance := fs.Float64("tolerance", 0,
		"percent increase tolerated before it counts as a regression")
	failOnRegression := fs.Bool("fail-on-regression", false, "exit 1 on a regression")
	format := fs.String("format", "text", "text or json")
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) != 2 {
		return fail(out, fmt.Errorf("compare takes two JSON reports: before and after"))
	}
	before, err := loadReport(rest[0])
	if err != nil {
		return fail(out, err)
	}
	after, err := loadReport(rest[1])
	if err != nil {
		return fail(out, err)
	}
	comparison := report.CompareReports(before, after, *tolerance)
	if *format == "json" {
		data, err := json.MarshalIndent(comparison, "", "  ")
		if err != nil {
			return fail(out, err)
		}
		emitln(out, string(data))
	} else {
		emitln(out, report.RenderComparison(comparison))
	}
	if comparison.Regression && *failOnRegression {
		return 1
	}
	return 0
}

// cmdUnits divides an existing disclosure by a count that only became known
// afterwards. C is fixed the moment the workload ends; R sometimes is not, and
// a service's units accrue for a month after the measurement. Rather than
// guess, measure now and reconcile later — the carbon is already correct.
func cmdUnits(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("units", flag.ContinueOnError)
	units := fs.Float64("units", 0, "how many functional units that measurement covered")
	label := fs.String("unit-label", "", "what one functional unit is")
	format := fs.String("format", "text", "text, json or markdown")
	output := fs.String("o", "", "write the updated disclosure to a file")
	fs.StringVar(output, "output", "", "write the updated disclosure to a file")
	budget := fs.Float64("budget", 0, "fail (exit 1) if the resulting SCI exceeds this")
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return fail(out, fmt.Errorf("units takes one JSON disclosure, after its flags: "+
			"`sci-disclose units -units 4300000 -unit-label request disclosure.json`"))
	}
	if *units <= 0 {
		return fail(out, fmt.Errorf("-units must be a positive count"))
	}
	disclosure, err := loadReport(rest[0])
	if err != nil {
		return fail(out, err)
	}
	if disclosure.Total <= 0 {
		return fail(out, fmt.Errorf("%s carries no total to divide", rest[0]))
	}

	previous := disclosure.FunctionalUnit
	disclosure.FunctionalUnit = sci.FunctionalUnit{
		Label:    cmp.Or(*label, previous.Label, "run"),
		Quantity: *units,
		Source:   "supplied after the measurement by `sci-disclose units`",
	}
	disclosure.SCI = disclosure.Total / *units
	disclosure.SCIUnit = "gCO2e per " + disclosure.FunctionalUnit.Label
	disclosure.Notes = append(disclosure.Notes, fmt.Sprintf(
		"R was reconciled after the fact: the measurement covered %s x %s, and E, I "+
			"and M are unchanged from it", report.FormatNumber(*units),
		disclosure.FunctionalUnit.Label))

	budgetSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "budget" {
			budgetSet = true
		}
	})
	within := sci.ApplyBudget(disclosure, *budget, budgetSet)
	if err := report.Emit(disclosure, *format, *output, out); err != nil {
		return fail(out, err)
	}
	if !within {
		return 1
	}
	return 0
}

func loadReport(path string) (*sci.Report, error) {
	data, err := os.ReadFile(path) //nolint:gosec // the disclosure the user passed to -f
	if err != nil {
		return nil, err
	}
	var disclosure sci.Report
	if err := json.Unmarshal(data, &disclosure); err != nil {
		return nil, fmt.Errorf("%s is not a JSON disclosure from `sci ... -format json`: %w",
			path, err)
	}
	return &disclosure, nil
}

func cmdCoefficients(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("coefficients", flag.ContinueOnError)
	format := fs.String("format", "text", "text or json")
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	if *format == "json" {
		sources := map[string]string{}
		for _, pair := range coefficients.CoefficientSources {
			sources[pair[0]] = pair[1]
		}
		data, err := json.MarshalIndent(map[string]any{
			"cpu_profiles":             coefficients.CPUProfiles,
			"memory_w_per_gb":          coefficients.MemoryWPerGB,
			"network_kwh_per_gb":       coefficients.NetworkKWhPerGB,
			"storage_w_per_tb":         coefficients.StorageWPerTB,
			"hardware":                 coefficients.Hardware,
			"grid_zones_gco2e_per_kwh": coefficients.GridZones,
			"region_zone":              coefficients.RegionZone,
			"region_country":           coefficients.RegionCountry,
			"intensity_api":            coefficients.DefaultIntensityAPI,
			"intensity_bases":          coefficients.IntensityBases,
			"sources":                  sources,
		}, "", "  ")
		if err != nil {
			return fail(out, err)
		}
		emitln(out, string(data))
		return 0
	}

	emitln(out, "Power and PUE per provider (watts per vCPU):")
	for _, name := range coefficients.ProviderNames {
		profile := coefficients.CPUProfiles[name]
		emitf(out, "  %-8s %.2f idle  %.2f full   PUE %g\n",
			name, profile.MinW, profile.MaxW, profile.PUE)
	}
	emitf(out, "\nMemory %g W/GB · network %g kWh/GB · storage %g W/TB ssd, "+
		"%g W/TB hdd\n", coefficients.MemoryWPerGB, coefficients.NetworkKWhPerGB,
		coefficients.StorageWPerTB["ssd"], coefficients.StorageWPerTB["hdd"])
	emitln(out, "\nEmbodied emissions:")
	for _, name := range coefficients.HardwareNames {
		device := coefficients.Hardware[name]
		emitf(out, "  %-8s %.0f kgCO2e over %.0f years\n",
			name, device.EmbodiedKg, device.LifespanYears)
	}
	emitf(out, "\nCarbon intensity: %s, last-hour readings per country and "+
		"bidding zone.\n  Default figure: %s. Cached for an hour; falls back to the "+
		"bundled\n  yearly averages below when unreachable or with -offline.\n",
		coefficients.DefaultIntensityAPI, coefficients.IntensityBases[0])
	emitln(out, "\nBundled grid intensity (gCO2e/kWh, yearly averages):")
	zones := make([]string, 0, len(coefficients.GridZones))
	for zone := range coefficients.GridZones {
		zones = append(zones, zone)
	}
	sort.Slice(zones, func(i, j int) bool { return coefficients.GridZones[zones[i]] < coefficients.GridZones[zones[j]] })
	for i := 0; i < len(zones); i += 4 {
		var row []string
		for _, zone := range zones[i:min(i+4, len(zones))] {
			row = append(row, fmt.Sprintf("%-8s%4.0f", zone, coefficients.GridZones[zone]))
		}
		emitln(out, "  "+strings.Join(row, "  "))
	}
	emitf(out, "\n%d cloud regions map onto those zones and onto countries "+
		"(`sci-disclose coefficients -format json` lists them).\n", len(coefficients.RegionZone))
	emitln(out, "\nSources:")
	for _, pair := range coefficients.CoefficientSources {
		emitf(out, "  %s\n    %s\n", pair[0], pair[1])
	}
	emitf(out, "\nHost: %s %s, %d vCPU. RAPL counters readable: %t\n",
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), energy.RAPLAvailable())
	return 0
}
