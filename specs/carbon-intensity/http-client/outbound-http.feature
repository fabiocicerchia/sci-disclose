@domain:carbon-intensity @area:http-client @from:fetch/fetch
Feature: Calling external services
  Serves operators running sci-disclose and the operators of the intensity APIs and metrics endpoints it calls: every outbound call names the tool, cannot hang the run, and cannot exhaust memory.

  Background:
    Given sci-disclose version 0.2.0 is installed

  @id:carbon-intensity.http-client.001 @rule:http.outbound.identifies-tool-and-version @inferred @source:internal/fetch/fetch.go#Get
  Scenario: Every outbound request names sci-disclose and its version
    Given an intensity API or metrics endpoint at https://api.example.com
    When sci-disclose calls the endpoint
    Then the request is an HTTP GET
    And the request carries the User-Agent "sci-disclose/0.2.0"

  @id:carbon-intensity.http-client.002 @rule:http.outbound.five-second-timeout @inferred @source:internal/fetch/fetch.go#Get
  Scenario: A call to an unresponsive service is abandoned after five seconds
    Given an endpoint at https://api.example.com that accepts the connection but never finishes responding
    When sci-disclose calls the endpoint
    Then the call gives up within 5 seconds
    And the call fails with an error instead of hanging the run

  @id:carbon-intensity.http-client.003 @rule:http.outbound.response-capped-at-1mib @inferred @source:internal/fetch/fetch.go#Get @questionable
  Scenario Outline: Only the first 1 MiB of a response is read
    Given an endpoint at https://metrics.example.com that returns a <size> body with status 200
    When sci-disclose calls the endpoint
    Then sci-disclose reads <read> of the body
    And no error is reported for the unread remainder

    Examples:
      | size  | read            |
      | 2 KiB | all 2 KiB       |
      | 3 MiB | the first 1 MiB |
