@domain:measurement @area:command-runs @from:cmd/sci-disclose
Feature: Measuring a command or a script
  Developers measure the carbon of an arbitrary command or script file by running it under sci-disclose and getting a disclosure per run.

  @id:measurement.command-runs.001 @rule:measurement.commands.run-disclosed-per-run @verified @source:cmd/sci-disclose/measure.go#cmdRun @test:cmd/sci-disclose/main_test.go#TestRunMeasuresACommandAndEmitsJSON
  Scenario: A measured command is disclosed per run by default
    Given the grid is set offline to region eu-north-1
    When the operator runs sci-disclose run -format json -- followed by a command
    Then the disclosure names the command as its target
    And the SCI is greater than zero and expressed per "run"
    And the intensity used is the bundled yearly average for SE
    And the disclosure records tool "sci-disclose" and its version

  @id:measurement.command-runs.002 @rule:measurement.scripts.interpreter-by-extension @verified @source:cmd/sci-disclose/measure.go#cmdFile @test:cmd/sci-disclose/main_test.go#TestFileTargetRunsTheScript
  Scenario Outline: A script runs under the interpreter its extension implies, disclosed per execution
    Given a script file named <script>
    When the operator runs sci-disclose file <script>
    Then the script is run as "<interpreter> <script>"
    And the disclosure target kind is "file"
    And the SCI is expressed per "execution"

    Examples:
      | script   | interpreter |
      | work.sh  | sh          |
      | bench.py | python3     |
      | bench.js | node        |
      | bench.rb | ruby        |
      | main.go  | go run      |

  @id:measurement.command-runs.003 @rule:measurement.scripts.unrunnable-script-refused @verified @source:cmd/sci-disclose/measure.go#cmdFile @test:cmd/sci-disclose/main_test.go#TestFileTargetNeedsAKnownInterpreter
  Scenario Outline: A script that cannot be run is refused before anything is measured
    Given <situation>
    When the operator runs sci-disclose file on it
    Then the message "<message>" is shown
    And the exit status is 2

    Examples:
      | situation                                                              | message                                            |
      | the path does not exist or is a directory                              | no such file: <path>                               |
      | the file has an extension with no known interpreter, such as thing.zzz | no interpreter known for ".zzz"; pass -interpreter |
      | the interpreter for its extension is not installed                     | interpreter not installed: <interpreter>           |

  @id:measurement.command-runs.004 @rule:energy.backend.rapl-required-but-unreadable-refused @inferred @source:cmd/sci-disclose/measure.go#energyBackend
  Scenario: Requiring hardware counters on a host without readable ones is refused with a fix
    Given a host where no RAPL energy counters are readable
    When the operator measures a workload with --energy rapl
    Then the message says RAPL was requested but no readable counters exist
    And it suggests "sudo chmod -R a+r /sys/class/powercap/intel-rapl*" or running as root
    And nothing is measured and the exit status is 2
