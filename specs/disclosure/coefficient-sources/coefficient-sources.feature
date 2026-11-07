@domain:disclosure @area:coefficient-sources @from:coefficients/coefficients
Feature: Publishing the coefficients behind every estimate
  Operators and auditors can see every published figure a disclosure is built from, with its citation, so the reported SCI is defensible.

  Background:
    Given the operator has sci-disclose installed

  @id:disclosure.coefficient-sources.003 @rule:disclosure.coefficients.every-figure-cited @verified @source:internal/coefficients/coefficients.go#CoefficientSources @test:cmd/sci-disclose/main_test.go#TestCoefficientsArePrintedWithTheirSources
  Scenario: Every published figure is listed with the source it came from
    When the operator runs "sci-disclose coefficients"
    Then the command exits with status 0
    And a "Sources:" section lists each figure followed by its citation
    And "CPU min/max watts per vCPU" is cited to the Cloud Carbon Footprint methodology
    And "Embodied server 1200 kgCO2e / 4y" is cited to mid-range 2U server LCAs (Dell PAIA, Boavizta)
    And "Grid intensity table" is cited to Electricity Maps yearly zone averages with a world average of 481 gCO2e/kWh (Ember)

  @id:disclosure.coefficient-sources.004 @rule:disclosure.coefficients.provider-power-profiles @inferred @source:internal/coefficients/coefficients.go#CPUProfiles
  Scenario Outline: Each provider's assumed power draw and PUE is published
    When the operator runs "sci-disclose coefficients"
    Then the provider "<provider>" is listed with <idle> W idle and <full> W full load per vCPU
    And its PUE is shown as <pue>

    Examples:
      | provider | idle | full | pue   |
      | aws      | 0.74 | 3.50 | 1.135 |
      | gcp      | 0.71 | 4.26 | 1.1   |
      | azure    | 0.78 | 3.76 | 1.185 |
      | onprem   | 0.74 | 3.50 | 1.58  |
      | laptop   | 0.30 | 2.00 | 1     |

  @id:disclosure.coefficient-sources.005 @rule:disclosure.coefficients.embodied-hardware-figures @inferred @source:internal/coefficients/coefficients.go#Hardware
  Scenario Outline: Each hardware class's embodied emissions and lifespan are published
    When the operator runs "sci-disclose coefficients"
    Then under "Embodied emissions:" the hardware "<hardware>" is listed as <kg> kgCO2e over <years> years

    Examples:
      | hardware | kg   | years |
      | server   | 1200 | 4     |
      | laptop   | 250  | 4     |
      | phone    | 60   | 3     |

  @id:disclosure.coefficient-sources.006 @rule:disclosure.coefficients.bundled-grid-intensity @inferred @source:internal/coefficients/coefficients.go#GridZones
  Scenario Outline: The bundled yearly grid intensities are published, cleanest zone first
    When the operator runs "sci-disclose coefficients"
    Then under "Bundled grid intensity" the zone "<zone>" is shown at <intensity> gCO2e/kWh
    And zones are listed from lowest to highest intensity
    And the listing states that 34 cloud regions map onto those zones and countries

    Examples:
      | zone  | intensity |
      | SE    | 25        |
      | FR    | 56        |
      | GB    | 230       |
      | DE    | 380       |
      | WORLD | 481       |
      | ZA    | 700       |

  @id:disclosure.coefficient-sources.007 @rule:disclosure.coefficients.live-intensity-source-stated @verified @source:internal/coefficients/coefficients.go#DefaultIntensityAPI @test:cmd/sci-disclose/main_test.go#TestCoefficientsArePrintedWithTheirSources
  Scenario: The listing names the live intensity service and the figure reported by default
    When the operator runs "sci-disclose coefficients"
    Then the listing names "https://ci-api.fabiocicerchia.it" as the source of last-hour readings
    And it states the default figure is "consumption_lifecycle"
    And it states the bundled yearly averages are used when the service is unreachable or with -offline

  @id:disclosure.coefficient-sources.008 @rule:disclosure.coefficients.machine-readable-table @verified @source:internal/coefficients/coefficients.go#CPUProfile @test:cmd/sci-disclose/main_test.go#TestCoefficientsArePrintedWithTheirSources
  Scenario: The full coefficient table is available as JSON for other tools
    When the operator runs "sci-disclose coefficients -format json"
    Then the command exits with status 0
    And the output is a JSON object with keys cpu_profiles, memory_w_per_gb, network_kwh_per_gb, storage_w_per_tb, hardware, grid_zones_gco2e_per_kwh, region_zone, region_country, intensity_api, intensity_bases and sources
    And "sources" maps each figure name to its citation
    And each cpu_profiles entry carries min_w, max_w and pue, and each hardware entry carries embodied_kg and lifespan_years
