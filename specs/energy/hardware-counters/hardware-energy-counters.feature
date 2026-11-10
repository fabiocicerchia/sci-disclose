@domain:energy @area:hardware-counters @from:energy/energy
Feature: Measuring energy from hardware power counters
  Operators measuring on a Linux host get energy figures read from the CPU's own RAPL counters, with the report saying where the number came from and what it covers.

  Background:
    Given the operator measures a workload with sci-disclose

  @id:energy.hardware-counters.001 @rule:energy.rapl.preferred-when-readable @verified @source:internal/energy/energy.go#ForSample @test:internal/energy/energy_test.go#TestRAPLJoulesAreConvertedAndPreferredOverTheModel
  Scenario: Readable hardware counters are used in preference to the model
    Given the host's RAPL energy counters are readable
    And the energy source is left at auto
    When the operator measures a workload whose counters record 3,600,000 joules
    Then the report gives the energy source as "rapl"
    And the CPU line of the energy breakdown is 1 kWh
    And the report notes that RAPL counts the whole machine's CPU package, not just this workload, and advises measuring on an otherwise idle host

  @id:energy.hardware-counters.002 @rule:energy.rapl.memory-not-double-counted @verified @source:internal/energy/energy.go#ForSample @test:internal/energy/energy_test.go#TestRAPLJoulesAreConvertedAndPreferredOverTheModel @test:internal/energy/energy_test.go#TestRAPLWithoutADRAMDomainAddsModelledMemory
  Scenario Outline: Memory energy is modelled only when the counters do not already include it
    Given the host's RAPL energy counters are readable
    And the host <dram> a separate DRAM counter
    And the operator reserves 8 GB of memory
    When the operator measures a workload for one hour
    Then the energy breakdown <memory_line>
    And <note>

    Examples:
      | dram            | memory_line                                | note                                                          |
      | exposes         | has no memory line                         | the report notes that memory energy is inside the RAPL figure |
      | does not expose | has a modelled memory line of 0.003136 kWh | no DRAM note is shown                                         |

  @id:energy.hardware-counters.003 @rule:energy.rapl.package-and-dram-only @verified @source:internal/energy/energy.go#RAPLReader @test:internal/energy/energy_test.go#TestRAPLCountsPackageAndDRAMButNotTheCoreSubdomain @test:internal/energy/energy_test.go#TestRAPLReaderSumsTheDeltaAcrossDomains
  Scenario: Hardware energy is the sum of the package and DRAM counters, never their subdomains
    Given the host has one CPU package with a core subdomain and a DRAM subdomain, all readable
    When the operator measures a workload during which the package counter rises by 2,000,000 µJ and the DRAM counter by 200,000 µJ
    Then the measured energy is 2.2 joules
    And the core subdomain's energy is not added a second time
    And the run is recorded as covering DRAM

  @id:energy.hardware-counters.004 @rule:energy.rapl.unreadable-counter-skipped @verified @source:internal/energy/energy.go#NewRAPLReader @test:internal/energy/energy_test.go#TestUnreadableCountersAreSkippedRatherThanFatal @questionable
  Scenario: An unreadable counter is left out of the figure instead of failing the run
    Given the host's DRAM counter is readable
    And the package counter is not readable by the operator
    When the operator measures a workload
    Then the run completes
    And the package counter contributes nothing to the measured energy
