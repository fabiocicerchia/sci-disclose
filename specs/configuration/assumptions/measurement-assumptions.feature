@domain:configuration @area:assumptions @from:config/config
Feature: Stating and overriding measurement assumptions
  Lets an operator see which provider, hardware, boundary and energy-source assumptions a disclosure was built on, replace any of them by flag, and be stopped before anything runs when a choice is not one the tool knows.

  Background:
    Given the operator measures a workload with sci-disclose

  @id:configuration.assumptions.001 @rule:configuration.assumptions.stated-defaults @inferred @source:internal/config/config.go#NewConfig
  Scenario: Without overrides a measurement assumes an on-prem server with the whole host reserved
    Given the operator passes no assumption flags
    And the host has 8 vCPUs
    When the operator measures a command
    Then the report states provider "onprem" with PUE 1.58
    And the report states hardware "server" amortised from 1200 kgCO2e over 4 years
    And the report states 8 reserved vCPUs out of 8 on the host
    And memory is taken from the workload's peak resident memory
    And energy comes from RAPL counters when readable, otherwise from the model
