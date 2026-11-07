@domain:energy @area:idle-baseline @from:energy/energy
Feature: Reporting marginal energy above the idle baseline
  Operators on a shared machine can subtract the host's idle draw, so the disclosure attributes only the workload's own share of energy.

  Background:
    Given the host's RAPL energy counters are readable

  @id:energy.idle-baseline.001 @rule:energy.idle.baseline-subtracted-floored-at-zero @verified @source:internal/energy/energy.go#ForSample @test:internal/energy/energy_test.go#TestIdleBaselineIsSubtractedAndNeverGoesNegative
  Scenario Outline: The idle draw is subtracted from measured energy, never below zero
    Given the host idles at <idle_w> W
    When the operator measures a 10-second workload that records 1000 joules, with an idle baseline
    Then the CPU line of the energy breakdown is <joules> joules expressed in kWh
    And the report notes the idle baseline of <idle_w> W was subtracted and that the reported energy is marginal, not total

    Examples:
      | idle_w | joules |
      | 40.0   | 600    |
      | 1000.0 | 0      |

  @id:energy.idle-baseline.002 @rule:energy.idle.only-with-counters-and-duration @inferred @source:internal/energy/energy.go#SampleIdleWatts @questionable
  Scenario Outline: No idle baseline is taken without readable counters or a sampling duration
    Given <condition>
    When the operator measures a workload
    Then no idle baseline is subtracted
    And the report has no marginal-energy note

    Examples:
      | condition                                                                     |
      | the operator leaves --idle-seconds at 0                                       |
      | the operator uses --energy model with --idle-seconds 5                        |
      | the host has no readable RAPL counters and the operator sets --idle-seconds 5 |
