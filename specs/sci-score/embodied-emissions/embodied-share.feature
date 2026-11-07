@domain:sci-score @area:embodied-emissions @from:sci/sci
Feature: Amortising embodied hardware carbon
  Charges a run only its share of the hardware's manufacturing carbon, by how long it ran and how much of the machine it reserved.

  @id:sci-score.embodied-emissions.001 @rule:sci.embodied.time-and-resource-share @verified @source:internal/sci/sci.go#EmbodiedGCO2e @test:internal/sci/sci_test.go#TestEmbodiedIsTotalTimesTimeShareTimesResourceShare
  Scenario Outline: Embodied carbon is the hardware's total scaled by the run's share of its lifetime and of the machine
    Given the hardware embodies 1000 kgCO2e over a 4-year lifespan
    And the run lasted 8760 hours
    And the operator reserved <reserved> vCPUs of a host with <host> vCPUs
    When the operator scores the run
    Then the report shows a time share of 0.25
    And the report shows a resource share of <resource_share>
    And the report shows embodied carbon of <embodied> gCO2e

    Examples:
      | reserved | host    | resource_share | embodied |
      | 2        | 8       | 0.25           | 62500    |
      | 2        | unknown | 1              | 250000   |
