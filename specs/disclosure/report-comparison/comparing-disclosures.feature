@domain:disclosure @area:report-comparison @from:cmd/sci-disclose
Feature: Comparing two disclosures to catch regressions
  CI owners compare a before and after disclosure and optionally fail the build when carbon per unit rises.

  Background:
    Given a before disclosure at 1.0 gCO2e per run
    And an after disclosure at 2.0 gCO2e per run

  @id:disclosure.report-comparison.001 @rule:compare.gate.regression-fails-only-when-asked @verified @source:cmd/sci-disclose/disclosures.go#cmdCompare @test:cmd/sci-disclose/main_test.go#TestCompareCommandGatesOnRegression
  Scenario Outline: A regression fails the comparison only when the gate is enabled
    When the operator runs sci-disclose compare <flags> <first> <second>
    Then the comparison with its delta is printed
    And the exit status is <status>

    Examples:
      | flags               | first  | second | status |
      | (none)              | before | after  | 0      |
      | -fail-on-regression | before | after  | 1      |
      | -fail-on-regression | after  | before | 0      |

  @id:disclosure.report-comparison.002 @rule:compare.inputs.two-json-disclosures-required @verified @source:cmd/sci-disclose/disclosures.go#loadReport @test:cmd/sci-disclose/main_test.go#TestUsageErrorsExitTwo @questionable
  Scenario Outline: Compare is refused unless given exactly two readable JSON disclosures
    When the operator runs sci-disclose compare with <inputs>
    Then the message "<message>" is shown
    And the exit status is 2

    Examples:
      | inputs                          | message                                                                  |
      | only one file                   | compare takes two JSON reports: before and after                         |
      | a text report as the after file | after.txt is not a JSON disclosure from `sci ... -format json`: <reason> |
