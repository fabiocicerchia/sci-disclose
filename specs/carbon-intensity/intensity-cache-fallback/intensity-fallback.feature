@domain:carbon-intensity @area:intensity-cache-fallback @from:grid/intensity
Feature: Keeping a measurement going when live intensity is unavailable
  Ensures an operator always gets an intensity for a known location, with the report saying which substitute was used, and is refused rather than given a guess for an unknown one.

  @id:carbon-intensity.intensity-cache-fallback.001 @rule:carbon-intensity.fallback.bundled-average-when-unreachable @verified @source:internal/grid/intensity.go#resolveIntensity @test:internal/grid/intensity_test.go#TestAnUnreachableAPIFallsBackToTheBundledTable @test:internal/grid/intensity_test.go#TestOfflineNeverCallsOut
  Scenario Outline: When the intensity API cannot be used, the bundled yearly average for the zone is used and the report says why
    Given the Carbon Intensity API <api_state>
    And no earlier reading for Ireland is available
    When the operator measures a workload with --region eu-west-1 <offline_flag>
    Then the report uses the bundled yearly average for zone IE
    And the intensity source reads "bundled yearly average, zone IE<suffix>"
    And the Carbon Intensity API was asked <requests>

    Examples:
      | api_state                | offline_flag | suffix                              | requests |
      | answers 429 rate limited |              | (the intensity API was unreachable) | once     |
      | is reachable             | --offline    |                                     | never    |

  @id:carbon-intensity.intensity-cache-fallback.002 @rule:carbon-intensity.fallback.recent-reading-reused @verified @source:internal/grid/intensity.go#fetchReading @test:internal/grid/intensity_test.go#TestReadingsAreCachedSoTheRateLimitIsNotHammered @test:internal/grid/intensity_test.go#TestAStaleCachedReadingIsRefetched @test:internal/grid/intensity_test.go#TestCacheFreshnessFollowsTheReadingNotTheFile
  Scenario: A reading that still describes the current hour is reused instead of asking the API again
    Given the operator measured a workload with --country IE moments ago, from a freshly measured reading
    When the operator measures a workload with --country IE again
    Then the report uses the same intensity
    And the intensity source says the reading was "cached" and how long ago
    And the Carbon Intensity API was asked only once in total

  @id:carbon-intensity.intensity-cache-fallback.003 @rule:carbon-intensity.fallback.last-reading-before-bundled-table @verified @source:internal/grid/intensity.go#fetchReading @test:internal/grid/intensity_test.go#TestALiveFailureFallsBackToTheCacheBeforeTheBundledTable
  Scenario: When the API fails, the last reading obtained is preferred over the bundled yearly average
    Given the operator earlier obtained a reading of 339 gCO2e/kWh for Ireland
    And that reading no longer counts as current
    And the Carbon Intensity API is now unreachable
    When the operator measures a workload with --country IE
    Then the report uses an intensity of 339 gCO2e/kWh
    And the intensity source says it is a stale cache, how old it is, and that the live fetch failed
    And the measurement does not fail

  @id:carbon-intensity.intensity-cache-fallback.004 @rule:carbon-intensity.fallback.unknown-location-refused @verified @source:internal/grid/intensity.go#resolveIntensity @test:internal/grid/intensity_test.go#TestUnknownRegionIsRefusedRatherThanGuessed @test:internal/grid/intensity_test.go#TestNoLocationFallsBackToTheWorldAverage @test:internal/grid/intensity_test.go#TestEveryCloudRegionResolvesOfflineToo
  Scenario Outline: An unknown location is refused rather than guessed, and no location means the world average
    Given no intensity is supplied with --intensity
    And the operator is working offline
    When the operator measures a workload <location>
    Then <outcome>

    Examples:
      | location                              | outcome                                                                                                               |
      | with --region atlantis-1              | the run is refused with "unknown region or zone \"atlantis-1\"; pass --intensity, or see `sci-disclose coefficients`" |
      | without --region, --country or --zone | the report uses 481 gCO2e/kWh with source "world average (no --region, --country or --zone given)"                    |
