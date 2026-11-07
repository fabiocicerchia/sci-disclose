@domain:measurement @area:repo-workloads @from:cmd/sci-disclose
Feature: Measuring a repository through its own workload
  Maintainers measure a repository by running the test or build command it already declares, without writing a benchmark.

  @id:measurement.repo-workloads.001 @rule:measurement.repos.detected-workload-measured @verified @source:cmd/sci-disclose/measure.go#cmdRepo @test:cmd/sci-disclose/main_test.go#TestRepoTargetRunsTheDetectedWorkload
  Scenario: A repository is measured by running its detected workload, disclosed per run of it
    Given a repository whose Makefile has a test target
    When the maintainer runs sci-disclose repo on it
    Then the disclosure target kind is "repo" and says the workload was detected from the test target
    And the SCI is expressed per "make test run"

  @id:measurement.repo-workloads.002 @rule:measurement.repos.command-overrides-detection @inferred @source:cmd/sci-disclose/measure.go#cmdRepo @questionable
  Scenario: An explicit workload command replaces the detected one
    Given a repository
    When the maintainer runs sci-disclose repo -command "go test ./..." on it
    Then "go test ./..." is run inside the repository
    And the disclosure says the workload came from --command
    And the SCI is expressed per "go test ./... run"

  @id:measurement.repo-workloads.003 @rule:measurement.repos.no-runnable-workload-refused @verified @source:cmd/sci-disclose/measure.go#cmdRepo @test:cmd/sci-disclose/main_test.go#TestRepoTargetSaysSoWhenThereIsNoWorkload
  Scenario Outline: A repository with no runnable workload is refused and pointed at declaring one instead
    Given <situation>
    When the maintainer runs sci-disclose repo on it
    Then the message "<message>" is shown
    And the exit status is 2

    Examples:
      | situation                                     | message                                                                                                                                                                      |
      | the path is not a directory                   | not a directory: <absolute path>                                                                                                                                             |
      | no test or build workload can be detected     | no workload found in this repo. A repository has no SCI until something runs — pass -command, or declare the deployment with `sci-disclose init` and `sci-disclose estimate` |
      | the detected workload's tool is not installed | "make" is not installed, so the detected workload (make test) cannot run                                                                                                     |
