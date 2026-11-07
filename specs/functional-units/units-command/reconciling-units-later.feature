@domain:functional-units @area:units-command @from:cmd/sci-disclose
Feature: Reconciling the unit count after the measurement
  Operators whose functional units accrue after a measurement (a month of requests) re-express an existing disclosure per the real count without measuring again.

  Background:
    Given a JSON disclosure disclosure.json with a total of 860000 gCO2e, measured per 1 "run"

  @id:functional-units.units-command.001 @rule:units.reconcile.sci-recomputed-from-total @inferred @source:cmd/sci-disclose/disclosures.go#cmdUnits
  Scenario: A later unit count re-divides the disclosed total, leaving E, I and M unchanged
    When the operator runs sci-disclose units -units 4300000 -unit-label request disclosure.json
    Then the SCI is 0.2 gCO2e per request
    And the functional unit says it was supplied after the measurement by `sci-disclose units`
    And the notes say R was reconciled after the fact and E, I and M are unchanged
    And the exit status is 0

  @id:functional-units.units-command.002 @rule:units.reconcile.budget-checked-on-new-rate @inferred @source:cmd/sci-disclose/disclosures.go#cmdUnits
  Scenario: A budget is checked against the reconciled rate
    When the operator runs sci-disclose units -units 4300000 -budget 0.1 disclosure.json
    Then the disclosure shows the 0.2 gCO2e per run rate as over the 0.1 budget
    And the exit status is 1

  @id:functional-units.units-command.003 @rule:units.reconcile.invalid-input-refused @inferred @source:cmd/sci-disclose/disclosures.go#cmdUnits @questionable
  Scenario Outline: Reconciliation is refused when there is nothing valid to divide
    Given <situation>
    When the operator runs sci-disclose units on it
    Then the message "<message>" is shown
    And the exit status is 2

    Examples:
      | situation                                      | message                                                                                                                   |
      | -units is missing, zero or negative            | -units must be a positive count                                                                                           |
      | no disclosure file, or more than one, is given | units takes one JSON disclosure, after its flags: `sci-disclose units -units 4300000 -unit-label request disclosure.json` |
      | the disclosure carries a total of 0            | disclosure.json carries no total to divide                                                                                |
      | the file is not JSON                           | disclosure.json is not a JSON disclosure from `sci ... -format json`: <reason>                                            |
