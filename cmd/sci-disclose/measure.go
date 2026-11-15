package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fabiocicerchia/sci-disclose/internal/config"
	"github.com/fabiocicerchia/sci-disclose/internal/discover"
	"github.com/fabiocicerchia/sci-disclose/internal/energy"
	"github.com/fabiocicerchia/sci-disclose/internal/harness"
	"github.com/fabiocicerchia/sci-disclose/internal/report"
	"github.com/fabiocicerchia/sci-disclose/internal/sci"
	"github.com/fabiocicerchia/sci-disclose/internal/units"
)

// Interpreters map a file extension onto the command that runs it.
var Interpreters = map[string][]string{
	".py": {"python3"}, ".js": {"node"}, ".mjs": {"node"}, ".ts": {"node"},
	".sh": {"sh"}, ".bash": {"bash"}, ".rb": {"ruby"}, ".php": {"php"},
	".pl": {"perl"}, ".lua": {"lua"}, ".go": {"go", "run"},
}

// energyBackend decides between RAPL and the model, taking an idle baseline
// first when one was asked for.
func energyBackend(cfg config.Config) (useRAPL bool, idleWatts float64, hasIdle bool, err error) {
	available := energy.RAPLAvailable()
	if cfg.EnergySource == "rapl" && !available {
		return false, 0, false, fmt.Errorf(
			"--energy rapl requested but no readable RAPL counters. They are " +
				"Linux/Intel-AMD only and usually root-owned:\n     " +
				"sudo chmod -R a+r /sys/class/powercap/intel-rapl*  (or run as root)")
	}
	useRAPL = available && cfg.EnergySource != "model"
	if useRAPL {
		idleWatts, hasIdle = energy.SampleIdleWatts(cfg.IdleSeconds)
	}
	return useRAPL, idleWatts, hasIdle, nil
}

const modelNote = "estimated from CPU time and reserved capacity, not measured at " +
	"the socket — see README for the error bars"

