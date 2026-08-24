package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

const usage = `sci — Software Carbon Intensity (SCI) from the command line.

SCI, as specified by the Green Software Foundation and standardised as
ISO/IEC 21031:2024, is a rate, not a total:

    SCI = ((E x I) + M) per R

    E  energy consumed by the software system            kWh
    I  location-based grid carbon intensity              gCO2e/kWh
    M  embodied emissions of the hardware it runs on     gCO2e
    R  the functional unit the rate is expressed per     e.g. one API call

A repository, a file or a function has no SCI until something runs. So every
target either runs a workload and measures it, or reads a manifest in which
you declare one. Nothing here infers carbon from source code alone.

Targets:
    sci run -- pytest -q               a command
    sci file bench.py                  a script (interpreter by extension)
    sci func mypkg.bench:main -n 200   a function, per call
    sci repo .                         a repo, via its own test/build command
    sci estimate -f sci.yaml           a declared deployment (no execution)
    sci init .                         scaffold sci.yaml from the repo
    sci units -units N report.json     divide a measurement by a later count
    sci compare before.json after.json two runs, as a delta
    sci coefficients                   every constant used, with its source

Exit codes: 0 fine · 1 over budget or a regression · 2 usage or config error
· 3 the measured workload itself failed.
`

// Interpreters map a file extension onto the command that runs it.
var Interpreters = map[string][]string{
	".py": {"python3"}, ".js": {"node"}, ".mjs": {"node"}, ".ts": {"node"},
	".sh": {"sh"}, ".bash": {"bash"}, ".rb": {"ruby"}, ".php": {"php"},
	".pl": {"perl"}, ".lua": {"lua"}, ".go": {"go", "run"},
}

func main() { os.Exit(Run(os.Args[1:], os.Stdout)) }

