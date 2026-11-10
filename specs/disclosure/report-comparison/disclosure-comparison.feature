@domain:disclosure @area:report-comparison @from:report/report
Feature: Comparing two disclosures
  Lets a team gate a change in CI on whether its carbon rate rose beyond a tolerance between a before and an after disclosure.

  Background:
    Given the operator has a before and an after JSON disclosure in the same unit, "gCO2e per run"

  @id:disclosure.report-comparison.003 @rule:disclosure.compare.regression-beyond-tolerance @verified @source:internal/report/report.go#CompareReports @test:internal/report/report_test.go#TestCompareFlagsARegressionOutsideTolerance
  Scenario Outline: An increase beyond the tolerance is reported as a regression
    Given the before disclosure scored <before> and the after disclosure scored <after>
    And the tolerance is <tolerance> percent
    When the operator compares the two disclosures
    Then the delta line reads "<delta>" and the verdict is <verdict>

    Examples:
      | before | after | tolerance | delta              | verdict    |
      | 1      | 1.200 | 0         | +0.200000 (+20.0%) | REGRESSION |
      | 1      | 1.200 | 25        | +0.200000 (+20.0%) | ok         |
      | 2      | 1     | 0         | -1 (-50.0%)        | ok         |

  @id:disclosure.report-comparison.005 @rule:disclosure.compare.warns-when-grid-moved @verified @source:internal/report/report.go#RenderComparison @test:internal/report/report_test.go#TestCompareWarnsWhenTheGridMovedUnderneathTheCode
  Scenario Outline: A comparison warns when the grid intensity differs between the two runs
    Given the before disclosure used <before intensity> gCO2e/kWh and the after disclosure used <after intensity> gCO2e/kWh
    When the operator compares the two disclosures
    Then <outcome>

    Examples:
      | before intensity | after intensity | outcome                                                                                                                                                       |
      | 100              | 200             | a warning says "I differs between the two runs (100 vs 200 gCO2e/kWh): part of this delta is the grid, not the code" and suggests pinning it with --intensity |
      | 100              | 100             | no warning is shown                                                                                                                                           |

  @id:disclosure.report-comparison.006 @rule:disclosure.compare.zero-baseline @inferred @source:internal/report/report.go#CompareReports @questionable
  Scenario: A before disclosure scoring zero makes any after score a regression
    Given the before disclosure scored 0
    And the after disclosure scored 0
    When the operator compares the two disclosures
    Then the text verdict is REGRESSION with a delta of "0 (+Inf%)"
    And asking for the comparison in json format fails with an error instead of printing it
