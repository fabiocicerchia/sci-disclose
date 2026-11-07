@domain:sci-score @area:carbon-budgets @from:cmd/sci-disclose
Feature: Failing a run that exceeds its carbon budget
  CI owners gate merges on a per-unit carbon budget, so an over-budget measurement fails the pipeline visibly.

  @id:sci-score.carbon-budgets.001 @rule:budgets.gate.over-budget-exits-one @verified @source:cmd/sci-disclose/measure.go#measureCommand @test:cmd/sci-disclose/main_test.go#TestBudgetGateFailsTheCommand
  Scenario Outline: A score over the budget is marked OVER and exits 1; within budget exits 0
    Given a workload that completes normally
    When the operator measures it with -budget <budget>
    Then the disclosure shows the verdict <verdict>
    And the exit status is <status>

    Examples:
      | budget  | verdict       | status |
      | 0       | OVER          | 1      |
      | 1000000 | within budget | 0      |

  @id:sci-score.carbon-budgets.002 @rule:budgets.gate.no-budget-never-fails @verified @source:cmd/sci-disclose/options.go#finish @test:cmd/sci-disclose/main_test.go#TestRunMeasuresACommandAndEmitsJSON
  Scenario: Without a budget, no score fails the run
    Given a workload that completes normally
    When the operator measures it without -budget
    Then a disclosure is printed
    And the exit status is 0
