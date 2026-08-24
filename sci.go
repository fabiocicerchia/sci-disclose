package main

import (
	"fmt"
	"runtime"
)

// The SCI equation: SCI = ((E x I) + M) per R.

// Config carries everything the equation needs beyond the measurement itself:
// the software boundary, the grid, the hardware and the functional unit.
type Config struct {
	Provider       string
	PUE            float64 // 0 = use the provider profile's PUE
	VCPUs          int     // vCPUs reserved for the workload
	TotalVCPUs     int     // vCPUs on the host, for the embodied resource share
	MemoryGB       float64 // negative = use the measured peak RSS
	StorageGB      float64
	StorageMedium  string
	NetworkGB      float64
	Intensity      float64 // 0 = resolve from the API or the bundled table
	Region         string
	Country        string
	Zone           string
	IntensityBasis string
	IntensityAPI   string
	Offline        bool
	HardwareName   string
	EmbodiedKg     float64 // 0 = use the hardware preset
	LifespanYears  float64 // 0 = use the hardware preset
	EnergySource   string  // auto | rapl | model
	IdleSeconds    float64
	Units          float64
	UnitLabel      string
	UnitSource     string // how the count was arrived at, for the disclosure
}

// NewConfig returns the defaults: the whole host reserved, memory from the
// measurement, energy from whichever backend is available.
func NewConfig() Config {
	cpus := runtime.NumCPU()
	return Config{
		Provider:       "onprem",
		VCPUs:          cpus,
		TotalVCPUs:     cpus,
		MemoryGB:       -1,
		StorageMedium:  "ssd",
		IntensityBasis: IntensityBases[0],
		IntensityAPI:   DefaultIntensityAPI,
		HardwareName:   "server",
		EnergySource:   "auto",
		Units:          1,
		UnitLabel:      "run",
	}
}

// Profile is the provider power profile with any PUE override applied.
func (c Config) Profile() CPUProfile {
	profile := CPUProfiles[c.Provider]
	if c.PUE > 0 {
		profile.PUE = c.PUE
	}
	return profile
}

// DeviceSpec is the hardware preset with any embodied overrides applied.
func (c Config) DeviceSpec() Device {
	device := Hardware[c.HardwareName]
	if c.EmbodiedKg > 0 {
		device.EmbodiedKg = c.EmbodiedKg
	}
	if c.LifespanYears > 0 {
		device.LifespanYears = c.LifespanYears
	}
	return device
}

// Validate rejects flag combinations the tool cannot honour, before anything runs.
func (c Config) Validate() error {
	if _, ok := CPUProfiles[c.Provider]; !ok {
		return fmt.Errorf("unknown provider %q (one of: %v)", c.Provider, ProviderNames)
	}
	if _, ok := Hardware[c.HardwareName]; !ok {
		return fmt.Errorf("unknown hardware %q (one of: %v)", c.HardwareName, HardwareNames)
	}
	if _, ok := StorageWPerTB[c.StorageMedium]; !ok {
		return fmt.Errorf("unknown storage medium %q (ssd or hdd)", c.StorageMedium)
	}
	switch c.EnergySource {
	case "auto", "rapl", "model":
	default:
		return fmt.Errorf("unknown energy source %q (auto, rapl or model)", c.EnergySource)
	}
	return nil
}

// EmbodiedDetail is M and the shares it was derived from.
type EmbodiedDetail struct {
	GCO2e           float64 `json:"gco2e"`
	TotalEmbodiedKg float64 `json:"total_embodied_kg"`
	LifespanYears   float64 `json:"lifespan_years"`
	TimeShare       float64 `json:"time_share"`
	ResourceShare   float64 `json:"resource_share"`
}

// EmbodiedGCO2e is M = TE x time-share x resource-share, per the specification.
func EmbodiedGCO2e(cfg Config, wallH float64) EmbodiedDetail {
	device := cfg.DeviceSpec()
	timeShare := wallH / (device.LifespanYears * HoursPerYear)
	resourceShare := 1.0
	if cfg.TotalVCPUs > 0 {
		resourceShare = float64(cfg.VCPUs) / float64(cfg.TotalVCPUs)
	}
	return EmbodiedDetail{
		GCO2e:           device.EmbodiedKg * 1000 * timeShare * resourceShare,
		TotalEmbodiedKg: device.EmbodiedKg,
		LifespanYears:   device.LifespanYears,
		TimeShare:       timeShare,
		ResourceShare:   resourceShare,
	}
}

// Target names what was measured.
type Target struct {
	Kind         string `json:"kind"`
	Description  string `json:"description"`
	Path         string `json:"path,omitempty"`
	DetectedFrom string `json:"detected_from,omitempty"`
}

// Measurement is what the bracket around the workload saw.
type Measurement struct {
	WallS       float64 `json:"wall_s"`
	CPUS        float64 `json:"cpu_s"`
	PeakRSSGB   float64 `json:"peak_rss_gb"`
	Utilisation float64 `json:"utilisation"`
	ExitCode    int     `json:"exit_code"`
	Host        string  `json:"host"`
}

// FunctionalUnit is R: what the rate is expressed per, how many happened, and
// where that count came from.
type FunctionalUnit struct {
	Label    string  `json:"label"`
	Quantity float64 `json:"quantity"`
	Source   string  `json:"source,omitempty"`
}

