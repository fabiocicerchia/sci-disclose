@domain:declared-deployments @area:manifest-estimates @from:manifest/manifest
Feature: Estimating the SCI of a declared deployment
  Lets an operator disclose the carbon intensity of a deployed system they are not running locally, by declaring its components in sci.yaml and getting one SCI report back.

  Background:
    Given a manifest file sci.yaml describing a deployment

  @id:declared-deployments.manifest-estimates.001 @rule:manifests.estimate.sci-is-total-per-declared-unit @verified @source:internal/manifest/manifest.go#EstimateManifest @test:internal/manifest/manifest_test.go#TestManifestTotalsAreTheSumOfTheComponents @test:internal/manifest/manifest_test.go#TestDocumentedManifestIsValidInputForEstimate
  Scenario Outline: A declared deployment's SCI is the sum of its components per declared functional unit
    Given the manifest declares the functional unit "<label>" with quantity <quantity>
    And its components together emit <total> gCO2e
    When the operator estimates the manifest
    Then the report states an SCI of <sci>
    And the SCI unit reads "<unit>"
    And the energy source reads "declared (manifest)"
    And the notes include "declared, not measured: the SCI is only as good as the utilisation and functional unit in the manifest"

    Examples:
      | label   | quantity | total | sci | unit              |
      | request | 1000     | 500   | 0.5 | gCO2e per request |
      | (none)  | (none)   | 500   | 500 | gCO2e per run     |

  @id:declared-deployments.manifest-estimates.002 @rule:manifests.estimate.compute-uses-measured-power-model @verified @source:internal/manifest/manifest.go#EstimateComponent @test:internal/manifest/manifest_test.go#TestManifestComputeMatchesTheModelUsedForMeasuredRuns @questionable
  Scenario Outline: A compute component is estimated with the same power model as measured runs, plus datacentre overhead
    Given a compute component with <vcpus> vCPUs, <replicas> replicas, utilisation <util> and <memory> GB of memory running <hours> hours on provider <provider>
    When the operator estimates the manifest
    Then its CPU energy is vCPUs × replicas × (idle watts + utilisation × (max watts − idle watts)) × hours / 1000 kWh for that provider
    And any declared memory adds memory energy for memory × replicas over the same hours
    And a datacentre overhead line adds the provider's PUE share on top

    Examples:
      | vcpus     | replicas  | util        | memory | hours     | provider       |
      | 2         | 2         | 1.0         | 4      | 10        | aws            |
      | (none: 1) | (none: 1) | (none: 0.5) | (none) | (none: 1) | (none: onprem) |

  @id:declared-deployments.manifest-estimates.003 @rule:manifests.estimate.devices-carry-no-datacentre-overhead @verified @source:internal/manifest/manifest.go#EstimateComponent @test:internal/manifest/manifest_test.go#TestEndUserDevicesCarryNoDatacentreOverhead
  Scenario: End-user devices are charged their own power draw with no datacentre overhead
    Given a device component of 5 watts with 10 replicas over 2 hours
    When the operator estimates the manifest
    Then the component's energy is 0.1 kWh
    And its energy breakdown has no datacentre overhead line

  @id:declared-deployments.manifest-estimates.004 @rule:manifests.estimate.component-grid-overrides-default @verified @source:internal/manifest/manifest.go#componentConfig @test:internal/manifest/manifest_test.go#TestComponentOverridesBeatTheDefaults
  Scenario: A component's own grid setting overrides the manifest default, and a mixed-grid deployment states no single intensity
    Given the manifest defaults to a grid intensity of 100 gCO2e/kWh
    And component "elsewhere" declares an intensity of 900 gCO2e/kWh
    And component "here" declares none
    When the operator estimates the manifest
    Then "elsewhere" is charged at 900 gCO2e/kWh
    And "here" is charged at 100 gCO2e/kWh
    And the report states no headline grid intensity

  @id:declared-deployments.manifest-estimates.005 @rule:manifests.estimate.embodied-share-of-declared-hardware @verified @source:internal/manifest/manifest.go#componentEmbodied @test:internal/manifest/manifest_test.go#TestManifestEmbodiedUsesVCPUsOverTotalVCPUs
  Scenario: A component's embodied carbon is its time and resource share of the declared hardware
    Given a compute component with 2 vCPUs and 2 replicas running 10 hours
    And its hardware is declared as 1000 kg embodied, 4-year lifespan, 64 vCPUs in total
    When the operator estimates the manifest
    Then its embodied emissions are 1,000,000 g × 10 / (4 years in hours) × 2/64 × 2 replicas
    And a component declaring no hardware carries no embodied emissions

  @id:declared-deployments.manifest-estimates.006 @rule:manifests.estimate.refuses-invalid-manifest @verified @source:internal/manifest/manifest.go#EstimateManifest @test:internal/manifest/manifest_test.go#TestManifestErrorsAreExplicit
  Scenario Outline: A manifest that cannot be estimated is refused with a stated reason
    Given the manifest <problem>
    When the operator estimates the manifest
    Then no report is produced
    And the operator sees an error mentioning <reason>

    Examples:
      | problem                              | reason                                     |
      | does not exist                       | the missing file                           |
      | is not valid YAML or JSON            | is not valid YAML or JSON                  |
      | is empty                             | is empty                                   |
      | declares no components               | manifest declares no components            |
      | declares a component of type quantum | unknown component type "quantum"           |
      | declares storage on medium tape      | unknown storage medium "tape" (ssd or hdd) |
      | declares provider acme-cloud         | unknown provider "acme-cloud"              |

  @id:declared-deployments.manifest-estimates.007 @rule:manifests.format.lenient-key-spelling @verified @source:internal/manifest/manifest.go#LoadManifest @test:internal/manifest/manifest_test.go#TestManifestAcceptsUnderscoresAndBothSpellingsOfUtilisation
  Scenario: A manifest may be written in YAML or JSON with hyphenated or underscored keys and either spelling of utilisation
    Given the manifest is JSON using functional_unit, period_hours, memory_gb and utilization
    When the operator estimates the manifest
    Then the functional-unit quantity, memory and utilisation are read as if written functional-unit, memory-gb and utilisation
