package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
	"github.com/fabiocicerchia/sci-disclose/internal/sci"
	"github.com/fabiocicerchia/sci-disclose/internal/testutil"
)

// busyWork is a short, portable CPU burn: no interpreter needed beyond sh.
var busyWork = []string{"sh", "-c", "i=0; while [ $i -lt 20000 ]; do i=$((i+1)); done"}

func runCLI(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	code := Run(args, &out)
	return code, out.String()
}

func readReport(t *testing.T, path string) *sci.Report {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report sci.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	return &report
}

func TestRunMeasuresACommandAndEmitsJSON(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")
	args := append([]string{"run", "-offline", "-region", "eu-north-1", "-vcpus", "1",
		"-format", "json", "-o", out, "--"}, busyWork...)
	code, stdout := runCLI(t, args...)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	report := readReport(t, out)
	if report.SCI <= 0 || report.Intensity.Value != coefficients.GridZones["SE"] {
		t.Fatalf("%+v", report)
	}
	if report.Measurement.ExitCode != 0 || report.Measurement.CPUS <= 0 {
		t.Fatalf("measurement: %+v", report.Measurement)
	}
	if report.Tool != "sci-disclose" || report.Version != coefficients.Version {
		t.Errorf("provenance: %s %s", report.Tool, report.Version)
	}
}

func TestRunReportsAFailedWorkloadAsExitCodeThree(t *testing.T) {
	code, stdout := runCLI(t, "run", "-offline", "--", "sh", "-c", "exit 3")
	if code != 3 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "not a representative measurement") {
		t.Errorf("a failed workload should say so:\n%s", stdout)
	}
}

func TestBudgetGateFailsTheCommand(t *testing.T) {
	args := append([]string{"run", "-offline", "-budget", "0", "--"}, busyWork...)
	code, stdout := runCLI(t, args...)
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "OVER") {
		t.Errorf("the verdict should be visible:\n%s", stdout)
	}
	args = append([]string{"run", "-offline", "-budget", "1000000", "--"}, busyWork...)
	if code, _ := runCLI(t, args...); code != 0 {
		t.Errorf("a generous budget should pass, got exit %d", code)
	}
}

func TestUsageErrorsExitTwo(t *testing.T) {
	for _, args := range [][]string{
		{},                             // no command at all
		{"run"},                        // nothing to run
		{"run", "-provider", "moon"},   // unknown provider
		{"file"},                       // no path
		{"nonsense"},                   // unknown command
		{"compare", "only-one.json"},   // wrong arity
		{"estimate", "-f", "gone.yml"}, // missing manifest
	} {
		if code, out := runCLI(t, args...); code != 2 {
			t.Errorf("%v: exit %d\n%s", args, code, out)
		}
	}
}

func TestFileTargetRunsTheScript(t *testing.T) {
	script := testutil.WriteFile(t, t.TempDir(), "work.sh", "i=0\nwhile [ $i -lt 5000 ]; do i=$((i+1)); done\n")
	code, stdout := runCLI(t, "file", "-offline", "-format", "json", script)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var report sci.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.Target.Kind != "file" || report.FunctionalUnit.Label != "execution" {
		t.Fatalf("%+v", report.Target)
	}
}

func TestFileTargetNeedsAKnownInterpreter(t *testing.T) {
	odd := testutil.WriteFile(t, t.TempDir(), "thing.zzz", "noop\n")
	if code, out := runCLI(t, "file", odd); code != 2 ||
		!strings.Contains(out, "no interpreter known") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestFuncTargetDividesByTheIterationCount(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not installed")
	}
	dir := t.TempDir()
	testutil.WriteFile(t, dir, "workload_under_test.py", "def go():\n    return sum(range(1000))\n")
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(previous) })

	code, stdout := runCLI(t, "func", "-offline", "-n", "50", "-warmup", "1",
		"-format", "json", "workload_under_test:go")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var report sci.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.FunctionalUnit.Quantity != 50 || report.FunctionalUnit.Label != "call" {
		t.Fatalf("R: %+v", report.FunctionalUnit)
	}
	testutil.Approx(t, report.SCI, report.Total/50, 1e-12, "per call")
	if report.Measurement.CPUS <= 0 || report.Measurement.WallS <= 0 {
		t.Errorf("the harness reported nothing: %+v", report.Measurement)
	}
}