// Run dispatches one invocation and returns its exit code.
func Run(args []string, out io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(out, usage)
		return 2
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(out, usage)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintf(out, "sci-disclose %s\n", Version)
		return 0
	case "run":
		return cmdRun(args[1:], out)
	case "file":
		return cmdFile(args[1:], out)
	case "func":
		return cmdFunc(args[1:], out)
	case "repo":
		return cmdRepo(args[1:], out)
	case "estimate":
		return cmdEstimate(args[1:], out)
	case "init":
		return cmdInit(args[1:], out)
	case "units":
		return cmdUnits(args[1:], out)
	case "compare":
		return cmdCompare(args[1:], out)
	case "coefficients":
		return cmdCoefficients(args[1:], out)
	default:
		fmt.Fprintf(out, "sci: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// options holds every flag shared by the measuring commands.
type options struct {
	cfg             *Config
	units           float64
	unitsSet        bool
	unitLabel       string
	unitsFromStdout bool
	unitsFile       string
	unitsCmd        string
	unitsPattern    string
	unitsMetric     string
	unitsURL        string
	format          string
	output          string
	budget          float64
	budgetSet       bool
}

// addUnitCountingFlags is only for the targets that run a command: `func` gets
// its count from -n, and `estimate` declares it in the manifest.
func addUnitCountingFlags(fs *flag.FlagSet, opts *options) {
	fs.BoolVar(&opts.unitsFromStdout, "units-from-stdout", false,
		"read the unit count from a marker the workload prints (see -units-pattern)")
	fs.StringVar(&opts.unitsFile, "units-file", "",
		"read the unit count from a file the workload wrote")
	fs.StringVar(&opts.unitsCmd, "units-cmd", "",
		"run this after the workload and read the unit count from its output")
	fs.StringVar(&opts.unitsPattern, "units-pattern", DefaultUnitPattern,
		"regexp whose first group captures the count")
	fs.StringVar(&opts.unitsMetric, "units-metric", "",
		"Prometheus counter whose delta across the run is the unit count")
	fs.StringVar(&opts.unitsURL, "units-url", "",
		"metrics endpoint to scrape for -units-metric")
}

// countUnits resolves R from whichever source was asked for. It runs after the
// workload, so none of it lands in the measurement.
func (o *options) countUnits(captured string) (UnitSource, error) {
	if !o.unitsFromStdout && o.unitsFile == "" && o.unitsCmd == "" {
		return UnitSource{}, nil
	}
	pattern, err := regexp.Compile(o.unitsPattern)
	if err != nil {
		return UnitSource{}, fmt.Errorf("-units-pattern is not a valid regexp: %w", err)
	}
	switch {
	case o.unitsFromStdout:
		count, ok := ScanUnits(captured, pattern)
		if !ok {
			return UnitSource{}, fmt.Errorf("the workload printed no unit count "+
				"matching %s — have it print e.g. `SCI-UNITS: 5000`", pattern)
		}
		return UnitSource{Count: count, Origin: "counted from the workload's stdout"}, nil
	case o.unitsFile != "":
		count, err := UnitsFromFile(o.unitsFile, pattern)
		if err != nil {
			return UnitSource{}, err
		}
		return UnitSource{Count: count, Origin: "counted from " + o.unitsFile}, nil
	default:
		count, err := UnitsFromCommand(o.unitsCmd, pattern)
		if err != nil {
			return UnitSource{}, err
		}
		return UnitSource{Count: count, Origin: "counted by `" + o.unitsCmd + "`"}, nil
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func addFlags(fs *flag.FlagSet) *options {
	cfg := NewConfig()
	opts := &options{cfg: &cfg}

	fs.StringVar(&cfg.Provider, "provider", cfg.Provider,
		"power profile and PUE to assume: "+strings.Join(ProviderNames, ", "))
	fs.Float64Var(&cfg.PUE, "pue", 0, "override the profile's PUE")
	fs.IntVar(&cfg.VCPUs, "vcpus", cfg.VCPUs, "vCPUs reserved for the workload")
	fs.IntVar(&cfg.TotalVCPUs, "total-vcpus", cfg.TotalVCPUs,
		"vCPUs on the host, for the embodied resource share")
	fs.Float64Var(&cfg.MemoryGB, "memory-gb", -1, "memory reserved (default: peak RSS)")
	fs.Float64Var(&cfg.StorageGB, "storage-gb", 0, "storage provisioned inside the boundary")
	fs.StringVar(&cfg.StorageMedium, "storage-medium", cfg.StorageMedium, "ssd or hdd")
	fs.Float64Var(&cfg.NetworkGB, "network-gb", 0, "data transferred inside the boundary")

	fs.Float64Var(&cfg.Intensity, "intensity", 0, "gCO2e/kWh, overrides every lookup")
	fs.StringVar(&cfg.Region, "region", "", "cloud region, e.g. eu-west-1")
	fs.StringVar(&cfg.Country, "country", "", "ISO-3166 country code, e.g. IE or IRL")
	fs.StringVar(&cfg.Zone, "zone", "", "bidding zone as COUNTRY/ZONE, e.g. IT/SICI")
	fs.StringVar(&cfg.IntensityBasis, "intensity-basis", cfg.IntensityBasis,
		"which figure to use: "+strings.Join(IntensityBases, ", "))
	fs.StringVar(&cfg.IntensityAPI, "intensity-api",
		envOr("SCI_INTENSITY_API", DefaultIntensityAPI), "carbon intensity API base URL")
	fs.BoolVar(&cfg.Offline, "offline", false,
		"never call the intensity API; use the bundled yearly averages")

	fs.StringVar(&cfg.HardwareName, "hardware", cfg.HardwareName,
		strings.Join(HardwareNames, ", "))
	fs.Float64Var(&cfg.EmbodiedKg, "embodied-kg", 0,
		"total embodied emissions of the whole device")
	fs.Float64Var(&cfg.LifespanYears, "lifespan-years", 0,
		"expected lifespan the device is amortised over")

	fs.StringVar(&cfg.EnergySource, "energy", cfg.EnergySource,
		"auto uses RAPL counters when readable, else the model (auto, rapl, model)")
	fs.Float64Var(&cfg.IdleSeconds, "idle-seconds", 0,
		"sample idle draw first and report marginal energy (RAPL only)")

	fs.Float64Var(&opts.units, "units", 0, "how many functional units the run delivered")
	fs.StringVar(&opts.unitLabel, "unit-label", "", "what one functional unit is")

	fs.StringVar(&opts.format, "format", "text", "text, json or markdown")
	fs.StringVar(&opts.output, "o", "", "write the report to a file")
	fs.StringVar(&opts.output, "output", "", "write the report to a file")
	fs.Float64Var(&opts.budget, "budget", 0,
		"fail (exit 1) if SCI exceeds this many gCO2e per unit")
	return opts
}

// finish applies the per-command defaults for R and notes whether --budget was set.
func (o *options) finish(fs *flag.FlagSet, defaultUnits float64, defaultLabel string) error {
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "budget":
			o.budgetSet = true
		case "units":
			o.unitsSet = true
		}
	})
	o.cfg.Units = defaultUnits
	if o.units > 0 {
		o.cfg.Units = o.units
	}
	o.cfg.UnitLabel = defaultLabel
	if o.unitLabel != "" {
		o.cfg.UnitLabel = o.unitLabel
	}
	return o.cfg.Validate()
}

func parse(fs *flag.FlagSet, args []string, out io.Writer) error {
	fs.SetOutput(out)
	return fs.Parse(args)
}

func fail(out io.Writer, err error) int {
	fmt.Fprintf(out, "sci: %v\n", err)
	return 2
}

// energyBackend decides between RAPL and the model, taking an idle baseline
// first when one was asked for.
func energyBackend(cfg Config) (useRAPL bool, idleWatts float64, hasIdle bool, err error) {
	available := RAPLAvailable()
	if cfg.EnergySource == "rapl" && !available {
		return false, 0, false, fmt.Errorf(
			"--energy rapl requested but no readable RAPL counters. They are " +
				"Linux/Intel-AMD only and usually root-owned:\n     " +
				"sudo chmod -R a+r /sys/class/powercap/intel-rapl*  (or run as root)")
	}
	useRAPL = available && cfg.EnergySource != "model"
	if useRAPL {
		idleWatts, hasIdle = SampleIdleWatts(cfg.IdleSeconds)
	}
	return useRAPL, idleWatts, hasIdle, nil
}

const modelNote = "estimated from CPU time and reserved capacity, not measured at " +
	"the socket — see README for the error bars"

// measureCommand runs a workload and turns it into a disclosure.
func measureCommand(argv []string, dir string, target Target, opts *options,
	out io.Writer) int {
	useRAPL, idleWatts, hasIdle, err := energyBackend(*opts.cfg)
	if err != nil {
		return fail(out, err)
	}
	var captured *bytes.Buffer
	var tap io.Writer
	if opts.unitsFromStdout {
		captured = &bytes.Buffer{}
		tap = captured
	}
	var counterBefore float64
	if opts.unitsMetric != "" {
		if opts.unitsURL == "" {
			return fail(out, fmt.Errorf("-units-metric needs -units-url"))
		}
		counterBefore, err = ScrapeCounter(opts.unitsURL, opts.unitsMetric)
		if err != nil {
			return fail(out, err)
		}
	}
	sample, err := MeasureCommand(argv, dir, useRAPL, tap)
	if err != nil {
		return fail(out, err)
	}
	var notes []string
	if !opts.unitsSet {
		counted, err := opts.countUnits(capturedText(captured))
		if err != nil {
			return fail(out, err)
		}
		if opts.unitsMetric != "" {
			after, err := ScrapeCounter(opts.unitsURL, opts.unitsMetric)
			if err != nil {
				return fail(out, err)
			}
			delta := after - counterBefore
			if delta <= 0 {
				return fail(out, fmt.Errorf("%s did not advance during the run "+
					"(%g to %g): nothing was served, so there is no rate to report",
					opts.unitsMetric, counterBefore, after))
			}
			counted = UnitSource{Count: delta,
				Origin: fmt.Sprintf("%s advanced by %s across the run",
					opts.unitsMetric, FormatNumber(delta))}
		}
		if counted.Count > 0 {
			opts.cfg.Units, opts.cfg.UnitSource = counted.Count, counted.Origin
		}
	} else if opts.unitsFromStdout || opts.unitsFile != "" || opts.unitsCmd != "" ||
		opts.unitsMetric != "" {
		notes = append(notes, "--units was given explicitly, so the unit count was "+
			"not read from the workload")
	}
	if opts.unitsFromStdout {
		notes = append(notes, "stdout was duplicated to read the unit count, so the "+
			"workload saw a pipe rather than a terminal")
	}
	if sample.ExitCode != 0 {
		notes = append(notes, fmt.Sprintf("the workload exited %d: a failed run is not "+
			"a representative measurement", sample.ExitCode))
	}
	if !useRAPL {
		notes = append(notes, modelNote)
	}
	report, err := SCIReport(target, sample, *opts.cfg, idleWatts, hasIdle, notes)
	if err != nil {
		return fail(out, err)
	}
	within := ApplyBudget(report, opts.budget, opts.budgetSet)
	if err := Emit(report, opts.format, opts.output, out); err != nil {
		return fail(out, err)
	}
	switch {
	case sample.ExitCode != 0:
		return 3
	case !within:
		return 1
	}
	return 0
}

func capturedText(buffer *bytes.Buffer) string {
	if buffer == nil {
		return ""
	}
	return buffer.String()
}

func cmdRun(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	opts := addFlags(fs)
	addUnitCountingFlags(fs, opts)
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	if err := opts.finish(fs, 1, "run"); err != nil {
		return fail(out, err)
	}
	argv := fs.Args()
	if len(argv) == 0 {
		return fail(out, fmt.Errorf("nothing to run — `sci run -- pytest -q`"))
	}
	target := Target{Kind: "command", Description: strings.Join(argv, " ")}
	return measureCommand(argv, "", target, opts, out)
}

func cmdFile(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("file", flag.ContinueOnError)
	interpreter := fs.String("interpreter", "", "override the interpreter")
	opts := addFlags(fs)
	addUnitCountingFlags(fs, opts)
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	if err := opts.finish(fs, 1, "execution"); err != nil {
		return fail(out, err)
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fail(out, fmt.Errorf("which file? — `sci file bench.py`"))
	}
	path := rest[0]
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return fail(out, fmt.Errorf("no such file: %s", path))
	}

	var prefix []string
	if *interpreter != "" {
		prefix = strings.Fields(*interpreter)
	} else {
		known, ok := Interpreters[strings.ToLower(filepath.Ext(path))]
		if !ok {
			return fail(out, fmt.Errorf("no interpreter known for %q; pass -interpreter",
				filepath.Ext(path)))
		}
		prefix = known
	}
	if _, err := exec.LookPath(prefix[0]); err != nil {
		return fail(out, fmt.Errorf("interpreter not installed: %s", prefix[0]))
	}
	argv := append(append([]string{}, prefix...), append([]string{path}, rest[1:]...)...)
	target := Target{Kind: "file", Description: strings.Join(argv, " "), Path: path}
	return measureCommand(argv, "", target, opts, out)
}

