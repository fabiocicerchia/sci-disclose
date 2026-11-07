@domain:carbon-intensity @area:intensity-lookup @from:grid/intensity
Feature: Choosing the grid carbon intensity for a run
  Lets an operator disclosing an SCI score know exactly which grid intensity (I) was used, where it came from and how fresh it is, so the figure can be defended or challenged.

  @id:carbon-intensity.intensity-lookup.001 @rule:carbon-intensity.lookup.explicit-figure-wins @verified @source:internal/grid/intensity.go#resolveIntensity @test:internal/grid/intensity_test.go#TestExplicitIntensityBeatsEveryLookup
  Scenario: An intensity the operator supplies overrides every lookup
    Given the operator has set the location with --country DE
    When the operator measures a workload with --intensity 42
    Then the report uses an intensity of 42 gCO2e/kWh
    And the intensity source reads "--intensity (user supplied)"
    And no request is made to the Carbon Intensity API

  @id:carbon-intensity.intensity-lookup.002 @rule:carbon-intensity.lookup.live-reading-with-provenance @verified @source:internal/grid/intensity.go#resolveIntensity @test:internal/grid/intensity_test.go#TestLastHourReadingIsUsedWithItsProvenance
  Scenario: A cloud region is resolved to its country's last-hour reading, quoted with its provenance and credit
    Given the Carbon Intensity API publishes a measured last-hour reading for Ireland of 339 gCO2e/kWh consumption_lifecycle, window 2026-08-21T14:30:00Z to 2026-08-21T15:00:00Z, from ENTSO-E
    When the operator measures a workload with --region eu-west-1
    Then the report uses an intensity of 339 gCO2e/kWh on the consumption_lifecycle basis
    And the intensity is marked measured and not stale
    And the intensity window reads "2026-08-21T14:30:00Z to 2026-08-21T15:00:00Z"
    And the intensity source names Ireland, the measured basis, the window and ENTSO-E
    And the credit line names the Carbon Intensity API, its licence AGPL-3.0-or-later and the grid data source
