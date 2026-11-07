@domain:functional-units @area:unit-counting @from:units/units
Feature: Counting functional units from the workload's own output
  Lets an operator take R, the functional unit count, from what the measured workload already prints, writes or can be asked for, so the SCI rate is traceable instead of typed in by hand.

  Background:
    Given the operator measures a workload with the run command

  @id:functional-units.unit-counting.004 @rule:units.counting.marker-spellings @verified @source:internal/units/units.go#ScanUnits @test:internal/units/units_test.go#TestScanUnitsAcceptsTheUsualMarkerSpellings @test:cmd/sci-disclose/units_cli_test.go#TestUnitsAreReadFromTheWorkloadsOwnOutput
  Scenario Outline: The workload's unit marker is recognised in its usual spellings
    Given the operator asked for the unit count from the workload's output
    When the workload prints the line "<line>"
    Then the functional unit quantity in the report is <count>
    And the report names the workload's stdout as the origin of the count

    Examples:
      | line                       | count |
      | SCI-UNITS: 5000            | 5000  |
      | sci_units=5000             | 5000  |
      | sci units 5000             | 5000  |
      | done. SCI-UNIT: 5000 in 3s | 5000  |
      | SCI-Units:5000             | 5000  |

  @id:functional-units.unit-counting.005 @rule:units.counting.last-reported-count-wins @verified @source:internal/units/units.go#ScanUnits @test:internal/units/units_test.go#TestScanUnitsTakesTheWorkloadsFinalWord
  Scenario: A workload that reports progress is counted at its final figure
    Given the operator asked for the unit count from the workload's output
    When the workload prints "SCI-UNITS: 100", then "still going", then "SCI-UNITS: 2500"
    Then the functional unit quantity in the report is 2500

  @id:functional-units.unit-counting.006 @rule:units.counting.no-count-is-refused @verified @source:internal/units/units.go#ScanUnits @test:internal/units/units_test.go#TestScanUnitsRejectsNonsense @test:cmd/sci-disclose/units_cli_test.go#TestAMissingMarkerIsAnErrorNotASilentOne
  Scenario Outline: A workload that reports no positive count is refused, not counted as one
    Given the operator asked for the unit count from the workload's output
    When the workload's output is "<output>"
    Then the run fails with exit status 2
    And the message says the workload printed no unit count and suggests printing e.g. `SCI-UNITS: 5000`
    And no report is produced

    Examples:
      | output         |
      | nothing        |
      | no marker here |
      | SCI-UNITS: 0   |
      | SCI-UNITS: -5  |

  @id:functional-units.unit-counting.007 @rule:units.counting.from-written-file @verified @source:internal/units/units.go#FromFile @test:internal/units/units_test.go#TestUnitsFromFileTakesABareNumberOrAMarker
  Scenario Outline: The unit count can come from a file the workload wrote
    Given the operator named a unit-count file for the run
    And the workload leaves that file holding "<contents>"
    When the run finishes
    Then the functional unit quantity in the report is <count>
    And the report names that file as the origin of the count

    Examples:
      | contents                | count |
      | 4200                    | 4200  |
      | finished\nSCI-UNITS: 77 | 77    |

  @id:functional-units.unit-counting.008 @rule:units.counting.unreadable-file-is-refused @verified @source:internal/units/units.go#FromFile @test:internal/units/units_test.go#TestUnitsFromFileTakesABareNumberOrAMarker
  Scenario Outline: A unit-count file that is missing or holds no count is refused
    Given the operator named a unit-count file for the run
    And <state>
    When the run finishes
    Then the run fails with a message that <message>
    And no report is produced

    Examples:
      | state                             | message                                                                             |
      | the file holds "no number at all" | the file holds no unit count, expected a bare number or a line matching the pattern |
      | the file does not exist           | the unit count cannot be read                                                       |

  @id:functional-units.unit-counting.009 @rule:units.counting.from-follow-up-command @verified @source:internal/units/units.go#FromCommand @test:internal/units/units_test.go#TestUnitsFromCommandReadsStdout
  Scenario Outline: The unit count can come from a command run after the workload
    Given the operator gave the follow-up count command "<command>"
    When the workload finishes
    Then the functional unit quantity in the report is <count>
    And the report names that command as the origin of the count

    Examples:
      | command                 | count |
      | echo 640                | 640   |
      | printf 'a\nb\n' \| wc -l | 2     |

  @id:functional-units.unit-counting.010 @rule:units.counting.failed-follow-up-command-is-refused @verified @source:internal/units/units.go#FromCommand @test:internal/units/units_test.go#TestUnitsFromCommandReadsStdout
  Scenario Outline: A follow-up count command that fails or prints no count is refused
    Given the operator gave the follow-up count command "<command>"
    When the workload finishes
    Then the run fails with a message that <message>
    And no report is produced

    Examples:
      | command        | message                                      |
      | sh -c 'exit 1' | --units-cmd failed                           |
      | echo nothing   | --units-cmd printed no unit count: "nothing" |