func cmdFunc(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("func", flag.ContinueOnError)
	iterations := fs.Int("n", 1, "how many calls to measure")
	fs.IntVar(iterations, "iterations", 1, "how many calls to measure")
	warmup := fs.Int("warmup", 0, "calls to make before measuring")
	python := fs.String("python", "python3", "interpreter running the harness")
	opts := addFlags(fs)
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	if *iterations < 1 {
		return fail(out, fmt.Errorf("-n must be at least 1"))
	}
	if err := opts.finish(fs, float64(*iterations), "call"); err != nil {
		return fail(out, err)
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fail(out, fmt.Errorf("which function? — `sci func mypkg.bench:main -n 200`"))
	}
	if _, err := exec.LookPath(*python); err != nil {
		return fail(out, fmt.Errorf("interpreter not installed: %s", *python))
	}

	useRAPL, idleWatts, hasIdle, err := energyBackend(*opts.cfg)
	if err != nil {
		return fail(out, err)
	}
	sample, err := MeasureFunction(*python, rest[0], *iterations, *warmup, useRAPL)
	if err != nil {
		return fail(out, err)
	}
	notes := []string{fmt.Sprintf("%d call(s) measured inside the interpreter, so "+
		"startup and import time are outside the bracket", *iterations)}
	if *warmup > 0 {
		notes = append(notes, fmt.Sprintf("%d warmup call(s) discarded first", *warmup))
	}
	if !sample.HasRAPL {
		notes = append(notes, modelNote)
	}
	if sample.WallS < 0.05 {
		notes = append(notes, "the run was shorter than 50 ms: raise -n until the "+
			"measurement is stable")
	}
	target := Target{Kind: "function", Description: rest[0]}
	report, err := SCIReport(target, sample, *opts.cfg, idleWatts, hasIdle, notes)
	if err != nil {
		return fail(out, err)
	}
	within := ApplyBudget(report, opts.budget, opts.budgetSet)
	if err := Emit(report, opts.format, opts.output, out); err != nil {
		return fail(out, err)
	}
	if !within {
		return 1
	}
	return 0
}

