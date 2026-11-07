@domain:declared-deployments @area:infrastructure-discovery @from:discover/discover
Feature: Discovering the deployed boundary from infrastructure files
  Gives an operator who is scaffolding a manifest the compute their Kubernetes and Terraform files already declare, and says where each number came from or what was assumed.

  Background:
    Given the operator scaffolds a manifest for a repository with `sci-disclose init`

  @id:declared-deployments.infrastructure-discovery.001 @rule:discovery.kubernetes.requests-become-components @verified @source:internal/discover/discover.go#ScanRepo @test:internal/discover/discover_test.go#TestScanRepoReadsDeploymentsAndTerraform @test:internal/discover/discover_test.go#TestKubernetesQuantitiesAndInstanceSizes
  Scenario Outline: Each Kubernetes workload becomes a component sized by its resource requests
    Given the repository has a <kind> named "<name>" with <replicas> replicas whose containers request cpu <cpu> and memory <memory>
    When the operator scaffolds the manifest
    Then the manifest has a component "<name>" with <vcpus> vCPUs, <replicas_out> replicas and <memory_gb> GB of memory
    And the component is commented "<kind> from <file>"

    Examples:
      | kind        | name    | replicas | cpu  | memory | file        | vcpus | replicas_out | memory_gb |
      | Deployment  | api     | 3        | 500m | 512Mi  | deploy.yaml | 0.5   | 3            | 0.5       |
      | StatefulSet | db      | 2        | 2    | 2Gi    | k8s/db.yaml | 2     | 2            | 2         |
      | CronJob     | nightly | unset    | "2"  | unset  | cron.yaml   | 2     | 1            | 0         |

  @id:declared-deployments.infrastructure-discovery.002 @rule:discovery.kubernetes.missing-cpu-request-assumed-and-stated @inferred @source:internal/discover/discover.go#ScanRepo @questionable
  Scenario: A workload with no CPU request is assumed to use 1 vCPU, and the scaffold says so
    Given the repository has a Deployment "worker" whose containers set no CPU request
    When the operator scaffolds the manifest
    Then the component "worker" has 1 vCPU
    And its comment ends "no CPU requests set, assumed 1 vCPU"

  @id:declared-deployments.infrastructure-discovery.003 @rule:discovery.terraform.instance-size-to-vcpus @verified @source:internal/discover/discover.go#ScanRepo @test:internal/discover/discover_test.go#TestScanRepoReadsDeploymentsAndTerraform @test:internal/discover/discover_test.go#TestKubernetesQuantitiesAndInstanceSizes @questionable
  Scenario Outline: A Terraform instance type is sized from its size suffix, and an unknown size is guessed at 2 vCPUs
    Given the repository has a Terraform file "main.tf" declaring instance_type "<instance_type>"
    When the operator scaffolds the manifest
    Then the manifest has a component "<instance_type> (main.tf)" with <vcpus> vCPUs and 1 replica
    And the component is commented "<comment>"

    Examples:
      | instance_type | vcpus | comment                                |
      | c6i.4xlarge   | 16    | from Terraform                         |
      | m5.2xlarge    | 8     | from Terraform                         |
      | t3.micro      | 2     | from Terraform                         |
      | m6g.metal     | 2     | from Terraform; vCPUs unknown, guessed |

  @id:declared-deployments.infrastructure-discovery.004 @rule:discovery.scan.skips-dependency-and-build-dirs @verified @source:internal/discover/discover.go#ScanRepo @test:internal/discover/discover_test.go#TestScanRepoReadsDeploymentsAndTerraform
  Scenario Outline: Vendored dependencies and build output are not counted as part of the deployment
    Given the repository has a Deployment manifest "skipme" under <directory>
    When the operator scaffolds the manifest
    Then the manifest has no component "skipme"

    Examples:
      | directory          |
      | node_modules/pkg   |
      | vendor/lib         |
      | .terraform/modules |
      | dist               |

  @id:declared-deployments.infrastructure-discovery.005 @rule:discovery.scan.partial-scan-is-disclosed @inferred @source:internal/discover/discover.go#ScanRepo @questionable
  Scenario: A path the scan could not read is disclosed rather than silently left out
    Given the repository has a subdirectory "secrets" the operator cannot read
    When the operator scaffolds the manifest
    Then the manifest ends with a note "skipped <path>/secrets: open <path>/secrets: permission denied"
    And components from the readable files are still listed
