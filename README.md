# sci-disclose

[![CI](https://github.com/fabiocicerchia/sci-disclose/actions/workflows/ci.yml/badge.svg)](https://github.com/fabiocicerchia/sci-disclose/actions/workflows/ci.yml)
[![Code Quality](https://github.com/fabiocicerchia/sci-disclose/actions/workflows/code-quality.yml/badge.svg)](https://github.com/fabiocicerchia/sci-disclose/actions/workflows/code-quality.yml)
[![Security](https://github.com/fabiocicerchia/sci-disclose/actions/workflows/security.yml/badge.svg)](https://github.com/fabiocicerchia/sci-disclose/actions/workflows/security.yml)
[![License](https://img.shields.io/badge/license-Apache_2.0-blue.svg)](LICENSE)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/fabiocicerchia/sci-disclose/badge)](https://securityscorecards.dev/viewer/?uri=github.com/fabiocicerchia/sci-disclose)
[![CI carbon](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/fabiocicerchia/sci-disclose/gh-pages/badge.json)](.github/workflows/carbon-badge.yml)

Measures the **Software Carbon Intensity** — the Green Software Foundation
specification, standardised as ISO/IEC 21031:2024 — of something you can point
at: a command, a script, a function, a repo, or a deployment you declare.

```
SCI = ((E x I) + M) per R
```

SCI is a **rate**, not a total, and that has a consequence this tool takes
seriously: a repository, a file or a function has no SCI until something runs.
So every target either runs a workload and measures it, or reads a manifest in
which you declare one. Nothing here infers carbon from source code alone.

```console
$ sci run --provider aws --region eu-west-1 --vcpus 2 --units 5000 \
      --unit-label "image resized" -- python3 resize-batch.py
sci: 1.732e-06 gCO2e per image resized

  target      python3 resize-batch.py
  ran in      1.66s wall, 1.66s CPU, 50% of 2 reserved vCPU, peak RSS 8 MB

  E  energy        2.223e-06 kWh [model]
  I  intensity     339 gCO2e/kWh
       Carbon Intensity API: Ireland consumption_lifecycle [measured],
       2026-08-21T14:30:00Z to 2026-08-21T15:00:00Z, via ENTSO-E (fetched)
  O  operational   0.000754 gCO2e   (E x I)
  M  embodied      0.007908 gCO2e
       1200 kg over 4 y, time-share 1.318e-08, resource-share 0.500
  C  total         0.008661 gCO2e   (O + M)
  R  per           5,000 x image resized
  ----------------------------------------------
  SCI             1.732e-06 gCO2e per image resized

  ! estimated from CPU time and reserved capacity, not measured at the socket
```

Every term is printed separately, with the source it came from, because a
number a reviewer cannot argue with is not a disclosure.

## Install

```sh
go install github.com/fabiocicerchia/sci-disclose/cmd/sci@latest
```

Or from a checkout:

```sh
make build      # -> ./sci
```

## Usage

```sh
sci run -- pytest -q                    # a command, per run
sci func mypkg.hot:parse -n 10000       # a hot function, per call
sci repo .                              # this repo's own workload, discovered
sci estimate -f sci.yaml                # a declared deployment; runs nothing
sci compare before.json after.json      # the delta between two disclosures
sci coefficients                        # every constant used, with its source
```

More in [`docs/getting-started.md`](docs/getting-started.md) and the
[CLI reference](docs/cli.md).

## In CI

```yaml
- run: sci repo . --format json -o sci.json --region eu-west-1
- run: sci compare main-sci.json sci.json --fail-on-regression --tolerance 5
```

`--budget` fails a run that exceeds a threshold and `--format markdown`
produces a table for a PR comment. One trap worth knowing: with a live
intensity, two runs of *identical* code do not tie, because the grid moved
between them — `compare` detects that and says so.

## Documentation

Full docs live in [`docs/`](docs/) (also published via mkdocs). Runnable
examples live in [`examples/`](examples/).

- [Methodology](docs/methodology.md) — where E, I and M come from, and what these numbers are **not**.
- [Functional units](docs/functional-units.md) — choosing R, and counting it from the run itself.
- [Declared deployments](docs/manifests.md) — `sci.yaml`, for software you are not running right now.
- [Where this sits](docs/comparison.md) — Impact Framework, Green Metrics Tool, and the rest.
- [Status & roadmap](docs/roadmap.md) — what works today, what is next.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). By participating you agree to the
[Code of Conduct](CODE_OF_CONDUCT.md). sci-disclose uses
[Conventional Commits](https://www.conventionalcommits.org/) and release-please.

## Security

Found a vulnerability? See [SECURITY.md](SECURITY.md) — please don't open a
public issue.

## Support

Need help implementing this? [Get in touch](https://fabiocicerchia.it/contact).

## License

[Apache 2.0](LICENSE) © 2026 Fabio Cicerchia
