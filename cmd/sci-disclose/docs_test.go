package main

import (
	"flag"
	"os"
	"regexp"
	"strings"
	"testing"
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
