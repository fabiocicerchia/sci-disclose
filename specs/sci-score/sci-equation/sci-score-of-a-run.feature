@domain:sci-score @area:sci-equation @from:sci/sci
Feature: Scoring a measured run
  Gives an operator the SCI rate of a measured workload, ((E × I) + M) per functional unit, with the caveats and boundary a reviewer needs to trust it.

  Background:
    Given the operator has measured a workload run

  @id:sci-score.sci-equation.001 @rule:sci.equation.rate-per-functional-unit @verified @source:internal/sci/sci.go#New @test:internal/sci/sci_test.go#TestSCIIsTotalCarbonDividedByTheFunctionalUnit
  Scenario Outline: The score is total carbon divided by the functional units served
    Given the run used <energy> kWh
    And the grid intensity is <intensity> gCO2e/kWh
    And the embodied share of the hardware is <embodied> gCO2e
    And the run served <units> <label>s
    When the operator scores the run
    Then the report shows operational carbon of <operational> gCO2e and a total of <total> gCO2e
    And the report shows an SCI of <sci> "gCO2e per <label>"

    Examples:
      | energy | intensity | embodied | units | label   | operational | total | sci |
      | 0.2    | 500       | 20       | 100   | request | 100         | 120   | 1.2 |
      | 0.01   | 300       | 1        | 4     | build   | 3           | 4     | 1   |

  @id:sci-score.sci-equation.002 @rule:sci.equation.uncounted-units-score-as-one @inferred @source:internal/sci/sci.go#New @questionable
  Scenario Outline: A run with no counted functional units is scored as a single unit
    Given the run's functional unit count is <count>
    When the operator scores the run
    Then the report shows a functional unit quantity of 1
    And the SCI equals the run's total carbon

    Examples:
      | count |
      | 0     |
      | -3    |

  @id:sci-score.sci-equation.003 @rule:sci.equation.intensity-caveats-disclosed @verified @source:internal/sci/sci.go#New @test:cmd/sci-disclose/crosscutting_test.go#TestAFallenBackBasisIsNotedInTheReport
  Scenario Outline: A grid intensity that is not a current measured reading is called out in the report
    Given the grid intensity reading is <condition>
    When the operator scores the run
    Then the report's notes include "<note>"

    Examples:
      | condition                                                                        | note                                                                                         |
      | generated more than 65 minutes ago or on a non-measured basis                                         | the intensity reading is not a current measured hour (basis or age); see its source line                  |
      | modelled rather than measured                                                                         | the intensity reading is modelled rather than measured from a live grid feed                              |
      | on the lifecycle basis because the requested consumption_lifecycle is not published for the zone | intensity basis fell back to lifecycle: consumption_lifecycle is not published for this location |

  @id:sci-score.sci-equation.004 @rule:sci.equation.boundary-and-assumptions-stated @inferred @source:internal/sci/sci.go#New
  Scenario: Every score states its boundary and the assumptions it used
    Given the provider profile is "aws" with a PUE of 1.135
    And the operator reserved 2 of the host's 8 vCPUs
    When the operator scores the run
    Then the report's boundary lists compute, memory, storage and network, embodied hardware and datacentre overhead via PUE
    And the report's boundary includes "provider profile: aws, PUE 1.135"
    And the report's assumptions show provider aws, PUE 1.135, 2 reserved vCPUs and 8 host vCPUs
    And the report's measurement shows the host as its OS, architecture and "8 vCPU"

  @id:sci-score.sci-equation.005 @rule:sci.equation.no-score-without-energy-or-intensity @inferred @source:internal/sci/sci.go#New
  Scenario Outline: No score is produced when the run's energy or grid intensity cannot be determined
    Given the <input> for the run cannot be determined
    When the operator scores the run
    Then no report is produced
    And the operator sees the reason the <input> could not be determined

    Examples:
      | input          |
      | grid intensity |
      | energy         |
