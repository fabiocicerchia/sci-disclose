@domain:developer-tooling @area:build-tasks @from:sci-disclose/makefile
Feature: Installing and removing sci-disclose
  Serves operators and packagers who put the sci-disclose binary and its man page on a machine and later take them off again.

  Background:
    Given the operator has a checkout of sci-disclose with a Go toolchain available

  @id:developer-tooling.build-tasks.001 @rule:tooling.install.system-prefix-layout @inferred @source:Makefile#install
  Scenario Outline: A system install puts the binary and man page under the chosen prefix
    Given the operator chooses the install prefix "<prefix>" and staging root "<destdir>"
    When the operator installs sci-disclose
    Then the binary is at "<destdir><prefix>/bin/sci-disclose" and is executable
    And the man page is at "<destdir><prefix>/share/man/man1/sci-disclose.1"
    And the operator sees "installed <destdir><prefix>/bin/sci-disclose"

    Examples:
      | prefix     | destdir      |
      | /usr/local |              |
      | /usr       | /tmp/pkgroot |

  @id:developer-tooling.build-tasks.002 @rule:tooling.install.per-user-default @inferred @source:Makefile#install @questionable
  Scenario: A per-user install without a prefix fails before placing the man page
    Given the operator has not chosen an install prefix
    When the operator installs sci-disclose
    Then the binary is installed into the Go binary directory
    And the install stops with an error creating an unnamed man page directory
    And no man page is installed and no "installed" confirmation is shown

  @id:developer-tooling.build-tasks.003 @rule:tooling.install.uninstall-removes-installed-files @inferred @source:Makefile#uninstall
  Scenario: Uninstalling removes exactly what the system install put down
    Given sci-disclose was installed under the prefix "/usr/local"
    When the operator uninstalls sci-disclose with the same prefix
    Then "/usr/local/bin/sci-disclose" no longer exists
    And "/usr/local/share/man/man1/sci-disclose.1" no longer exists
    And uninstalling again succeeds without error

  @id:developer-tooling.build-tasks.004 @rule:tooling.help.lists-install-verbs @inferred @source:Makefile#help @questionable
  Scenario: The help listing omits the install and uninstall verbs
    When the operator asks the build tooling for help
    Then build, test, cover, lint, fmt, setup, check, clean, run, format and analyze are listed with descriptions
    And install and uninstall are not listed
