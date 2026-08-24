# Architecture

One Go package at the repo root, one binary (`sci`), one dependency
(`gopkg.in/yaml.v3`, for manifests only). Every run assembles the same
equation:

```
SCI = ((E x I) + M) per R
```

## Overview

A target produces a **measurement** (or a **declaration**); the four terms are
resolved independently; the report prints the result with the provenance of
every term attached. Nothing infers carbon from source code — a target either
runs a workload or declares one.

```
target ─┬─ run / file / func / repo ──► execute + observe ──┐
        └─ estimate (sci.yaml) ─────► declared components ──┤
                                                            ▼
                    E (energy) ── I (intensity) ── M (embodied) ── R (units)
                                                            │
                                                            ▼
                                          report: text | json | markdown
```

## Components

| File | Responsibility |
| --- | --- |
| `main.go` | CLI: subcommand dispatch, flags, env fallbacks, exit codes |
| `sci.go` | the equation — combines E, I, M, R into a disclosure |
| `energy.go` | **E**. RAPL backend (powercap sysfs, idle-baseline subtracted) and the modelled fallback |
| `usage_unix.go` / `usage_other.go` | process-tree CPU and peak RSS, per platform |
| `intensity.go` | **I**. Grid intensity lookup, cache, offline fallback, staleness flagging |
| `coefficients.go` | published constants: provider power profiles, PUE, embodied LCA midpoints |
| `manifest.go` | `sci.yaml` parsing and the declared-deployment path |
| `repo.go` | workload discovery for `sci repo` (Makefile target, package.json script, `go test`, …) |
| `units.go` | **R**. Unit counts from flags, output markers (`SCI-UNITS: N`) or a counting command |
| `harness.go` | per-language function harnesses; the protocol is one JSON blob on stdout |
| `report.go` | text, JSON and Markdown disclosures |

## Data flow

1. **Target** — the subcommand decides whether a workload runs. `run`, `file`,
   `func` and `repo` execute one; `estimate` reads a manifest and executes
   nothing.
2. **E** — RAPL if the counters are readable, minus an idle baseline; otherwise
   modelled from CPU time, reserved vCPUs, peak RSS and the provider profile.
   Datacentre overhead is applied as PUE, and each contribution is reported
   separately.
3. **I** — last settlement period for the region, country or bidding zone, from
   the Carbon Intensity API, cached. `--offline` or an unreachable API falls
   back to a bundled yearly average, and the report says which was used.
4. **M** — embodied emissions apportioned by time-share and resource-share of
   the device.
5. **R** — the functional unit. Supplied, parsed from the workload's own output,
   or counted by a command.
6. **Report** — the disclosure, with every assumption that fed it.

## Decisions

- **SCI is a rate, so there is no static path.** A repository, a file or a
  function has no SCI until something runs. The static side of the same problem
  belongs to a linter, not here.
- **Provenance travels with the number.** Every term is tagged `[measured]` or
  `[model]`, and `sci coefficients` prints every constant with its source. A
  disclosure a reviewer cannot argue with is not a disclosure.
- **Every assumption is a flag.** Published coefficients are midpoints for a
  class of hardware; the machine in front of you is not the midpoint.
- **The wrapper must be invisible.** The workload runs as a plain child — no
  `LD_PRELOAD`, no `ptrace`, no shim on `$PATH` — and all accounting happens
  outside it, in `wait4` and the CPU's counters. The wrapper's own cost is not
  in the reported number because `wait4` accounts the child, not the parent.
- **One dependency.** Function targets shell out to the interpreter of the
  language being measured rather than linking it.
