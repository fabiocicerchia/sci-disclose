package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
	"github.com/fabiocicerchia/sci-disclose/internal/config"
	"github.com/fabiocicerchia/sci-disclose/internal/units"
)

// options holds every flag shared by the measuring commands.
type options struct {
	cfg             *config.Config
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
	markdown        string
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
	fs.StringVar(&opts.unitsPattern, "units-pattern", units.DefaultUnitPattern,
		"regexp whose first group captures the count")
	fs.StringVar(&opts.unitsMetric, "units-metric", "",
		"Prometheus counter whose delta across the run is the unit count")
	fs.StringVar(&opts.unitsURL, "units-url", "",
		"metrics endpoint to scrape for -units-metric")
}

// countUnits resolves R from whichever source was asked for. It runs after the
// workload, so none of it lands in the measurement.
func (o *options) countUnits(captured string) (units.UnitSource, error) {
	if !o.unitsFromStdout && o.unitsFile == "" && o.unitsCmd == "" {
		return units.UnitSource{}, nil
	}
	pattern, err := regexp.Compile(o.unitsPattern)
	if err != nil {
		return units.UnitSource{}, fmt.Errorf("-units-pattern is not a valid regexp: %w", err)
	}
	switch {
	case o.unitsFromStdout:
		count, ok := units.ScanUnits(captured, pattern)
		if !ok {
			return units.UnitSource{}, fmt.Errorf("the workload printed no unit count "+
				"matching %s — have it print e.g. `SCI-UNITS: 5000`", pattern)
		}
		return units.UnitSource{Count: count, Origin: "counted from the workload's stdout"}, nil
	case o.unitsFile != "":
		count, err := units.FromFile(o.unitsFile, pattern)
		if err != nil {
			return units.UnitSource{}, err
		}
		return units.UnitSource{Count: count, Origin: "counted from " + o.unitsFile}, nil
	default:
		count, err := units.FromCommand(o.unitsCmd, pattern)
		if err != nil {
			return units.UnitSource{}, err
		}
		return units.UnitSource{Count: count, Origin: "counted by `" + o.unitsCmd + "`"}, nil
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func addFlags(fs *flag.FlagSet) *options {
	cfg := config.NewConfig()
	opts := &options{cfg: &cfg}

	fs.StringVar(&cfg.Provider, "provider", cfg.Provider,
		"power profile and PUE to assume: "+strings.Join(coefficients.ProviderNames, ", "))
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
		"which figure to use: "+strings.Join(coefficients.IntensityBases, ", "))
	fs.StringVar(&cfg.IntensityAPI, "intensity-api",
		envOr("SCI_INTENSITY_API", coefficients.DefaultIntensityAPI), "carbon intensity API base URL")
	fs.BoolVar(&cfg.Offline, "offline", false,
		"never call the intensity API; use the bundled yearly averages")

	fs.StringVar(&cfg.HardwareName, "hardware", cfg.HardwareName,
		strings.Join(coefficients.HardwareNames, ", "))
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
	fs.StringVar(&opts.output, "o", "", "write the disclosure to a file")
	fs.StringVar(&opts.output, "output", "", "write the disclosure to a file")
	fs.StringVar(&opts.markdown, "markdown", "", "also write the disclosure as markdown to this file")
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
