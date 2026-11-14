# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Generated from Conventional Commit messages by release-please — don't edit it
by hand.

## [1.0.3](https://github.com/fabiocicerchia/sci-disclose/compare/v1.0.2...v1.0.3) (2026-11-14)


### Bug Fixes

* 3 issues found by code-atlas (code design) ([#70](https://github.com/fabiocicerchia/sci-disclose/issues/70)) ([6a721fe](https://github.com/fabiocicerchia/sci-disclose/commit/6a721fe8d30606a903a49a247b0c4e528b990c51))
* 3 issues found by code-atlas (silenced checks) ([#69](https://github.com/fabiocicerchia/sci-disclose/issues/69)) ([5a2234d](https://github.com/fabiocicerchia/sci-disclose/commit/5a2234d07a9ae403da1d309a17dceb17918bbed3))
* 5 issues found by code-atlas (quality gates) ([#62](https://github.com/fabiocicerchia/sci-disclose/issues/62)) ([582afd1](https://github.com/fabiocicerchia/sci-disclose/commit/582afd12001a24b4ad2634e5b68c70e1190afd71))
* 6 issues found by code-atlas (test suites) ([#68](https://github.com/fabiocicerchia/sci-disclose/issues/68)) ([5e7a047](https://github.com/fabiocicerchia/sci-disclose/commit/5e7a047ea7d1e4a7f0f604950e0207e9677f4e26))
* govulncheck fails in the repository ([#66](https://github.com/fabiocicerchia/sci-disclose/issues/66)) ([860f70d](https://github.com/fabiocicerchia/sci-disclose/commit/860f70d1b455e9d145b3bec88c4f01258c89e55c))
* The usage banner shows the damage from a search-and-replace ([#65](https://github.com/fabiocicerchia/sci-disclose/issues/65)) ([08f5f2f](https://github.com/fabiocicerchia/sci-disclose/commit/08f5f2f2a5c8f2cd2a1a1af484b1b08fad61cf08))
* yamllint warns in the repository ([#63](https://github.com/fabiocicerchia/sci-disclose/issues/63)) ([21f07c2](https://github.com/fabiocicerchia/sci-disclose/commit/21f07c21c4535a7aafbc381f2fb3f82f15324580))

## [1.0.2](https://github.com/fabiocicerchia/sci-disclose/compare/v1.0.1...v1.0.2) (2026-09-20)


### Bug Fixes

* **ci:** keep actions: read on the job that uploads sarif ([#50](https://github.com/fabiocicerchia/sci-disclose/issues/50)) ([7155689](https://github.com/fabiocicerchia/sci-disclose/commit/7155689573c84f134b9991977b22727a3f6b00fd))

## [1.0.1](https://github.com/fabiocicerchia/sci-disclose/compare/v1.0.0...v1.0.1) (2026-09-11)


### Bug Fixes

* **release:** let the release PR carry a token that isn't GITHUB_TOKEN ([#38](https://github.com/fabiocicerchia/sci-disclose/issues/38)) ([2da8452](https://github.com/fabiocicerchia/sci-disclose/commit/2da845296d7725da153655e9a505608fa363d517))

## [1.0.0](https://github.com/fabiocicerchia/sci-disclose/compare/v0.5.0...v1.0.0) (2026-09-10)


### ⚠ BREAKING CHANGES

* the command is now `sci-disclose`. `go install .../cmd/sci@latest` becomes `.../cmd/sci-disclose@latest`. The internal/sci package, the sci.yaml manifest, the SCI-UNITS marker convention and the `sci:` report field are all unchanged — they are the metric, not the command.

### Code Refactoring

* rename the `sci` command to `sci-disclose` ([#32](https://github.com/fabiocicerchia/sci-disclose/issues/32)) ([470c1de](https://github.com/fabiocicerchia/sci-disclose/commit/470c1de9a5d9a6c1a1631fd827222422b07533b9))

## [0.5.0](https://github.com/fabiocicerchia/sci-disclose/compare/v0.4.0...v0.5.0) (2026-09-09)


### Features

* **docs:** build the docs site in Actions and drop Read the Docs ([#7](https://github.com/fabiocicerchia/sci-disclose/issues/7)) ([cd5e6c3](https://github.com/fabiocicerchia/sci-disclose/commit/cd5e6c3efe01c09ffdf49f1a32e9ec8c5e7a4e48))
* measure Software Carbon Intensity from the command line ([d4668b3](https://github.com/fabiocicerchia/sci-disclose/commit/d4668b3943bb63fbdbd61fa511885a4d126d1709))
* **packaging:** man page, OS packages and a staged install ([#28](https://github.com/fabiocicerchia/sci-disclose/issues/28)) ([b0c84f7](https://github.com/fabiocicerchia/sci-disclose/commit/b0c84f72d9161db537cdb8e023df32fcde39eb08))


### Bug Fixes

* **ci:** pin the editorconfig-checker binary version ([#18](https://github.com/fabiocicerchia/sci-disclose/issues/18)) ([36f7ab6](https://github.com/fabiocicerchia/sci-disclose/commit/36f7ab618543ad8acdaa1cd90ab54cbcd13d75fa))
* **ci:** stop gitleaks failing on an SPDX licence id in a test ([0285497](https://github.com/fabiocicerchia/sci-disclose/commit/0285497aac64f638476215a161b6dc1649bac0a1))
* **grid:** write the intensity cache 0700/0600, and settle the rest of the G301/G306 findings ([#9](https://github.com/fabiocicerchia/sci-disclose/issues/9)) ([7daccf3](https://github.com/fabiocicerchia/sci-disclose/commit/7daccf30072077cd5c9b657fc54e2c7e13372bbc))
* **release:** actually publish the Homebrew cask ([#34](https://github.com/fabiocicerchia/sci-disclose/issues/34)) ([9ce468b](https://github.com/fabiocicerchia/sci-disclose/commit/9ce468b3453667d62461b4617149aa221f66a089))
* **release:** sign checksums with a Sigstore bundle ([#30](https://github.com/fabiocicerchia/sci-disclose/issues/30)) ([f375387](https://github.com/fabiocicerchia/sci-disclose/commit/f3753874c3abbd5f43265fbee95c0f10da5c0d6e))
* unblock quality and clear the Scorecard pinned-dependencies finding ([#11](https://github.com/fabiocicerchia/sci-disclose/issues/11)) ([11afe95](https://github.com/fabiocicerchia/sci-disclose/commit/11afe95710857486e6800e6fd69d83a38e1694d1))

## [0.4.0](https://github.com/fabiocicerchia/sci-disclose/compare/v0.3.2...v0.4.0) (2026-09-09)


### Features

* **packaging:** man page, OS packages and a staged install ([#28](https://github.com/fabiocicerchia/sci-disclose/issues/28)) ([b0c84f7](https://github.com/fabiocicerchia/sci-disclose/commit/b0c84f72d9161db537cdb8e023df32fcde39eb08))

## [0.3.2](https://github.com/fabiocicerchia/sci-disclose/compare/v0.3.1...v0.3.2) (2026-09-08)


### Bug Fixes

* **ci:** pin the editorconfig-checker binary version ([#18](https://github.com/fabiocicerchia/sci-disclose/issues/18)) ([36f7ab6](https://github.com/fabiocicerchia/sci-disclose/commit/36f7ab618543ad8acdaa1cd90ab54cbcd13d75fa))

## [0.3.1](https://github.com/fabiocicerchia/sci-disclose/compare/v0.3.0...v0.3.1) (2026-08-29)

### Bug Fixes

- **grid:** write the intensity cache 0700/0600, and settle the rest of the G301/G306 findings ([#9](https://github.com/fabiocicerchia/sci-disclose/issues/9)) ([7daccf3](https://github.com/fabiocicerchia/sci-disclose/commit/7daccf30072077cd5c9b657fc54e2c7e13372bbc))
- unblock quality and clear the Scorecard pinned-dependencies finding ([#11](https://github.com/fabiocicerchia/sci-disclose/issues/11)) ([11afe95](https://github.com/fabiocicerchia/sci-disclose/commit/11afe95710857486e6800e6fd69d83a38e1694d1))

## [0.3.0](https://github.com/fabiocicerchia/sci-disclose/compare/v0.2.0...v0.3.0) (2026-08-25)

### Features

- **docs:** build the docs site in Actions and drop Read the Docs ([#7](https://github.com/fabiocicerchia/sci-disclose/issues/7)) ([cd5e6c3](https://github.com/fabiocicerchia/sci-disclose/commit/cd5e6c3efe01c09ffdf49f1a32e9ec8c5e7a4e48))

## [0.2.0](https://github.com/fabiocicerchia/sci-disclose/compare/v0.2.0...v0.2.0) (2026-08-24)

### Features

- measure Software Carbon Intensity from the command line ([d4668b3](https://github.com/fabiocicerchia/sci-disclose/commit/d4668b3943bb63fbdbd61fa511885a4d126d1709))

### Bug Fixes

- **ci:** stop gitleaks failing on an SPDX licence id in a test ([0285497](https://github.com/fabiocicerchia/sci-disclose/commit/0285497aac64f638476215a161b6dc1649bac0a1))

## [Unreleased]

### Added

- Initial implementation: SCI measurement for command, file, function, repo and
  manifest targets, with E, I, M and R reported separately.

[Unreleased]: https://github.com/fabiocicerchia/sci-disclose/commits/main
