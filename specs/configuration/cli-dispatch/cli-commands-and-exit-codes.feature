@domain:configuration @area:cli-dispatch @from:cmd/sci-disclose
Feature: Invoking sci-disclose and reading its exit status
  Operators and CI pipelines rely on the help, version and exit-code contract to script sci-disclose and to tell a usage mistake from a failed workload.

  @id:configuration.cli-dispatch.001 @rule:cli.dispatch.help-and-version @verified @source:cmd/sci-disclose/main.go#Run @test:cmd/sci-disclose/main_test.go#TestHelpAndVersion
  Scenario Outline: Help and version are printed and succeed
    When the operator runs sci-disclose with <argument>
    Then the output contains <shown>
    And the exit status is 0

    Examples:
      | argument  | shown                                                                |
      | --help    | the SCI equation "SCI = ((E x I) + M) per R" and the list of targets |
      | help      | the SCI equation "SCI = ((E x I) + M) per R" and the list of targets |
      | -h        | the SCI equation "SCI = ((E x I) + M) per R" and the list of targets |
      | --version | "sci-disclose" followed by the release version                       |
      | version   | "sci-disclose" followed by the release version                       |
      | -v        | "sci-disclose" followed by the release version                       |

  @id:configuration.cli-dispatch.002 @rule:cli.dispatch.usage-errors-exit-two @verified @source:cmd/sci-disclose/main.go#Run @test:cmd/sci-disclose/main_test.go#TestUsageErrorsExitTwo
  Scenario Outline: A usage or configuration mistake is refused with exit status 2
    When the operator runs sci-disclose <invocation>
    Then an error prefixed "sci:" or the usage text is printed
    And no disclosure is produced
    And the exit status is 2

    Examples:
      | invocation                |
      | with no command at all    |
      | nonsense                  |
      | run (with nothing to run) |
      | run -provider moon        |
      | file (with no path)       |
      | compare only-one.json     |
      | estimate -f gone.yml      |

  @id:configuration.cli-dispatch.003 @rule:cli.exit-codes.failed-workload-exit-three @verified @source:cmd/sci-disclose/measure.go#measureCommand @test:cmd/sci-disclose/main_test.go#TestRunReportsAFailedWorkloadAsExitCodeThree
  Scenario: A workload that fails is still disclosed, flagged as unrepresentative, and exits 3
    Given a command that exits with status 3
    When the operator measures it with sci-disclose run
    Then a disclosure is printed
    And its notes say "the workload exited 3: a failed run is not a representative measurement"
    And the exit status is 3, even when a carbon budget was also exceeded
