@domain:measurement @area:function-benchmarks @from:harness/harness
Feature: Benchmarking a single function
  Lets a developer measure the carbon of one Python function per call, without the interpreter's startup and imports inflating the number.

  Background:
    Given python3 is installed
    And the developer's working directory holds a Python module workload_under_test.py defining a function go

  @id:measurement.function-benchmarks.004 @rule:measurement.function-benchmarks.per-call-measurement @verified @source:internal/harness/harness.go#MeasureFunction @test:cmd/sci-disclose/main_test.go#TestFuncTargetDividesByTheIterationCount
  Scenario: A function is scored per measured call, with warmup, startup and imports left out
    Given the developer asks for 50 measured calls and 1 warmup call
    When the developer benchmarks the function workload_under_test:go
    Then the functional unit is 50 "call"
    And the SCI score equals the total emissions divided by 50
    And the measurement shows a CPU time and a wall time greater than zero
    And the disclosure notes that startup and import time are outside the bracket and that 1 warmup call was discarded

  @id:measurement.function-benchmarks.005 @rule:measurement.function-benchmarks.bad-reference-refused @verified @source:internal/harness/harness.go#pythonHarness @test:cmd/sci-disclose/main_test.go#TestFuncTargetRejectsABadReference
  Scenario Outline: A function reference that cannot be resolved is refused before anything is measured
    Given the developer names the function as <reference>
    When the developer benchmarks the function
    Then the interpreter prints "<message>"
    And sci-disclose reports "sci: the python3 harness failed"
    And no report is written
    And the command exits with status 2

    Examples:
      | reference                    | message                                                               |
      | os.path                      | sci: function target must be module:attribute, e.g. mypkg.bench:main  |
      | no_such_module:go            | sci: cannot import 'no_such_module': No module named 'no_such_module' |
      | workload_under_test:missing  | sci: workload_under_test has no attribute 'missing'                   |
      | workload_under_test:CONSTANT | sci: workload_under_test:CONSTANT is not callable                     |

  @id:measurement.function-benchmarks.006 @rule:measurement.function-benchmarks.modelled-without-hardware-counters @inferred @source:internal/harness/harness.go#MeasureFunction
  Scenario: Without readable hardware energy counters, a function's energy is modelled and the report says so
    Given the machine's hardware energy counters are unreadable, or the developer has chosen modelled energy
    When the developer benchmarks the function workload_under_test:go
    Then the measurement reports no hardware energy reading
    And the energy is estimated from the measured CPU time and memory
    And the disclosure carries the modelled-energy note

  @id:measurement.function-benchmarks.007 @rule:measurement.function-benchmarks.function-output-shared-with-report @inferred @source:internal/harness/harness.go#MeasureFunction @questionable
  Scenario: Whatever the measured function prints goes to the same output as the report
    Given the function prints a line each time it is called
    And the developer asks for the report as JSON on standard output
    When the developer benchmarks the function
    Then the function's printed lines appear on standard output before the report
    And standard output is no longer a single JSON document
