package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The other tests drive Run in-process. These build the real binary and run
// it as a user would, so main, os.Exit and the exit-code contract are covered.

func buildBinary(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds the binary")
	}
	bin := filepath.Join(t.TempDir(), "sci-disclose")
	if out, err := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func runBinary(t *testing.T, bin string, args ...string) (int, string) {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), bin, args...).CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), string(out)
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0, string(out)
}

func TestBinaryMeasuresThenGatesARegression(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	light := filepath.Join(dir, "light.json")
	heavy := filepath.Join(dir, "heavy.json")

	measure := func(path string, units string) {
		args := append([]string{"run", "-offline", "-region", "eu-north-1", "-units", units,
			"-format", "json", "-o", path, "--"}, busyWork...)
		if code, out := runBinary(t, bin, args...); code != 0 {
			t.Fatalf("run: exit %d\n%s", code, out)
		}
	}
	// The same workload over fewer units is a higher rate: a regression.
	measure(light, "100")
	measure(heavy, "1")
	if readReport(t, heavy).SCI <= readReport(t, light).SCI {
		t.Fatal("fewer functional units should give a higher SCI")
	}

	if code, out := runBinary(t, bin, "compare", "-fail-on-regression", light, heavy); code != 1 ||
		!strings.Contains(out, "delta") {
		t.Fatalf("a regression should fail the gate: exit %d\n%s", code, out)
	}
	if code, out := runBinary(t, bin, "compare", "-fail-on-regression", heavy, light); code != 0 {
		t.Fatalf("an improvement should pass: exit %d\n%s", code, out)
	}
}

func TestBinaryExitCodes(t *testing.T) {
	bin := buildBinary(t)
	for _, c := range []struct {
		want int
		args []string
	}{
		{0, []string{"--version"}},
		{1, append([]string{"run", "-offline", "-budget", "0", "--"}, busyWork...)},
		{2, []string{"nonsense"}},
		{3, []string{"run", "-offline", "--", "sh", "-c", "exit 3"}},
	} {
		if code, out := runBinary(t, bin, c.args...); code != c.want {
			t.Errorf("%v: exit %d, want %d\n%s", c.args, code, c.want, out)
		}
	}
}
