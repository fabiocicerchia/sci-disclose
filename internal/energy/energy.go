// Package energy measures E: from the RAPL counters where they are readable,
// and from the coefficient model everywhere else. Which one produced a
// number is carried with the number, because the two are not comparable.
package energy

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
	"github.com/fabiocicerchia/sci-disclose/internal/config"
)

// Energy measurement.
//
// RAPL is the real thing: the CPU's own energy counters, in joules. It covers
// the package (cores + uncore) and, where the platform exposes it, DRAM —
// nothing else, and it is machine-wide rather than per-process. The model
// fallback is an estimate from CPU time and reserved capacity.

// RAPLRoot is the powercap sysfs root, overridden in tests.
var RAPLRoot = "/sys/class/powercap"

type raplDomain struct {
	path string
	name string
}

// raplDomains lists the top-level packages plus any separate DRAM subdomain.
// Core and uncore subdomains are deliberately skipped: their energy is already
// inside the package figure, so counting them would double up.
func raplDomains() []raplDomain {
	entries, err := os.ReadDir(RAPLRoot)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "intel-rapl:") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	var domains []raplDomain
	for _, name := range names {
		pkg := filepath.Join(RAPLRoot, name)
		if !fileExists(filepath.Join(pkg, "energy_uj")) {
			continue
		}
		domains = append(domains, raplDomain{path: pkg, name: domainName(pkg)})
		subs, err := os.ReadDir(pkg)
		if err != nil {
			continue
		}
		var subNames []string
		for _, sub := range subs {
			if strings.HasPrefix(sub.Name(), "intel-rapl:") {
				subNames = append(subNames, sub.Name())
			}
		}
		sort.Strings(subNames)
		for _, subName := range subNames {
			sub := filepath.Join(pkg, subName)
			if domainName(sub) == "dram" && fileExists(filepath.Join(sub, "energy_uj")) {
				domains = append(domains, raplDomain{path: sub, name: "dram"})
			}
		}
	}
	return domains
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readable(path string) bool {
	_, err := os.ReadFile(path) //nolint:gosec // a RAPL counter under /sys/class/powercap
	return err == nil
}

func domainName(path string) string {
	data, err := os.ReadFile(filepath.Join(path, "name")) //nolint:gosec // a RAPL domain name under /sys/class/powercap
	if err != nil {
		return filepath.Base(path)
	}
	return strings.TrimSpace(string(data))
}