// measureCommand runs a workload and turns it into a disclosure.
func measureCommand(argv []string, dir string, target sci.Target, opts *options,
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
	counterBefore, err := scrapeUnitsBaseline(opts)
	if err != nil {
		return fail(out, err)
	}
	sample, err := energy.MeasureCommand(argv, dir, useRAPL, tap)
	if err != nil {
		return fail(out, err)
	}
	if !opts.unitsSet {
		if err := countRunUnits(opts, capturedText(captured), counterBefore); err != nil {
			return fail(out, err)
		}
	}
	notes := commandNotes(opts, sample.ExitCode, useRAPL)
	disclosure, err := sci.New(target, sample, *opts.cfg, idleWatts, hasIdle, notes)
	if err != nil {
		return fail(out, err)
	}
	within := sci.ApplyBudget(disclosure, opts.budget, opts.budgetSet)
	if err := emitDisclosure(disclosure, opts, out); err != nil {
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

// scrapeUnitsBaseline reads -units-metric before the run, so its delta can be
// taken afterwards. Without -units-metric there is nothing to read.
func scrapeUnitsBaseline(opts *options) (float64, error) {
	if opts.unitsMetric == "" {
		return 0, nil
	}
	if opts.unitsURL == "" {
		return 0, fmt.Errorf("-units-metric needs -units-url")
	}
	return units.ScrapeCounter(opts.unitsURL, opts.unitsMetric)
}

// countRunUnits reads the unit count from the finished workload into opts.cfg.
func countRunUnits(opts *options, stdout string, counterBefore float64) error {
	counted, err := opts.countUnits(stdout)
	if err != nil {
		return err
	}
	if opts.unitsMetric != "" {
		after, err := units.ScrapeCounter(opts.unitsURL, opts.unitsMetric)
		if err != nil {
			return err
		}
		delta := after - counterBefore
		if delta <= 0 {
			return fmt.Errorf("%s did not advance during the run "+
				"(%g to %g): nothing was served, so there is no rate to report",
				opts.unitsMetric, counterBefore, after)
		}
		counted = units.UnitSource{Count: delta,
			Origin: fmt.Sprintf("%s advanced by %s across the run",
				opts.unitsMetric, report.FormatNumber(delta))}
	}
	if counted.Count > 0 {
		opts.cfg.Units, opts.cfg.UnitSource = counted.Count, counted.Origin
	}
	return nil
}

// commandNotes lists the caveats a measured command's disclosure carries.
func commandNotes(opts *options, exitCode int, useRAPL bool) []string {
	var notes []string
	if opts.unitsSet && (opts.unitsFromStdout || opts.unitsFile != "" ||
		opts.unitsCmd != "" || opts.unitsMetric != "") {
		notes = append(notes, "--units was given explicitly, so the unit count was "+
			"not read from the workload")
	}
	if opts.unitsFromStdout {
		notes = append(notes, "stdout was duplicated to read the unit count, so the "+
			"workload saw a pipe rather than a terminal")
	}
	if exitCode != 0 {
		notes = append(notes, fmt.Sprintf("the workload exited %d: a failed run is not "+
			"a representative measurement", exitCode))
	}
	if !useRAPL {
		notes = append(notes, modelNote)
	}
	return notes
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
		return fail(out, fmt.Errorf("nothing to run — `sci-disclose run -- pytest -q`"))
	}
	target := sci.Target{Kind: "command", Description: strings.Join(argv, " ")}
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
		return fail(out, fmt.Errorf("which file? — `sci-disclose file bench.py`"))
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
	target := sci.Target{Kind: "file", Description: strings.Join(argv, " "), Path: path}
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
		return fail(out, fmt.Errorf("which function? — `sci-disclose func mypkg.bench:main -n 200`"))
	}
	if _, err := exec.LookPath(*python); err != nil {
		return fail(out, fmt.Errorf("interpreter not installed: %s", *python))
	}

	useRAPL, idleWatts, hasIdle, err := energyBackend(*opts.cfg)
	if err != nil {
		return fail(out, err)
	}
	sample, err := harness.MeasureFunction(*python, rest[0], *iterations, *warmup, useRAPL)
	if err != nil {
		return fail(out, err)
	}
	notes := funcNotes(*iterations, *warmup, sample)
	target := sci.Target{Kind: "function", Description: rest[0]}
	disclosure, err := sci.New(target, sample, *opts.cfg, idleWatts, hasIdle, notes)
	if err != nil {
		return fail(out, err)
	}
	within := sci.ApplyBudget(disclosure, opts.budget, opts.budgetSet)
	if err := emitDisclosure(disclosure, opts, out); err != nil {
		return fail(out, err)
	}
	if !within {
		return 1
	}
	return 0
}

// funcNotes lists the caveats a measured function's disclosure carries.
func funcNotes(iterations, warmup int, sample energy.Sample) []string {
	notes := []string{fmt.Sprintf("%d call(s) measured inside the interpreter, so "+
		"startup and import time are outside the bracket", iterations)}
	if warmup > 0 {
		notes = append(notes, fmt.Sprintf("%d warmup call(s) discarded first", warmup))
	}
	if !sample.HasRAPL {
		notes = append(notes, modelNote)
	}
	if sample.WallS < 0.05 {
		notes = append(notes, "the run was shorter than 50 ms: raise -n until the "+
			"measurement is stable")
	}
	return notes
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
		detected, from, ok := discover.DetectWorkload(absolute)
		if !ok {
			return fail(out, fmt.Errorf(
				"no workload found in this repo. A repository has no SCI until "+
					"something runs — pass -command, or declare the deployment with "+
					"`sci-disclose init` and `sci-disclose estimate`"))
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
	target := sci.Target{
		Kind:         "repo",
		Description:  filepath.Base(absolute) + ": " + strings.Join(argv, " "),
		Path:         absolute,
		DetectedFrom: why,
	}
	return measureCommand(argv, absolute, target, opts, out)
}
