package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fabiocicerchia/sci-disclose/internal/energy"
)

// Function targets need a meter inside the process being measured: interpreter
// startup and import time must not land in the bracket. The harness below runs
// in the target language, times only the call loop, reads the RAPL counters
// itself, and reports back as JSON. Adding a language means adding a harness,
// not changing anything above.

const pythonHarness = `import json, os, resource, sys, time

def rapl_domains(root):
    domains = []
    try:
        names = sorted(n for n in os.listdir(root) if n.startswith("intel-rapl:"))
    except OSError:
        return domains
    for name in names:
        pkg = os.path.join(root, name)
        if not os.path.exists(os.path.join(pkg, "energy_uj")):
            continue
        domains.append((pkg, name))
        for sub in sorted(os.listdir(pkg)):
            if not sub.startswith("intel-rapl:"):
                continue
            path = os.path.join(pkg, sub)
            try:
                label = open(os.path.join(path, "name")).read().strip()
            except OSError:
                continue
            if label == "dram" and os.path.exists(os.path.join(path, "energy_uj")):
                domains.append((path, "dram"))
    return domains

def read(path):
    with open(os.path.join(path, "energy_uj")) as handle:
        return int(handle.read().strip())

def limit(path):
    try:
        with open(os.path.join(path, "max_energy_range_uj")) as handle:
            return int(handle.read().strip())
    except OSError:
        return 0

target, iterations, warmup, out = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), sys.argv[4]
root = os.environ.get("SCI_RAPL_ROOT", "/sys/class/powercap")

if ":" not in target:
    sys.exit("sci: function target must be module:attribute, e.g. mypkg.bench:main")
module_name, attribute = target.split(":", 1)
sys.path.insert(0, os.getcwd())
import importlib
try:
    obj = importlib.import_module(module_name)
except ImportError as exc:
    sys.exit("sci: cannot import %r: %s" % (module_name, exc))
for part in attribute.split("."):
    try:
        obj = getattr(obj, part)
    except AttributeError:
        sys.exit("sci: %s has no attribute %r" % (module_name, attribute))
if not callable(obj):
    sys.exit("sci: %s is not callable" % target)

for _ in range(warmup):
    obj()

domains = [(p, n) for p, n in rapl_domains(root) if os.access(os.path.join(p, "energy_uj"), os.R_OK)]
start = {}
for path, _ in domains:
    try:
        start[path] = read(path)
    except OSError:
        pass

before = resource.getrusage(resource.RUSAGE_SELF)
began = time.monotonic()
for _ in range(iterations):
    obj()
wall = time.monotonic() - began
after = resource.getrusage(resource.RUSAGE_SELF)

total_uj, covers_dram, measured = 0, False, False
for path, name in domains:
    if path not in start:
        continue
    try:
        end = read(path)
    except OSError:
        continue
    begin = start[path]
    if end < begin:
        end += limit(path)
    total_uj += end - begin
    measured = True
    covers_dram = covers_dram or name == "dram"

cpu = (after.ru_utime - before.ru_utime) + (after.ru_stime - before.ru_stime)
divisor = 1024 ** 3 if sys.platform == "darwin" else 1024 ** 2
with open(out, "w") as handle:
    json.dump({"wall_s": wall, "cpu_s": cpu,
               "peak_rss_gb": after.ru_maxrss / divisor,
               "rapl_joules": total_uj / 1e6, "has_rapl": measured,
               "covers_dram": covers_dram, "iterations": iterations}, handle)
`

type harnessResult struct {
	WallS      float64 `json:"wall_s"`
	CPUS       float64 `json:"cpu_s"`
	PeakRSSGB  float64 `json:"peak_rss_gb"`
	RAPLJoules float64 `json:"rapl_joules"`
	HasRAPL    bool    `json:"has_rapl"`
	CoversDRAM bool    `json:"covers_dram"`
	Iterations float64 `json:"iterations"`
}

// MeasureFunction runs a function target through the language harness and
// returns the sample the harness measured from inside the process.
func MeasureFunction(interpreter, target string, iterations, warmup int,
	useRAPL bool) (energy.Sample, error) {
	dir, err := os.MkdirTemp("", "sci-harness")
	if err != nil {
		return energy.Sample{}, err
	}
	defer os.RemoveAll(dir)

	script := filepath.Join(dir, "harness.py")
	result := filepath.Join(dir, "result.json")
	if err := os.WriteFile(script, []byte(pythonHarness), 0o600); err != nil {
		return energy.Sample{}, err
	}

	cmd := exec.Command(interpreter, script, target,
		fmt.Sprint(iterations), fmt.Sprint(warmup), result)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = os.Environ()
	if useRAPL {
		cmd.Env = append(cmd.Env, "SCI_RAPL_ROOT="+energy.RAPLRoot)
	} else {
		cmd.Env = append(cmd.Env, "SCI_RAPL_ROOT=/nonexistent")
	}
	if err := cmd.Run(); err != nil {
		return energy.Sample{}, fmt.Errorf("the %s harness failed: %w", interpreter, err)
	}

	data, err := os.ReadFile(result)
	if err != nil {
		return energy.Sample{}, fmt.Errorf("the harness produced no measurement: %w", err)
	}
	var measured harnessResult
	if err := json.Unmarshal(data, &measured); err != nil {
		return energy.Sample{}, err
	}
	return energy.Sample{
		WallS:      measured.WallS,
		CPUS:       measured.CPUS,
		PeakRSSGB:  measured.PeakRSSGB,
		RAPLJoules: measured.RAPLJoules,
		HasRAPL:    measured.HasRAPL,
		CoversDRAM: measured.CoversDRAM,
		Iterations: measured.Iterations,
	}, nil
}
