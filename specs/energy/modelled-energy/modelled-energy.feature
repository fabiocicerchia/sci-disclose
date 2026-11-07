@domain:energy @area:modelled-energy @from:energy/energy
Feature: Estimating energy from utilisation and power profiles
  When counters are unavailable or not wanted, operators get an energy estimate built from CPU time, reserved capacity and published coefficients, broken down line by line.

  Background:
    Given the operator measures a workload with sci-disclose

  @id:energy.modelled-energy.001 @rule:energy.model.used-without-counters-or-on-request @inferred @source:internal/energy/energy.go#ForSample
  Scenario Outline: The model is used when counters are unreadable or the operator asks for it
    Given the host's RAPL counters are <counters>
    When the operator measures a workload with --energy <source>
    Then the report gives the energy source as "model"
    And the energy breakdown has a CPU line and a memory line

    Examples:
      | counters     | source |
      | not readable | auto   |
      | readable     | model  |

  @id:energy.modelled-energy.002 @rule:energy.model.cpu-linear-in-utilisation @verified @source:internal/energy/energy.go#CPUkWh @test:internal/energy/energy_test.go#TestCPUModelHitsIdleAndFullLoadEndpoints
  Scenario Outline: Modelled CPU energy rises linearly from idle to full draw per reserved vCPU
    Given the power profile draws <min_w> W per vCPU idle and <max_w> W at full load, with no datacentre overhead
    When the operator measures a one-hour workload at <utilisation> utilisation on <vcpus> vCPUs with the model
    Then the CPU line of the energy breakdown is <kwh> kWh

    Examples:
      | min_w | max_w | utilisation | vcpus | kwh   |
      | 1     | 5     | 0           | 1     | 0.001 |
      | 1     | 5     | 1           | 1     | 0.005 |
      | 1     | 5     | 0.5         | 4     | 0.012 |

  @id:energy.modelled-energy.003 @rule:energy.model.utilisation-of-reserved-capacity @verified @source:internal/energy/energy.go#Utilisation @test:internal/energy/energy_test.go#TestUtilisationIsAFractionOfReservedCapacity
  Scenario Outline: Utilisation is CPU time over reserved capacity, capped at full
    Given the operator reserves <vcpus> vCPUs for the workload
    When the operator measures a workload that uses <cpu_s> s of CPU over <wall_s> s of wall time
    Then the reported utilisation is <utilisation>

    Examples:
      | vcpus | cpu_s | wall_s | utilisation |
      | 1     | 2     | 4      | 0.5         |
      | 4     | 8     | 4      | 0.5         |
      | 1     | 99    | 1      | 1.0         |
      | 1     | 1     | 0      | 0           |

  @id:energy.modelled-energy.004 @rule:energy.model.memory-defaults-to-peak-rss @inferred @source:internal/energy/energy.go#ForSample
  Scenario Outline: Memory energy uses the workload's peak memory unless the operator reserves an amount
    Given the workload's peak resident memory is 0.5 GB
    When the operator measures the workload for one hour with the model <flag>
    Then memory energy is modelled on <memory_gb> GB of memory
    And the memory line of the energy breakdown is <kwh> kWh

    Examples:
      | flag               | memory_gb | kwh      |
      | and no --memory-gb | 0.5       | 0.000196 |
      | and --memory-gb 10 | 10        | 0.00392  |

  @id:energy.modelled-energy.005 @rule:energy.model.storage-and-network-when-provisioned @verified @source:internal/energy/energy.go#ForSample @test:internal/energy/energy_test.go#TestMemoryStorageAndNetworkUseThePublishedCoefficients
  Scenario Outline: Storage and network energy appear only when the operator declares them
    Given the operator declares <declared>
    When the operator measures a one-hour workload
    Then the energy breakdown has a <line> line of <kwh> kWh

    Examples:
      | declared                               | line    | kwh    |
      | --storage-gb 1024 --storage-medium ssd | storage | 0.0012 |
      | --network-gb 50                        | network | 0.05   |

  @id:energy.modelled-energy.006 @rule:energy.pue.datacentre-overhead-line @verified @source:internal/energy/energy.go#ForSample @test:internal/energy/energy_test.go#TestPUEShowsUpAsDatacentreOverhead
  Scenario: Datacentre overhead from PUE is shown as its own line and included in the total
    Given the operator sets --pue 2
    When the operator measures a workload
    Then the energy breakdown has a datacentre_overhead line equal to the sum of the other lines
    And the total energy is twice the sum of the other lines
