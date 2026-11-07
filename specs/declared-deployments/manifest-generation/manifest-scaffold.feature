@domain:declared-deployments @area:manifest-generation @from:discover/discover
Feature: Scaffolding a starter manifest
  Gives an operator a sci.yaml they can complete and run with estimate, with every value pre-filled and marked as something they still have to confirm.

  Background:
    Given the operator scaffolds a manifest for a repository with `sci-disclose init`

  @id:declared-deployments.manifest-generation.003 @rule:manifest-scaffold.discovered-components-with-starting-values @verified @source:internal/discover/discover.go#RenderManifest @test:cmd/sci-disclose/main_test.go#TestInitThenEstimateRoundTrips
  Scenario: Discovered components are written with starting values the operator is told to fill in
    Given the repository "shop" declares a Deployment "api" with 2 replicas requesting 1 CPU
    When the operator scaffolds the manifest
    Then the manifest is named "shop" and has a functional unit of 1000000 requests over 720 hours on aws eu-west-1
    And it has a compute component "api" with 1 vCPU, 2 replicas and utilisation 0.5
    And the component has embodied defaults of 1200 kg, a 4-year lifespan and 64 total vCPUs
    And the header says utilisation and the functional unit are the operator's to fill in
    And `sci-disclose estimate` accepts the manifest unchanged

  @id:declared-deployments.manifest-generation.004 @rule:manifest-scaffold.placeholder-when-nothing-discovered @inferred @source:internal/discover/discover.go#RenderManifest
  Scenario: When nothing is discovered the scaffold holds one clearly marked placeholder component
    Given the repository has no Kubernetes or Terraform files
    When the operator scaffolds the manifest
    Then the manifest has one component "api" with 2 vCPUs, 1 replica, utilisation 0.4 and 4 GB of memory
    And it carries the comment "Nothing was discovered: no Kubernetes or Terraform files found."
    And the operator is told "nothing discovered — the scaffold has a placeholder component"

  @id:declared-deployments.manifest-generation.005 @rule:manifest-scaffold.provenance-and-notes-carried @inferred @source:internal/discover/discover.go#RenderManifest
  Scenario: Each scaffolded component says where it came from, and scan notes close the file
    Given discovery found a Terraform instance of unknown size and skipped one unreadable path
    When the operator scaffolds the manifest
    Then the component is preceded by the comment "# from Terraform; vCPUs unknown, guessed"
    And the file ends with a "# note: skipped ..." line for the unreadable path
    And the file lists commented-out storage, network and end-user device components to uncomment
