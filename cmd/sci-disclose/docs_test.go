package main

import (
	"flag"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/fabiocicerchia/sci-disclose/internal/testutil"
)

// TestDocumentedFlagsExist keeps docs/cli.md's "Flags that change the
// number" table honest: every flag it names must be one the CLI registers.
func TestDocumentedFlagsExist(t *testing.T) {
	data, err := os.ReadFile("../../docs/cli.md")
	if err != nil {
		t.Fatal(err)
	}
	fs := flag.NewFlagSet("docs", flag.ContinueOnError)
	addFlags(fs)

	flagName := regexp.MustCompile("`--([a-z-]+)")
	var seen int
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "| `--") {
			continue
		}
		for _, m := range flagName.FindAllStringSubmatch(line, -1) {
			seen++
			if fs.Lookup(m[1]) == nil {
				t.Errorf("docs/cli.md documents --%s, which the CLI does not register", m[1])
			}
		}
	}
	if seen < 20 {
		t.Fatalf("found only %d flags in docs/cli.md's table; did its format change?", seen)
	}
}

// TestDocumentedIntensityAPIPrecedence holds docs/getting-started.md to its
// word against real HTTP servers: "$SCI_INTENSITY_API sets the API base URL
// when you run your own instance; --intensity-api wins over it."
func TestDocumentedIntensityAPIPrecedence(t *testing.T) {
	body := testutil.ReadingJSON("measured", testutil.Now())
	fromEnv, envHits, _ := testutil.IntensityServer(t, body, http.StatusOK)
	fromFlag, flagHits, _ := testutil.IntensityServer(t, body, http.StatusOK)
	t.Setenv("SCI_INTENSITY_API", fromEnv.URL)

	measure := func(extra ...string) {
		t.Helper()
		args := append([]string{"run", "-region", "eu-west-1", "-format", "json"}, extra...)
		if code, out := runCLI(t, append(append(args, "--"), busyWork...)...); code != 0 {
			t.Fatalf("run %v: exit %d\n%s", extra, code, out)
		}
	}

	measure()
	if envHits.Load() != 1 || flagHits.Load() != 0 {
		t.Fatalf("$SCI_INTENSITY_API alone: env server hit %d times, flag server %d; want 1 and 0",
			envHits.Load(), flagHits.Load())
	}
	measure("-intensity-api", fromFlag.URL)
	if envHits.Load() != 1 || flagHits.Load() != 1 {
		t.Fatalf("--intensity-api should win: env server hit %d times, flag server %d; want 1 and 1",
			envHits.Load(), flagHits.Load())
	}
}
