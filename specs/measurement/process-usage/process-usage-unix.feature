@domain:measurement @area:process-usage @from:energy/usage-unix
Feature: Reporting the CPU time and peak memory of a measured command
  Operators who measure a command on Linux or macOS need the CPU seconds and peak memory in the report to match what the whole workload used, because those numbers set utilisation and memory energy.

  Background:
    Given the operator is on a Unix system (Linux or macOS)
    And the operator measures a command with sci-disclose

  @id:measurement.process-usage.004 @rule:measurement.process-usage.cpu-covers-process-tree @inferred @source:internal/energy/usage_unix.go#processUsage
  Scenario: CPU time counts every process the command started, not just the command itself
    Given the measured command is a shell script that runs two child programs
    And each child program uses 3 seconds of CPU and the script itself uses almost none
    When the operator measures the script
    Then the report's "ran in" line shows about 6 s of CPU
    And the CPU time is the sum of user and system time across the script and the children it waited for

  @id:measurement.process-usage.005 @rule:measurement.process-usage.peak-memory-units-per-os @inferred @source:internal/energy/usage_unix.go#processUsage
  Scenario Outline: Peak memory is reported in the same unit on Linux and macOS
    Given the measured command's largest process peaks at 512 MB of resident memory
    And the operator runs on <os>, which reports peak memory in <native_unit>
    When the operator measures the command
    Then the report's "ran in" line shows "peak RSS 512 MB"
    And the JSON report's peak_rss_gb is 0.5

    Examples:
      | os    | native_unit |
      | Linux | kilobytes   |
      | macOS | bytes       |

  @id:measurement.process-usage.006 @rule:measurement.process-usage.peak-memory-is-largest-single-process @inferred @source:internal/energy/usage_unix.go#processUsage @questionable
  Scenario: Peak memory is the largest single process, not the whole tree added up
    Given the measured command starts four worker processes at the same time
    And each worker peaks at 200 MB of resident memory
    When the operator measures the command
    Then the report shows "peak RSS 200 MB", not 800 MB
