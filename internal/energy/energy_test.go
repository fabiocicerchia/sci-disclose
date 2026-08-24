package energy

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
	"github.com/fabiocicerchia/sci-disclose/internal/config"
	"github.com/fabiocicerchia/sci-disclose/internal/testutil"
)

func TestCPUModelHitsIdleAndFullLoadEndpoints(t *testing.T) {
	profile := coefficients.CPUProfile{MinW: 1, MaxW: 5, PUE: 1}
	testutil.Approx(t, CPUkWh(1, 0, 1, profile), 0.001, 1e-12, "idle")
	testutil.Approx(t, CPUkWh(1, 1, 1, profile), 0.005, 1e-12, "full load")
	testutil.Approx(t, CPUkWh(1, 0.5, 4, profile), 0.012, 1e-12, "half load, four vCPU")
}

func TestUtilisationIsAFractionOfReservedCapacity(t *testing.T) {
	testutil.Approx(t, Utilisation(2, 4, 1), 0.5, 1e-12, "half of one vCPU")
	testutil.Approx(t, Utilisation(8, 4, 4), 0.5, 1e-12, "half of four vCPUs")
	testutil.Approx(t, Utilisation(99, 1, 1), 1.0, 1e-12, "clamped at capacity")
	testutil.Approx(t, Utilisation(1, 0, 1), 0, 1e-12, "no elapsed time")
}

func TestMemoryStorageAndNetworkUseThePublishedCoefficients(t *testing.T) {
	testutil.Approx(t, MemorykWh(10, 1), 10*0.392/1000, 1e-12, "memory")
	testutil.Approx(t, StoragekWh(1024, 1, "ssd"), 1.2/1000, 1e-12, "one TB of SSD for an hour")
	testutil.Approx(t, NetworkkWh(50), 0.05, 1e-12, "50 GB transferred")
}

func TestPUEShowsUpAsDatacentreOverhead(t *testing.T) {
	cfg := testutil.Config(func(c *config.Config) { c.Provider, c.PUE, c.MemoryGB = "aws", 2, 0 })
	energy, err := EnergyForSample(Sample{WallS: 3600, CPUS: 3600}, cfg, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]float64{}
	for _, part := range energy.Breakdown {
		parts[part.Name] = part.KWh
	}
	testutil.Approx(t, parts["datacentre_overhead"], parts["cpu"], 1e-12, "PUE 2 doubles the draw")
	testutil.Approx(t, energy.KWh, 2*parts["cpu"], 1e-12, "total")
}