// Assumptions records the inputs a reviewer would want to argue with.
type Assumptions struct {
	Provider      string  `json:"provider,omitempty"`
	PUE           float64 `json:"pue,omitempty"`
	ReservedVCPUs int     `json:"reserved_vcpus,omitempty"`
	HostVCPUs     int     `json:"host_vcpus,omitempty"`
	Hardware      string  `json:"hardware,omitempty"`
	StorageGB     float64 `json:"storage_gb,omitempty"`
	NetworkGB     float64 `json:"network_gb,omitempty"`
	Components    int     `json:"components,omitempty"`
	PeriodHours   float64 `json:"period_hours,omitempty"`
}

// Budget is the verdict of a --budget gate.
type Budget struct {
	Limit float64 `json:"limit"`
	Pass  bool    `json:"pass"`
}

// Report is one SCI disclosure. Its JSON shape is the tool's stable interface:
// `sci compare` reads it, and CI keeps it as an artifact.
type Report struct {
	Tool            string            `json:"tool"`
	Version         string            `json:"version"`
	Target          Target            `json:"target"`
	Measurement     *Measurement      `json:"measurement,omitempty"`
	Components      []ComponentResult `json:"components,omitempty"`
	EnergyKWh       float64           `json:"energy_kwh"`
	EnergySource    string            `json:"energy_source"`
	EnergyBreakdown []EnergyPart      `json:"energy_breakdown_kwh"`
	Intensity       *Intensity        `json:"intensity,omitempty"`
	Operational     float64           `json:"operational_gco2e"`
	Embodied        float64           `json:"embodied_gco2e"`
	EmbodiedDetail  *EmbodiedDetail   `json:"embodied_detail,omitempty"`
	Total           float64           `json:"total_gco2e"`
	FunctionalUnit  FunctionalUnit    `json:"functional_unit"`
	SCI             float64           `json:"sci"`
	SCIUnit         string            `json:"sci_unit"`
	Boundary        []string          `json:"boundary"`
	Assumptions     Assumptions       `json:"assumptions"`
	Notes           []string          `json:"notes"`
	Budget          *Budget           `json:"budget,omitempty"`
}

// SCIReport assembles one disclosure from a measured run.
func SCIReport(target Target, sample Sample, cfg Config, idleWatts float64,
	hasIdle bool, notes []string) (*Report, error) {
	intensity, err := ResolveIntensity(cfg)
	if err != nil {
		return nil, err
	}
	energy, err := EnergyForSample(sample, cfg, idleWatts, hasIdle)
	if err != nil {
		return nil, err
	}
	embodied := EmbodiedGCO2e(cfg, sample.WallS/3600)
	operational := energy.KWh * intensity.Value
	total := operational + embodied.GCO2e
	units := cfg.Units
	if units <= 0 {
		units = 1
	}
	if intensity.Stale {
		notes = append(notes, "the intensity reading is not a current measured hour "+
			"(basis or age); see its source line")
	}
	if intensity.Estimated {
		notes = append(notes, "the intensity reading is modelled rather than measured "+
			"from a live grid feed")
	}
	if intensity.RequestedBasis != "" {
		notes = append(notes, fmt.Sprintf("intensity basis fell back to %s: %s is not "+
			"published for this location", intensity.Basis, intensity.RequestedBasis))
	}

	return &Report{
		Tool:    "sci-disclose",
		Version: Version,
		Target:  target,
		Measurement: &Measurement{
			WallS:       sample.WallS,
			CPUS:        sample.CPUS,
			PeakRSSGB:   sample.PeakRSSGB,
			Utilisation: energy.Utilisation,
			ExitCode:    sample.ExitCode,
			Host: fmt.Sprintf("%s %s, %d vCPU", runtime.GOOS, runtime.GOARCH,
				cfg.TotalVCPUs),
		},
		EnergyKWh:       energy.KWh,
		EnergySource:    energy.Source,
		EnergyBreakdown: energy.Breakdown,
		Intensity:       &intensity,
		Operational:     operational,
		Embodied:        embodied.GCO2e,
		EmbodiedDetail:  &embodied,
		Total:           total,
		FunctionalUnit:  FunctionalUnit{Label: cfg.UnitLabel, Quantity: units, Source: cfg.UnitSource},
		SCI:             total / units,
		SCIUnit:         "gCO2e per " + cfg.UnitLabel,
		Boundary: []string{
			"compute (CPU) for the measured process tree",
			"memory, from peak RSS unless --memory-gb overrides it",
			"storage and network only if --storage-gb / --network-gb are given",
			"embodied hardware, amortised over the run's share of the machine",
			"datacentre overhead via PUE",
			fmt.Sprintf("provider profile: %s, PUE %g", cfg.Provider, cfg.Profile().PUE),
		},
		Assumptions: Assumptions{
			Provider:      cfg.Provider,
			PUE:           cfg.Profile().PUE,
			ReservedVCPUs: cfg.VCPUs,
			HostVCPUs:     cfg.TotalVCPUs,
			Hardware:      cfg.HardwareName,
			StorageGB:     cfg.StorageGB,
			NetworkGB:     cfg.NetworkGB,
		},
		Notes: append(energy.Notes, notes...),
	}, nil
}

// ApplyBudget attaches a budget verdict and reports whether the run is within it.
func ApplyBudget(report *Report, budget float64, set bool) bool {
	if !set {
		return true
	}
	pass := report.SCI <= budget
	report.Budget = &Budget{Limit: budget, Pass: pass}
	return pass
}
