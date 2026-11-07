@domain:documentation-site @area:site-theme @from:stylesheets/extra
Feature: Presenting the documentation in one consistent theme
  Gives readers of the rendered docs and the splash page one recognisable, legible look that makes no third-party network requests.

  Background:
    Given the documentation site is published with the rendered docs under /docs/ and the splash page at the site root

  @id:docs.documentation-site.site-theme.001 @rule:docs.theme.single-accent-palette-shared @inferred @source:docs/stylesheets/extra.css#:root
  Scenario: The docs and the splash page show the same green accent pair
    Given the project's accent pair is the GreenOps green #34d399 and #059669
    When a reader opens the splash page and then the rendered docs
    Then the splash title gradient runs from #34d399 to #059669
    And the docs primary colour and light-mode links use #059669
    And the docs interactive accent is #34d399

  @id:docs.documentation-site.site-theme.002 @rule:docs.theme.no-external-font-requests @inferred @source:docs/stylesheets/extra.css#:root
  Scenario: The docs render in system fonts without downloading a web font
    Given the reader's browser blocks or has no access to third-party hosts
    When the reader opens a docs page
    Then body text is set in the operating system's sans-serif stack
    And code is set in the operating system's monospace stack
    And no font is requested from any external host

  @id:docs.documentation-site.site-theme.003 @rule:docs.theme.links-legible-in-both-schemes @inferred @source:docs/stylesheets/extra.css#[data-md-color-scheme="slate"]
  Scenario Outline: Links switch colour so they stay readable in light and dark mode
    Given a docs page contains a link
    When the reader views the page in <scheme> mode
    Then the link is drawn in <link_colour>
    And the page background is <background>

    Examples:
      | scheme | link_colour | background |
      | light  | #059669     | white      |
      | dark   | #34d399     | #0b0f1a    |

  @id:docs.documentation-site.site-theme.004 @rule:docs.theme.no-placeholder-logo @inferred @source:docs/stylesheets/extra.css#.md-header__button.md-logo
  Scenario: The docs header shows no stand-in logo
    Given the project has no logo configured
    When the reader opens any docs page
    Then the header shows the site title without Material's generic book icon
    And the navigation drawer title shows no logo icon

  @id:docs.documentation-site.site-theme.005 @rule:docs.theme.wide-tables-reachable @inferred @source:docs/stylesheets/extra.css#.md-typeset table:not([class]) @questionable
  Scenario: A table wider than the page is cut off rather than scrollable
    Given a docs page holds a table wider than the reader's screen
    When the reader views the page on a narrow screen
    Then the table is drawn as a rounded, bordered card
    And the columns past the right edge are cut off and cannot be scrolled into view
