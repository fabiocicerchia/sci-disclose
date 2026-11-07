@domain:sci-score @area:carbon-budgets @from:sci/sci
Feature: Gating a run on a carbon budget
  Lets an operator fail a CI run whose SCI exceeds an agreed budget.

  Background:
    Given the run scored an SCI of 1.2 gCO2e per request

  @id:sci-score.carbon-budgets.003 @rule:sci.budget.score-at-or-below-limit-passes @verified @source:internal/sci/sci.go#ApplyBudget @test:internal/sci/sci_test.go#TestBudgetVerdictIsAttachedAndReturned
  Scenario Outline: A score at or below the budget passes and one above it fails
    Given the operator set a budget of <limit> gCO2e per request
    When the operator measures the run
    Then the report shows a budget of <limit> with verdict <verdict>
    And the command exits with status <exit>

    Examples:
      | limit | verdict | exit |
      | 2.4   | pass    | 0    |
      | 1.2   | pass    | 0    |
      | 0.6   | fail    | 1    |

  @id:sci-score.carbon-budgets.004 @rule:sci.budget.no-verdict-without-budget @verified @source:internal/sci/sci.go#ApplyBudget @test:internal/sci/sci_test.go#TestBudgetVerdictIsAttachedAndReturned
  Scenario: A run without a budget carries no budget verdict and is never failed for carbon
    Given the operator set no budget
    When the operator measures the run
    Then the report has no budget section
    And the command exits with status 0 when the workload succeeded