func cmdRepo(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("repo", flag.ContinueOnError)
	command := fs.String("command", "", "workload to run instead of the detected one")
	opts := addFlags(fs)
	addUnitCountingFlags(fs, opts)
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
	if info, err := os.Stat(absolute); err != nil || !info.IsDir() {
		return fail(out, fmt.Errorf("not a directory: %s", absolute))
	}

	var argv []string
	var why string
	if *command != "" {
		argv, why = strings.Fields(*command), "--command"
	} else {
		detected, from, ok := DetectWorkload(absolute)
		if !ok {
			return fail(out, fmt.Errorf(
				"no workload found in this repo. A repository has no SCI until "+
					"something runs — pass -command, or declare the deployment with "+
					"`sci init` and `sci estimate`"))
		}
		argv, why = detected, from
	}
	if _, err := exec.LookPath(argv[0]); err != nil {
		return fail(out, fmt.Errorf("%q is not installed, so the detected workload (%s) "+
			"cannot run", argv[0], strings.Join(argv, " ")))
	}
	if err := opts.finish(fs, 1, strings.Join(argv, " ")+" run"); err != nil {
		return fail(out, err)
	}
	target := Target{
		Kind:         "repo",
		Description:  filepath.Base(absolute) + ": " + strings.Join(argv, " "),
		Path:         absolute,
		DetectedFrom: why,
	}
	return measureCommand(argv, absolute, target, opts, out)
}