func TestFuncTargetRejectsABadReference(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not installed")
	}
	for _, target := range []string{"no_such_module:go", "os.path"} {
		if code, _ := runCLI(t, "func", "-offline", target); code != 2 {
			t.Errorf("%s: exit %d", target, code)
		}
	}
}

func TestRepoTargetRunsTheDetectedWorkload(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, dir, "Makefile", "test:\n\t@true\n")
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not installed")
	}
	code, stdout := runCLI(t, "repo", "-offline", "-format", "json", dir)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var report sci.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.Target.Kind != "repo" || !strings.Contains(report.Target.DetectedFrom, "test") {
		t.Fatalf("%+v", report.Target)
	}
	if report.FunctionalUnit.Label != "make test run" {
		t.Errorf("R should name the workload: %q", report.FunctionalUnit.Label)
	}
}

func TestRepoTargetSaysSoWhenThereIsNoWorkload(t *testing.T) {
	code, out := runCLI(t, "repo", t.TempDir())
	if code != 2 || !strings.Contains(out, "no workload found") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestInitThenEstimateRoundTrips(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "sci.yaml")
	testutil.WriteFile(t, dir, "deploy.yaml", `kind: Deployment
metadata:
  name: api
spec:
  replicas: 2
  template:
    spec:
      containers:
        - name: api
          resources:
            requests:
              cpu: "1"
`)
	if code, out := runCLI(t, "init", "-o", manifest, dir); code != 0 {
		t.Fatalf("init: exit %d: %s", code, out)
	}
	text, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), "name: api") {
		t.Fatalf("the discovered workload is missing:\n%s", text)
	}

	report := filepath.Join(dir, "report.json")
	if code, out := runCLI(t, "estimate", "-offline", "-f", manifest,
		"-format", "json", "-o", report); code != 0 {
		t.Fatalf("estimate: exit %d: %s", code, out)
	}
	if readReport(t, report).SCI <= 0 {
		t.Error("the scaffolded manifest should estimate to something")
	}
}

func TestInitRefusesToClobber(t *testing.T) {
	dir := t.TempDir()
	manifest := testutil.WriteFile(t, dir, "sci.yaml", "keep me\n")
	if code, out := runCLI(t, "init", "-o", manifest, dir); code != 2 ||
		!strings.Contains(out, "already exists") {
		t.Fatalf("exit %d: %s", code, out)
	}
	data, _ := os.ReadFile(manifest)
	if string(data) != "keep me\n" {
		t.Error("the existing manifest was overwritten")
	}
	if code, _ := runCLI(t, "init", "-force", "-o", manifest, dir); code != 0 {
		t.Error("-force should overwrite")
	}
}

func TestCompareCommandGatesOnRegression(t *testing.T) {
	dir := t.TempDir()
	before := testutil.WriteFile(t, dir, "a.json", `{"sci": 1.0, "sci_unit": "gCO2e per run"}`)
	after := testutil.WriteFile(t, dir, "b.json", `{"sci": 2.0, "sci_unit": "gCO2e per run"}`)
	if code, out := runCLI(t, "compare", before, after); code != 0 ||
		!strings.Contains(out, "delta") {
		t.Fatalf("exit %d: %s", code, out)
	}
	if code, _ := runCLI(t, "compare", "-fail-on-regression", before, after); code != 1 {
		t.Error("a regression should fail the gate")
	}
	if code, _ := runCLI(t, "compare", "-fail-on-regression", after, before); code != 0 {
		t.Error("an improvement is not a regression")
	}
}

func TestCoefficientsArePrintedWithTheirSources(t *testing.T) {
	code, out := runCLI(t, "coefficients")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"Cloud Carbon Footprint", "Embodied", coefficients.DefaultIntensityAPI} {
		if !strings.Contains(out, want) {
			t.Errorf("coefficients output is missing %q", want)
		}
	}
	code, out = runCLI(t, "coefficients", "-format", "json")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed["cpu_profiles"]; !ok {
		t.Error("the JSON form should carry the profiles")
	}
}

func TestHelpAndVersion(t *testing.T) {
	if code, out := runCLI(t, "--help"); code != 0 || !strings.Contains(out, "SCI = ((E x I) + M)") {
		t.Errorf("help: exit %d\n%s", code, out)
	}
	if code, out := runCLI(t, "--version"); code != 0 || !strings.Contains(out, coefficients.Version) {
		t.Errorf("version: exit %d\n%s", code, out)
	}
}