func TestRAPLJoulesAreConvertedAndPreferredOverTheModel(t *testing.T) {
	cfg := testutil.Config(func(c *config.Config) {
		c.Provider, c.MemoryGB, c.EnergySource = "laptop", 0, "auto"
	})
	sample := Sample{WallS: 10, CPUS: 10, PeakRSSGB: 0.5,
		RAPLJoules: 3.6e6, HasRAPL: true, CoversDRAM: true}
	energy, err := EnergyForSample(sample, cfg, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if energy.Source != "rapl" {
		t.Fatalf("source: %s", energy.Source)
	}
	testutil.Approx(t, energy.KWh, 1.0, 1e-9, "3.6 MJ is exactly 1 kWh")
	for _, part := range energy.Breakdown {
		if part.Name == "memory" {
			t.Error("DRAM is already inside the RAPL figure")
		}
	}
}

func TestRAPLWithoutADRAMDomainAddsModelledMemory(t *testing.T) {
	cfg := testutil.Config(func(c *config.Config) {
		c.Provider, c.MemoryGB, c.EnergySource = "laptop", 8, "auto"
	})
	sample := Sample{WallS: 3600, CPUS: 3600, RAPLJoules: 3.6e6, HasRAPL: true}
	energy, err := EnergyForSample(sample, cfg, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if energy.Source != "rapl" {
		t.Fatalf("source: %s", energy.Source)
	}
	var memory float64
	for _, part := range energy.Breakdown {
		if part.Name == "memory" {
			memory = part.KWh
		}
	}
	testutil.Approx(t, memory, 8*0.392/1000, 1e-12, "modelled memory")
}

func TestIdleBaselineIsSubtractedAndNeverGoesNegative(t *testing.T) {
	cfg := testutil.Config(func(c *config.Config) {
		c.Provider, c.MemoryGB, c.EnergySource = "laptop", 0, "auto"
	})
	sample := Sample{WallS: 10, RAPLJoules: 1000, HasRAPL: true, CoversDRAM: true}
	marginal, err := EnergyForSample(sample, cfg, 40, true)
	if err != nil {
		t.Fatal(err)
	}
	testutil.Approx(t, marginal.Breakdown[0].KWh, 600/3.6e6, 1e-12, "1000 J minus 40 W for 10 s")
	floored, err := EnergyForSample(sample, cfg, 1000, true)
	if err != nil {
		t.Fatal(err)
	}
	testutil.Approx(t, floored.Breakdown[0].KWh, 0, 1e-12, "clamped at zero")
}

func TestAskingForRAPLWithoutCountersIsAnError(t *testing.T) {
	cfg := testutil.Config(func(c *config.Config) { c.EnergySource = "rapl" })
	if _, err := EnergyForSample(Sample{WallS: 1, CPUS: 1}, cfg, 0, false); err == nil {
		t.Fatal("expected an error when RAPL was demanded but unavailable")
	}
}

// fakePowercap builds a sysfs layout like the kernel's: one package, with a
// core and a DRAM child.
func fakePowercap(t *testing.T, energyUJ, maxRange uint64) string {
	t.Helper()
	root := t.TempDir()
	pkg := filepath.Join(root, "intel-rapl:0")
	core := filepath.Join(pkg, "intel-rapl:0:0")
	dram := filepath.Join(pkg, "intel-rapl:0:2")
	for _, dir := range []string{core, dram} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(dir, name, value string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(pkg, "name", "package-0")
	write(pkg, "energy_uj", strconv.FormatUint(energyUJ, 10))
	write(pkg, "max_energy_range_uj", strconv.FormatUint(maxRange, 10))
	write(core, "name", "core")
	write(core, "energy_uj", "500000")
	write(dram, "name", "dram")
	write(dram, "energy_uj", "200000")

	previous := RAPLRoot
	RAPLRoot = root
	t.Cleanup(func() { RAPLRoot = previous })
	return pkg
}

func TestRAPLCountsPackageAndDRAMButNotTheCoreSubdomain(t *testing.T) {
	fakePowercap(t, 1_000_000, 1_000_000_000)
	var names []string
	for _, domain := range raplDomains() {
		names = append(names, domain.name)
	}
	if len(names) != 2 || names[0] != "package-0" || names[1] != "dram" {
		t.Fatalf("domains: %v", names)
	}
	if !RAPLAvailable() {
		t.Error("readable counters should be available")
	}
}

func TestRAPLReaderSumsTheDeltaAcrossDomains(t *testing.T) {
	pkg := fakePowercap(t, 1_000_000, 1_000_000_000)
	reader := NewRAPLReader()
	os.WriteFile(filepath.Join(pkg, "energy_uj"), []byte("3000000\n"), 0o644)
	os.WriteFile(filepath.Join(pkg, "intel-rapl:0:2", "energy_uj"), []byte("400000\n"), 0o644)
	reader.Stop()
	testutil.Approx(t, reader.Joules, (2_000_000+200_000)/1e6, 1e-9, "joules")
	if !reader.CoversDRAM {
		t.Error("the DRAM domain should be counted")
	}
}

func TestRAPLReaderHandlesCounterWraparound(t *testing.T) {
	pkg := fakePowercap(t, 999_000_000, 1_000_000_000)
	reader := NewRAPLReader()
	os.WriteFile(filepath.Join(pkg, "energy_uj"), []byte("500000\n"), 0o644)
	reader.Stop()
	// 1_000_000 - 999_000_000 + 1_000_000_000 = 1_500_000 uJ on the package.
	testutil.Approx(t, reader.Joules, 1.5, 1e-9, "wrapped counter")
}

func TestUnreadableCountersAreSkippedRatherThanFatal(t *testing.T) {
	pkg := fakePowercap(t, 1_000_000, 1_000_000_000)
	if err := os.Chmod(filepath.Join(pkg, "energy_uj"), 0o000); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root, where every counter stays readable")
	}
	reader := NewRAPLReader()
	reader.Stop()
	for _, domain := range reader.domains {
		if domain.name == "package-0" {
			t.Error("an unreadable domain should have been dropped")
		}
	}
}