func cmdEstimate(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("estimate", flag.ContinueOnError)
	file := fs.String("f", "sci.yaml", "manifest to read")
	fs.StringVar(file, "file", "sci.yaml", "manifest to read")
	opts := addFlags(fs)
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	if err := opts.finish(fs, 1, "run"); err != nil {
		return fail(out, err)
	}
	manifest, err := LoadManifest(*file)
	if err != nil {
		return fail(out, err)
	}
	report, err := EstimateManifest(manifest, *opts.cfg, *file)
	if err != nil {
		return fail(out, err)
	}
	within := ApplyBudget(report, opts.budget, opts.budgetSet)
	if err := Emit(report, opts.format, opts.output, out); err != nil {
		return fail(out, err)
	}
	if !within {
		return 1
	}
	return 0
}

func cmdInit(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	output := fs.String("o", "sci.yaml", "where to write the manifest")
	fs.StringVar(output, "output", "sci.yaml", "where to write the manifest")
	force := fs.Bool("force", false, "overwrite an existing manifest")
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

	components, notes := ScanRepo(absolute)
	text := RenderManifest(filepath.Base(absolute), components, notes)
	if err := os.WriteFile(*output, []byte(text), 0o644); err != nil {
		return fail(out, err)
	}
	found := "nothing discovered — the scaffold has a placeholder component"
	if len(components) > 0 {
		found = fmt.Sprintf("%d component(s) discovered", len(components))
	}
	fmt.Fprintf(out, "sci: wrote %s (%s)\n", *output, found)
	fmt.Fprintf(out, "sci: fill in the functional unit and utilisation, then run "+
		"`sci estimate -f %s`\n", *output)
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
	comparison := CompareReports(before, after, *tolerance)
	if *format == "json" {
		data, err := json.MarshalIndent(comparison, "", "  ")
		if err != nil {
			return fail(out, err)
		}
		fmt.Fprintln(out, string(data))
	} else {
		fmt.Fprintln(out, RenderComparison(comparison))
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
	output := fs.String("o", "", "write the updated report to a file")
	fs.StringVar(output, "output", "", "write the updated report to a file")
	budget := fs.Float64("budget", 0, "fail (exit 1) if the resulting SCI exceeds this")
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return fail(out, fmt.Errorf("units takes one JSON report, after its flags: "+
			"`sci units -units 4300000 -unit-label request report.json`"))
	}
	if *units <= 0 {
		return fail(out, fmt.Errorf("-units must be a positive count"))
	}
	report, err := loadReport(rest[0])
	if err != nil {
		return fail(out, err)
	}
	if report.Total <= 0 {
		return fail(out, fmt.Errorf("%s carries no total to divide", rest[0]))
	}

	previous := report.FunctionalUnit
	report.FunctionalUnit = FunctionalUnit{
		Label:    firstNonEmpty(*label, previous.Label, "run"),
		Quantity: *units,
		Source:   "supplied after the measurement by `sci units`",
	}
	report.SCI = report.Total / *units
	report.SCIUnit = "gCO2e per " + report.FunctionalUnit.Label
	report.Notes = append(report.Notes, fmt.Sprintf(
		"R was reconciled after the fact: the measurement covered %s x %s, and E, I "+
			"and M are unchanged from it", FormatNumber(*units),
		report.FunctionalUnit.Label))

	budgetSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "budget" {
			budgetSet = true
		}
	})
	within := ApplyBudget(report, *budget, budgetSet)
	if err := Emit(report, *format, *output, out); err != nil {
		return fail(out, err)
	}
	if !within {
		return 1
	}
	return 0
}

