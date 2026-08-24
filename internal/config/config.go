package config

import (
	"fmt"
	"runtime"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
)

// Config is the software boundary, the grid, the hardware and the functional
// unit: everything the SCI equation needs beyond the measurement itself.

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
		IntensityBasis: coefficients.IntensityBases[0],
		IntensityAPI:   coefficients.DefaultIntensityAPI,
		HardwareName:   "server",
		EnergySource:   "auto",
		Units:          1,
		UnitLabel:      "run",
	}
}

// Profile is the provider power profile with any PUE override applied.
func (c Config) Profile() coefficients.CPUProfile {
	profile := coefficients.CPUProfiles[c.Provider]
	if c.PUE > 0 {
		profile.PUE = c.PUE
	}
	return profile
}

// DeviceSpec is the hardware preset with any embodied overrides applied.
func (c Config) DeviceSpec() coefficients.Device {
	device := coefficients.Hardware[c.HardwareName]
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
	if _, ok := coefficients.CPUProfiles[c.Provider]; !ok {
		return fmt.Errorf("unknown provider %q (one of: %v)", c.Provider, coefficients.ProviderNames)
	}
	if _, ok := coefficients.Hardware[c.HardwareName]; !ok {
		return fmt.Errorf("unknown hardware %q (one of: %v)", c.HardwareName, coefficients.HardwareNames)
	}
	if _, ok := coefficients.StorageWPerTB[c.StorageMedium]; !ok {
		return fmt.Errorf("unknown storage medium %q (ssd or hdd)", c.StorageMedium)
	}
	switch c.EnergySource {
	case "auto", "rapl", "model":
	default:
		return fmt.Errorf("unknown energy source %q (auto, rapl or model)", c.EnergySource)
	}
	return nil
}
