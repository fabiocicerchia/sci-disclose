package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
)

const usage = `sci-disclose — Software Carbon grid.Intensity (SCI) from the command line.

SCI, as specified by the Green Software Foundation and standardised as
ISO/IEC 21031:2024, is a rate, not a total:

    SCI = ((E x I) + M) per R

    E  energy consumed by the software system            kWh
    I  location-based grid carbon intensity              gCO2e/kWh
    M  embodied emissions of the hardware it runs on     gCO2e
    R  the functional unit the rate is expressed per     e.g. one API call

A repository, a file or a function has no SCI until something runs. So every
target either runs a workload and measures it, or reads a declared in which
you declare one. Nothing here infers carbon from source code alone.

Targets:
    sci-disclose run -- pytest -q               a command
    sci-disclose file bench.py                  a script (interpreter by extension)
    sci-disclose func mypkg.bench:main -n 200   a function, per call
    sci-disclose repo .                         a repo, via its own test/build command
    sci-disclose estimate -f sci.yaml           a declared deployment (no execution)
    sci-disclose init .                         scaffold sci.yaml from the repo
    sci-disclose units -units N disclosure.json     divide a measurement by a later count
    sci-disclose compare before.json after.json two runs, as a delta
    sci-disclose coefficients                   every constant used, with its source

Exit codes: 0 fine · 1 over budget or a regression · 2 usage or config error
· 3 the measured workload itself failed.
`

func main() { os.Exit(Run(os.Args[1:], os.Stdout)) }

// subcommands maps each subcommand onto the function that runs it.
var subcommands = map[string]func([]string, io.Writer) int{
	"run": cmdRun, "file": cmdFile, "func": cmdFunc, "repo": cmdRepo,
	"estimate": cmdEstimate, "init": cmdInit, "units": cmdUnits,
	"compare": cmdCompare, "coefficients": cmdCoefficients,
}

// Run dispatches one invocation and returns its exit code.
func Run(args []string, out io.Writer) int {
	if len(args) == 0 {
		emit(out, usage)
		return 2
	}
	switch args[0] {
	case "-h", "--help", "help":
		emit(out, usage)
		return 0
	case "-v", "--version", "version":
		emitf(out, "sci-disclose %s\n", coefficients.Version)
		return 0
	}
	if cmd, ok := subcommands[args[0]]; ok {
		return cmd(args[1:], out)
	}
	emitf(out, "sci: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func parse(fs *flag.FlagSet, args []string, out io.Writer) error {
	fs.SetOutput(out)
	return fs.Parse(args)
}

func fail(out io.Writer, err error) int {
	emitf(out, "sci: %v\n", err)
	return 2
}

// emit, emitf and emitln write to the CLI's output.
//
// A failed write to the process's own stdout cannot be reported: the only
// place to report it is the stream that just failed. The error is dropped
// here, once, rather than at each call site.

func emit(w io.Writer, s string) { _, _ = fmt.Fprint(w, s) } //nolint:errcheck // see above

func emitf(w io.Writer, format string, a ...any) { _, _ = fmt.Fprintf(w, format, a...) } //nolint:errcheck // see above

func emitln(w io.Writer, a ...any) { _, _ = fmt.Fprintln(w, a...) } //nolint:errcheck // see above
