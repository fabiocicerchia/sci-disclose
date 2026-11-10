@domain:measurement @area:process-usage @from:energy/usage-other
Feature: Measuring process usage on platforms without peak memory
  Serves operators measuring a command on a non-Unix host (e.g. Windows), who need to know which figures were measured and which must be declared.

  Background:
    Given the operator is running sci-disclose on a non-Unix platform

  @id:measurement.process-usage.001 @rule:measurement.process-usage.cpu-time-is-user-plus-system @inferred @source:internal/energy/usage_other.go#processUsage
  Scenario: CPU time on a non-Unix host is the workload's user plus system time
    Given a command spent 1.2 s of user time and 0.4 s of system time
    When the operator measures the command
    Then the report shows 1.60s CPU
    And the CPU utilisation is that CPU time divided by wall time times the reserved vCPUs, capped at 100%

  @id:measurement.process-usage.002 @rule:measurement.process-usage.undeclared-memory-counts-as-zero @inferred @source:internal/energy/usage_other.go#processUsage @questionable
  Scenario: Without a declared memory size a non-Unix run reports no memory and no memory energy
    Given the operator has not set --memory-gb
    When the operator measures a command
    Then the report shows peak RSS 0 MB
    And the memory line of the energy breakdown is 0 kWh
    And the report gives no warning that memory was not measured
