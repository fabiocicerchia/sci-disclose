package main

// Tests that span more than one internal package. They live here because
// package main is the only place allowed to import all of them at once:
// sci imports grid, and disclosure imports sci, so neither of those packages
// can test the whole chain from inside itself.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fabiocicerchia/sci-disclose/internal/config"
	"github.com/fabiocicerchia/sci-disclose/internal/energy"
	"github.com/fabiocicerchia/sci-disclose/internal/report"
	"github.com/fabiocicerchia/sci-disclose/internal/sci"
	"github.com/fabiocicerchia/sci-disclose/internal/testutil"
)

func TestAFallenBackBasisIsNotedInTheReport(t *testing.T) {
	server, _, _ := testutil.IntensityServer(t, testutil.ZoneReadingJSON(testutil.Now()), http.StatusOK)
	cfg := testutil.Config(func(c *config.Config) {
		c.Intensity, c.Offline = 0, false
		c.IntensityAPI, c.Zone = server.URL, "IT/SICI"
	})
	disclosure, err := sci.SCIReport(sci.Target{Kind: "test"}, energy.Sample{WallS: 1, CPUS: 1}, cfg, 0, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	var noted bool
	for _, note := range disclosure.Notes {
		if strings.Contains(note, "basis fell back to lifecycle") {
			noted = true
		}
	}
	if !noted {
		t.Errorf("the basis substitution should reach the disclosure: %v", disclosure.Notes)
	}
	if !strings.Contains(report.RenderText(disclosure), "AGPL-3.0-or-later") {
		t.Error("the text disclosure should carry the attribution")
	}
}
