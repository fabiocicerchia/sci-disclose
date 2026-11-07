@domain:disclosure @area:coefficient-sources @from:cmd/sci-disclose
Feature: Publishing every coefficient with its source
  Auditors check every constant behind a disclosure, with its citation, without reading the code.

  @id:disclosure.coefficient-sources.001 @rule:coefficients.listing.every-constant-with-source @verified @source:cmd/sci-disclose/disclosures.go#cmdCoefficients @test:cmd/sci-disclose/main_test.go#TestCoefficientsArePrintedWithTheirSources
  Scenario: The coefficients listing shows each constant and the source it came from
    When the auditor runs sci-disclose coefficients
    Then power and PUE per provider, memory, network and storage figures, and embodied emissions per device are listed
    And the default intensity API and the bundled yearly grid averages are listed
    And a Sources section cites each source, including Cloud Carbon Footprint
    And the last line reports the host's OS, architecture, vCPUs and whether RAPL counters are readable

  @id:disclosure.coefficient-sources.002 @rule:coefficients.listing.machine-readable-form @verified @source:cmd/sci-disclose/disclosures.go#cmdCoefficients @test:cmd/sci-disclose/main_test.go#TestCoefficientsArePrintedWithTheirSources
  Scenario: The coefficients are also available as JSON for other tools
    When the auditor runs sci-disclose coefficients -format json
    Then the output is a JSON object with keys cpu_profiles, memory_w_per_gb, network_kwh_per_gb, storage_w_per_tb, hardware, grid_zones_gco2e_per_kwh, region_zone, region_country, intensity_api, intensity_bases and sources
