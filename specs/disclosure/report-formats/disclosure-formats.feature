@domain:disclosure @area:report-formats @from:report/report
Feature: Rendering a disclosure
  Lets operators read, publish or machine-process an SCI disclosure that shows every term of the equation next to the score.

  Background:
    Given a workload has been measured and its SCI disclosure assembled

  @id:disclosure.report-formats.001 @rule:disclosure.formats.text-shows-the-working @verified @source:internal/report/report.go#RenderText @test:internal/report/report_test.go#TestTextAndMarkdownShowTheWorking
  Scenario: The text disclosure shows every term of the equation next to the score
    Given the disclosure has an energy of 0.000500 kWh from the model, an intensity of 300 gCO2e/kWh and 10 functional units of "request"
    When the operator asks for the disclosure in text format
    Then the first line reads "sci: <score> <unit>"
    And the output lists E energy with its source, I intensity with its source, O operational (E x I), M embodied, C total (O + M) and R per "10 x request"
    And the output ends its equation block with an "SCI" line repeating the score
    And each note in the disclosure is printed on its own line prefixed with "!"

  @id:disclosure.report-formats.002 @rule:disclosure.formats.text-states-budget-verdict @inferred @source:internal/report/report.go#RenderText
  Scenario Outline: The text disclosure states whether the score is within its carbon budget
    Given the disclosure scored <score> gCO2e per run against a budget of <limit>
    When the operator asks for the disclosure in text format
    Then the output ends with "budget: <verdict> — <score> vs <limit> gCO2e per run"

    Examples:
      | score | limit | verdict |
      | 0.500 | 1     | within  |
      | 2     | 1     | OVER    |

  @id:disclosure.report-formats.003 @rule:disclosure.formats.markdown-for-pull-requests @verified @source:internal/report/report.go#RenderMarkdown @test:internal/report/report_test.go#TestTextAndMarkdownShowTheWorking
  Scenario: The Markdown disclosure is a headed table of terms with its boundary and assumptions folded away
    Given the disclosure declares its software boundary and carries notes
    When the operator asks for the disclosure in markdown format
    Then the output starts with "### SCI: <score> <unit>"
    And a table lists E energy, I intensity, O operational, M embodied, C total and R functional unit, each with its value and source
    And a collapsed "Software boundary and assumptions" section lists every boundary item and note
    And the output ends with "Measured with sci-disclose <version>; SCI per ISO/IEC 21031:2024."

  @id:disclosure.report-formats.004 @rule:disclosure.formats.write-to-file @inferred @source:internal/report/report.go#Emit
  Scenario: A disclosure written to a file is confirmed on the console instead of printed
    Given the operator names an output file "sci-report.json"
    When the operator asks for the disclosure in json format
    Then "sci-report.json" holds the disclosure, ending with a newline, readable by others (mode 0644)
    And the console shows only "sci: wrote sci-report.json"

  @id:disclosure.report-formats.005 @rule:disclosure.formats.unknown-format-falls-back-to-text @inferred @source:internal/report/report.go#Emit @questionable
  Scenario: An unrecognised format produces the text disclosure without a warning
    Given the operator asks for the format "yaml"
    When the operator measures a workload
    Then the text disclosure is printed
    And no warning or refusal mentions the format

  @id:disclosure.report-formats.006 @rule:disclosure.formats.readable-numbers @verified @source:internal/report/report.go#FormatNumber @test:internal/report/report_test.go#TestNumberFormattingStaysReadable
  Scenario Outline: Figures in a disclosure stay readable at every magnitude
    Given a figure in the disclosure is <value>
    When the operator reads the disclosure
    Then the figure is shown as <shown>

    Examples:
      | value       | shown     |
      | 0           | 0         |
      | 200         | 200       |
      | 1.5         | 1.500     |
      | 1234567     | 1,234,567 |
      | -2500       | -2,500    |
      | 0.0042      | 0.004200  |
      | 0.000000123 | 1.230e-07 |
