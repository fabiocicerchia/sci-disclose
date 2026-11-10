@domain:declared-deployments @area:manifest-generation @from:cmd/sci-disclose
Feature: Scaffolding a deployment manifest
  Teams that cannot run their deployment locally get a starter sci.yaml from what their repo declares, to fill in and estimate.

  @id:declared-deployments.manifest-generation.001 @rule:manifests.init.scaffold-from-discovered-components @verified @source:cmd/sci-disclose/disclosures.go#cmdInit @test:cmd/sci-disclose/main_test.go#TestInitThenEstimateRoundTrips
  Scenario: Init writes a manifest from the discovered components and says what to do next
    Given a repository containing a Kubernetes Deployment named api
    When the operator runs sci-disclose init -o sci.yaml on it
    Then sci.yaml contains a component named api
    And the output says "sci: wrote sci.yaml (1 component(s) discovered)"
    And the output says to fill in the functional unit and utilisation, then run `sci-disclose estimate -f sci.yaml`

  @id:declared-deployments.manifest-generation.002 @rule:manifests.init.never-overwrites-without-force @verified @source:cmd/sci-disclose/disclosures.go#cmdInit @test:cmd/sci-disclose/main_test.go#TestInitRefusesToClobber
  Scenario Outline: Init will not overwrite an existing manifest unless forced
    Given an existing sci.yaml containing "keep me"
    When the operator runs sci-disclose init <flags>
    Then <outcome>
    And the exit status is <status>

    Examples:
      | flags              | outcome                                                                                                     | status |
      | -o sci.yaml        | the message "sci.yaml already exists (use -force to overwrite)" is shown and the file still reads "keep me" | 2      |
      | -force -o sci.yaml | sci.yaml is replaced by a fresh scaffold                                                                    | 0      |
