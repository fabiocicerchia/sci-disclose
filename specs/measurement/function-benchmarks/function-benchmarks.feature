@domain:measurement @area:function-benchmarks @from:cmd/sci-disclose
Feature: Measuring a function per call
  Developers get the carbon of one function call by timing many calls in a harness and dividing by the call count.

  @id:measurement.function-benchmarks.001 @rule:measurement.functions.sci-per-call @verified @source:cmd/sci-disclose/measure.go#cmdFunc @test:cmd/sci-disclose/main_test.go#TestFuncTargetDividesByTheIterationCount
  Scenario: A function's SCI is its total divided by the number of measured calls
    Given a Python function workload_under_test:go
    When the developer runs sci-disclose func -n 50 -warmup 1 on it
    Then the functional unit is 50 x "call"
    And the SCI equals the total emissions divided by 50
    And the notes say 50 calls were measured inside the interpreter, so startup and import time are outside the bracket
    And the notes say 1 warmup call was discarded first

  @id:measurement.function-benchmarks.002 @rule:measurement.functions.unrunnable-function-refused @verified @source:cmd/sci-disclose/measure.go#cmdFunc @test:cmd/sci-disclose/main_test.go#TestFuncTargetRejectsABadReference
  Scenario Outline: A function benchmark that cannot run is refused
    Given <situation>
    When the developer runs sci-disclose func
    Then an error is shown, such as "<message>"
    And no disclosure is produced and the exit status is 2

    Examples:
      | situation                                                | message                                                       |
      | -n 0 was given                                           | -n must be at least 1                                         |
      | no function reference was given                          | which function? — `sci-disclose func mypkg.bench:main -n 200` |
      | the -python interpreter is not installed                 | interpreter not installed: <the -python value, python3 by default> |
      | the reference names a missing module (no_such_module:go) | the harness's error                                           |
      | the reference has no function part (os.path)             | the harness's error                                           |

  @id:measurement.function-benchmarks.003 @rule:measurement.functions.short-run-warned @inferred @source:cmd/sci-disclose/measure.go#funcNotes
  Scenario: A function run shorter than 50 ms carries a warning to raise the call count
    Given a function whose measured calls take under 50 ms in total
    When the developer runs sci-disclose func on it
    Then the notes say "the run was shorter than 50 ms: raise -n until the measurement is stable"
