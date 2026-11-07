@domain:measurement @area:repo-workloads @from:discover/discover
Feature: Detecting a repository's own workload
  Lets an operator point sci-disclose at a repository and have it measure the command contributors already run, without inventing one when there is none.

  Background:
    Given the operator has a repository checked out

  @id:measurement.repo-workloads.004 @rule:repo.workload.detected-from-project-layout @verified @source:internal/discover/discover.go#DetectWorkload @test:internal/discover/discover_test.go#TestDetectWorkloadFallsThroughToLanguageDefaults @test:cmd/sci-disclose/main_test.go#TestRepoTargetRunsTheDetectedWorkload
  Scenario Outline: The workload is the command the repository's own layout implies
    Given the repository contains only <marker>
    When the operator measures the repository
    Then the measured command is "<command>"
    And the report says the workload was detected from "<detected_from>"

    Examples:
      | marker                                                  | command       | detected_from               |
      | a Makefile with a `test:` target                        | make test     | Makefile `test` target      |
      | a Makefile with a `check:` target and no `test:` target | make check    | Makefile `check` target     |
      | a package.json with a "test" script                     | npm test      | package.json `test` script  |
      | a package.json with only a "build" script               | npm run build | package.json `build` script |
      | a go.mod                                                | go test ./... | go.mod                      |
      | a Cargo.toml                                            | cargo test    | Cargo.toml                  |
      | a pyproject.toml                                        | pytest -q     | Python test layout          |
      | a tests directory                                       | pytest -q     | Python test layout          |

  @id:measurement.repo-workloads.005 @rule:repo.workload.makefile-takes-precedence @verified @source:internal/discover/discover.go#DetectWorkload @test:internal/discover/discover_test.go#TestDetectWorkloadPrefersTheMakefile @test:internal/discover/discover_test.go#TestDetectWorkloadFallsThroughToLanguageDefaults
  Scenario: A Makefile target is preferred over language defaults
    Given the repository has a Makefile with a `test:` target
    And the repository also has a go.mod and a package.json with a "test" script
    When the operator measures the repository
    Then the measured command is "make test"

  @id:measurement.repo-workloads.006 @rule:repo.workload.none-found-is-refused @verified @source:internal/discover/discover.go#DetectWorkload @test:internal/discover/discover_test.go#TestDetectWorkloadReturnsNothingForAnEmptyRepo @test:cmd/sci-disclose/main_test.go#TestRepoTargetSaysSoWhenThereIsNoWorkload
  Scenario: A repository with no recognisable workload is refused, not guessed
    Given the repository has no Makefile target, package.json script, go.mod, Cargo.toml or Python test layout
    When the operator measures the repository
    Then the command exits with status 2
    And the output says "no workload found in this repo"
    And the output suggests passing -command or declaring the deployment with `sci-disclose init` and `sci-disclose estimate`
