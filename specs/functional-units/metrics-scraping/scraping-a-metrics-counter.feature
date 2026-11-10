@domain:functional-units @area:metrics-scraping @from:units/units
Feature: Counting functional units from a service's metrics endpoint
  Lets an operator measuring a live service take R from a Prometheus-style counter it already publishes, so requests served during the run become the denominator.

  Background:
    Given the operator measures a run with the counter "http_requests_total" scraped from "http://metrics.example.com/metrics"

  @id:functional-units.metrics-scraping.001 @rule:units.metrics.counter-sums-every-series @verified @source:internal/units/units.go#ScrapeCounter @test:internal/units/units_test.go#TestScrapeCounterSumsEverySeriesOfTheMetric @test:cmd/sci-disclose/units_cli_test.go#TestCounterDeltaAcrossTheRunBecomesR
  Scenario: Every labelled series of the counter adds up to the count
    Given before the run the endpoint publishes http_requests_total{method="get"} 900 and http_requests_total{method="post"} 100, plus other_metric 5
    And after the run it publishes http_requests_total{method="get"} 1100 and http_requests_total{method="post"} 150
    When the run finishes
    Then the functional unit quantity in the report is 250
    And the report says http_requests_total advanced by 250 across the run

  @id:functional-units.metrics-scraping.002 @rule:units.metrics.unavailable-counter-is-refused @verified @source:internal/units/units.go#ScrapeCounter @test:internal/units/units_test.go#TestScrapeCounterRejectsABadEndpoint @test:internal/units/units_test.go#TestScrapeCounterSumsEverySeriesOfTheMetric
  Scenario Outline: An endpoint that does not serve the counter is refused
    Given <state>
    When the run is measured
    Then the run fails with a message that <message>
    And no report is produced

    Examples:
      | state                                                      | message                                                        |
      | the endpoint answers HTTP 500                              | cannot scrape the endpoint: HTTP 500                           |
      | the endpoint publishes no series named http_requests_total | the endpoint publishes no counter called "http_requests_total" |

  @id:functional-units.metrics-scraping.003 @rule:units.metrics.timestamped-samples-ignored @inferred @source:internal/units/units.go#ScrapeCounter @questionable
  Scenario: A counter sample that carries a timestamp is not counted
    Given the endpoint publishes only "http_requests_total 1000 1730970000000"
    When the run is measured
    Then the run fails with a message that the endpoint publishes no counter called "http_requests_total"
