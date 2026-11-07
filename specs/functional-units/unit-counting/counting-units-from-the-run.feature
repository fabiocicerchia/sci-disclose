@domain:functional-units @area:unit-counting @from:cmd/sci-disclose
Feature: Counting functional units from a measured run
  Operators express a run's carbon per real unit of work (requests, items) by having sci-disclose read the count the workload reports.

  @id:functional-units.unit-counting.001 @rule:units.counting.count-read-from-chosen-source @inferred @source:cmd/sci-disclose/measure.go#countRunUnits
  Scenario Outline: The unit count is read from the source the operator names and its origin is disclosed
    Given a workload whose <source> reports 5000 units
    When the operator measures it with sci-disclose run <flag>
    Then the functional unit quantity is 5000
    And the disclosure states the count was <origin>

    Examples:
      | source                                                                                | flag                                                                  | origin                                          |
      | stdout line "SCI-UNITS: 5000"                                                         | -units-from-stdout                                                    | counted from the workload's stdout              |
      | file out/units.txt                                                                    | -units-file out/units.txt                                             | counted from out/units.txt                      |
      | follow-up command "cat out/units.txt"                                                 | -units-cmd "cat out/units.txt"                                        | counted by `cat out/units.txt`                  |
      | Prometheus counter requests_total at http://localhost:9090/metrics, advancing by 5000 | -units-metric requests_total -units-url http://localhost:9090/metrics | requests_total advanced by 5,000 across the run |

  @id:functional-units.unit-counting.002 @rule:units.counting.explicit-units-win @inferred @source:cmd/sci-disclose/measure.go#commandNotes @questionable
  Scenario: An explicit unit count wins over any counted one, and the disclosure says so
    Given a workload that prints "SCI-UNITS: 5000"
    When the operator measures it with sci-disclose run -units 100 -units-from-stdout
    Then the functional unit quantity is 100
    And the notes say "--units was given explicitly, so the unit count was not read from the workload"

  @id:functional-units.unit-counting.003 @rule:units.counting.uncountable-run-refused @inferred @source:cmd/sci-disclose/measure.go#countRunUnits
  Scenario Outline: A run whose unit count cannot be established is refused rather than disclosed per run
    Given <situation>
    When the operator measures the workload with sci-disclose run
    Then the message "<message>" is shown
    And no disclosure is produced and the exit status is 2

    Examples:
      | situation                                                             | message                                                                                                       |
      | -units-from-stdout is set but the workload printed no matching marker | the workload printed no unit count matching <pattern> — have it print e.g. `SCI-UNITS: 5000`                  |
      | -units-pattern is not a valid regular expression                      | -units-pattern is not a valid regexp: <reason>                                                                |
      | -units-metric is set without -units-url                               | -units-metric needs -units-url                                                                                |
      | the counter requests_total stayed at 120 during the run               | requests_total did not advance during the run (120 to 120): nothing was served, so there is no rate to report |