func loadReport(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("%s is not a JSON report from `sci ... -format json`: %w",
			path, err)
	}
	return &report, nil
}

func cmdCoefficients(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("coefficients", flag.ContinueOnError)
	format := fs.String("format", "text", "text or json")
	if err := parse(fs, args, out); err != nil {
		return 2
	}
	if *format == "json" {
		sources := map[string]string{}
		for _, pair := range CoefficientSources {
			sources[pair[0]] = pair[1]
		}
		data, err := json.MarshalIndent(map[string]any{
			"cpu_profiles":             CPUProfiles,
			"memory_w_per_gb":          MemoryWPerGB,
			"network_kwh_per_gb":       NetworkKWhPerGB,
			"storage_w_per_tb":         StorageWPerTB,
			"hardware":                 Hardware,
			"grid_zones_gco2e_per_kwh": GridZones,
			"region_zone":              RegionZone,
			"region_country":           RegionCountry,
			"intensity_api":            DefaultIntensityAPI,
			"intensity_bases":          IntensityBases,
			"sources":                  sources,
		}, "", "  ")
		if err != nil {
			return fail(out, err)
		}
		fmt.Fprintln(out, string(data))
		return 0
	}

	fmt.Fprintln(out, "Power and PUE per provider (watts per vCPU):")
	for _, name := range ProviderNames {
		profile := CPUProfiles[name]
		fmt.Fprintf(out, "  %-8s %.2f idle  %.2f full   PUE %g\n",
			name, profile.MinW, profile.MaxW, profile.PUE)
	}
	fmt.Fprintf(out, "\nMemory %g W/GB · network %g kWh/GB · storage %g W/TB ssd, "+
		"%g W/TB hdd\n", MemoryWPerGB, NetworkKWhPerGB,
		StorageWPerTB["ssd"], StorageWPerTB["hdd"])
	fmt.Fprintln(out, "\nEmbodied emissions:")
	for _, name := range HardwareNames {
		device := Hardware[name]
		fmt.Fprintf(out, "  %-8s %.0f kgCO2e over %.0f years\n",
			name, device.EmbodiedKg, device.LifespanYears)
	}
	fmt.Fprintf(out, "\nCarbon intensity: %s, last-hour readings per country and "+
		"bidding zone.\n  Default figure: %s. Cached for an hour; falls back to the "+
		"bundled\n  yearly averages below when unreachable or with -offline.\n",
		DefaultIntensityAPI, IntensityBases[0])
	fmt.Fprintln(out, "\nBundled grid intensity (gCO2e/kWh, yearly averages):")
	zones := make([]string, 0, len(GridZones))
	for zone := range GridZones {
		zones = append(zones, zone)
	}
	sort.Slice(zones, func(i, j int) bool { return GridZones[zones[i]] < GridZones[zones[j]] })
	for i := 0; i < len(zones); i += 4 {
		var row []string
		for _, zone := range zones[i:min(i+4, len(zones))] {
			row = append(row, fmt.Sprintf("%-8s%4.0f", zone, GridZones[zone]))
		}
		fmt.Fprintln(out, "  "+strings.Join(row, "  "))
	}
	fmt.Fprintf(out, "\n%d cloud regions map onto those zones and onto countries "+
		"(`sci coefficients -format json` lists them).\n", len(RegionZone))
	fmt.Fprintln(out, "\nSources:")
	for _, pair := range CoefficientSources {
		fmt.Fprintf(out, "  %s\n    %s\n", pair[0], pair[1])
	}
	fmt.Fprintf(out, "\nHost: %s %s, %d vCPU. RAPL counters readable: %t\n",
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), RAPLAvailable())
	return 0
}