func readUint(path string) (uint64, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a RAPL counter under /sys/class/powercap
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

// RAPLAvailable reports whether any counter can actually be read. They are
// usually root-owned, so presence is not the same as readability.
func RAPLAvailable() bool {
	for _, domain := range raplDomains() {
		if readable(filepath.Join(domain.path, "energy_uj")) {
			return true
		}
	}
	return false
}

// RAPLReader accumulates joules across domains, handling counter wraparound.
type RAPLReader struct {
	domains     []raplDomain
	start       map[string]uint64
	CoversDRAM  bool
	Joules      float64
	measuredAny bool
}

// NewRAPLReader snapshots every readable counter. Domains that cannot be read
// are dropped here rather than failing mid-measurement.
func NewRAPLReader() *RAPLReader {
	reader := &RAPLReader{start: map[string]uint64{}}
	for _, domain := range raplDomains() {
		value, err := readUint(filepath.Join(domain.path, "energy_uj"))
		if err != nil {
			continue
		}
		reader.domains = append(reader.domains, domain)
		reader.start[domain.path] = value
		if domain.name == "dram" {
			reader.CoversDRAM = true
		}
	}
	return reader
}

// Stop reads the counters again and stores the accumulated joules.
func (r *RAPLReader) Stop() {
	var totalUJ uint64
	for _, domain := range r.domains {
		end, err := readUint(filepath.Join(domain.path, "energy_uj"))
		if err != nil {
			continue
		}
		begin := r.start[domain.path]
		if end < begin { // the counter wrapped
			max, err := readUint(filepath.Join(domain.path, "max_energy_range_uj"))
			if err == nil {
				end += max
			}
		}
		totalUJ += end - begin
		r.measuredAny = true
	}
	r.Joules = float64(totalUJ) / 1e6
}

// Measured reports whether any domain contributed a reading.
func (r *RAPLReader) Measured() bool { return r.measuredAny }

// SampleIdleWatts measures the machine-wide RAPL draw with the workload not
// running, so the reported energy can be made marginal rather than total.
func SampleIdleWatts(seconds float64) (float64, bool) {
	if seconds <= 0 || !RAPLAvailable() {
		return 0, false
	}
	reader := NewRAPLReader()
	// Measuring elapsed time IS this function; an injected clock would be a
	// clock that cannot measure.
	started := time.Now() //nolint:forbidigo // see above
	time.Sleep(time.Duration(seconds * float64(time.Second)))
	elapsed := time.Since(started).Seconds()
	reader.Stop()
	if elapsed <= 0 || !reader.Measured() {
		return 0, false
	}
	return reader.Joules / elapsed, true
}

// Sample is what actually happened while the workload ran.
type Sample struct {
	WallS      float64
	CPUS       float64
	PeakRSSGB  float64
	RAPLJoules float64
	HasRAPL    bool
	CoversDRAM bool
	ExitCode   int
	Iterations float64
}

// MeasureCommand runs a command to completion and measures it from the
// outside: the child inherits stdio, environment and working directory, so it
// cannot tell it is being measured.
//
// The one exception is tap: when non-nil, stdout is duplicated into it as well
// as the terminal, so a unit count can be read from the workload's own output.
// That makes the child's stdout a pipe rather than a tty, which some programs
// notice — hence opt-in.
func MeasureCommand(argv []string, dir string, useRAPL bool, tap io.Writer) (Sample, error) {
	// Running the command the user asked for is what this tool does; there is
	// no version of it that does not launch what it was told to launch.
	//
	// context.Background(), deliberately: this is a one-shot CLI with no
	// cancellation to plumb through, and the process the user asked for runs
	// until it is done or until they interrupt the whole run.
	cmd := exec.CommandContext(context.Background(), argv[0], argv[1:]...) //nolint:gosec // see above
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if tap != nil {
		cmd.Stdout = io.MultiWriter(os.Stdout, tap)
	}
	cmd.Dir = dir

	var reader *RAPLReader
	if useRAPL {
		reader = NewRAPLReader()
	}
	started := time.Now() //nolint:forbidigo // the measurement itself; see SampleIdleWatts
	err := cmd.Run()
	wall := time.Since(started).Seconds()
	if reader != nil {
		reader.Stop()
	}
	if cmd.ProcessState == nil {
		return Sample{}, fmt.Errorf("could not run %s: %w", argv[0], err)
	}

	cpu, rssGB := processUsage(cmd.ProcessState)
	sample := Sample{
		WallS:      wall,
		CPUS:       cpu,
		PeakRSSGB:  rssGB,
		ExitCode:   cmd.ProcessState.ExitCode(),
		Iterations: 1,
	}
	if reader != nil && reader.Measured() {
		sample.RAPLJoules, sample.HasRAPL, sample.CoversDRAM = reader.Joules, true, reader.CoversDRAM
	}
	return sample, nil
}

// Utilisation is the fraction of the reserved vCPU capacity the workload used.
func Utilisation(cpuS, wallS float64, vcpus int) float64 {
	capacity := wallS * float64(vcpus)
	if capacity <= 0 {
		return 0
	}
	return min(1.0, cpuS/capacity)
}

// CPUkWh is Cloud Carbon Footprint's linear model: idle draw plus load, per vCPU.
func CPUkWh(wallH, util float64, vcpus int, profile coefficients.CPUProfile) float64 {
	wattsPerVCPU := profile.MinW + util*(profile.MaxW-profile.MinW)
	return float64(vcpus) * wattsPerVCPU * wallH / 1000
}

// MemorykWh applies the Cloud Jewels per-GB coefficient.
func MemorykWh(gb, wallH float64) float64 { return gb * coefficients.MemoryWPerGB * wallH / 1000 }

// StoragekWh applies the per-terabyte coefficient for the medium.
func StoragekWh(gb, wallH float64, medium string) float64 {
	return (gb / 1024) * coefficients.StorageWPerTB[medium] * wallH / 1000
}

// NetworkkWh applies the per-GB-transferred coefficient.
func NetworkkWh(gb float64) float64 { return gb * coefficients.NetworkKWhPerGB }

// Part is one named line of the energy breakdown. A slice rather than a
// map so both the report and its JSON keep a stable order.
type Part struct {
	Name string  `json:"name"`
	KWh  float64 `json:"kwh"`
}

// Energy is E for one measured run: the total, where it came from, and its parts.
type Energy struct {
	KWh         float64
	Source      string
	Breakdown   []Part
	Utilisation float64
	MemoryGB    float64
	Notes       []string
}

// ForSample turns a measurement into E, from RAPL joules where they were
// captured and from the model otherwise.
func ForSample(sample Sample, cfg config.Config, idleWatts float64, hasIdle bool) (Energy, error) {
	wallH := sample.WallS / 3600
	profile := cfg.Profile()
	memoryGB := cfg.MemoryGB
	if memoryGB < 0 {
		memoryGB = sample.PeakRSSGB
	}
	energy := Energy{
		Utilisation: Utilisation(sample.CPUS, sample.WallS, cfg.VCPUs),
		MemoryGB:    memoryGB,
	}

	if cfg.EnergySource == "rapl" && !sample.HasRAPL {
		return energy, fmt.Errorf("--energy rapl requested but RAPL counters are not " +
			"readable (Linux only, and usually root: see README)")
	}
	useRAPL := sample.HasRAPL && cfg.EnergySource != "model"

	if useRAPL {
		joules := sample.RAPLJoules
		if hasIdle {
			joules = max(0, joules-idleWatts*sample.WallS)
			energy.Notes = append(energy.Notes, fmt.Sprintf(
				"idle baseline of %.1f W subtracted; reported energy is marginal, not total",
				idleWatts))
		}
		energy.Source = "rapl"
		energy.Breakdown = append(energy.Breakdown, Part{"cpu", joules / 3.6e6})
		energy.Notes = append(energy.Notes,
			"RAPL counts the whole machine's CPU package, not just this workload — "+
				"measure on an otherwise idle host")
		if sample.CoversDRAM {
			energy.Notes = append(energy.Notes,
				"DRAM domain present: memory energy is inside the RAPL figure")
		} else {
			energy.Breakdown = append(energy.Breakdown,
				Part{"memory", MemorykWh(memoryGB, wallH)})
		}
	} else {
		energy.Source = "model"
		energy.Breakdown = append(energy.Breakdown,
			Part{"cpu", CPUkWh(wallH, energy.Utilisation, cfg.VCPUs, profile)},
			Part{"memory", MemorykWh(memoryGB, wallH)})
	}

	if cfg.StorageGB > 0 {
		energy.Breakdown = append(energy.Breakdown,
			Part{"storage", StoragekWh(cfg.StorageGB, wallH, cfg.StorageMedium)})
	}
	if cfg.NetworkGB > 0 {
		energy.Breakdown = append(energy.Breakdown,
			Part{"network", NetworkkWh(cfg.NetworkGB)})
	}

	var subtotal float64
	for _, part := range energy.Breakdown {
		subtotal += part.KWh
	}
	overhead := subtotal * (profile.PUE - 1)
	if overhead != 0 {
		energy.Breakdown = append(energy.Breakdown, Part{"datacentre_overhead", overhead})
	}
	energy.KWh = subtotal + overhead
	return energy, nil
}
