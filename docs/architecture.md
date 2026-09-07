# Architecture

One binary (`sci`) in `cmd/`, the rest under `internal/`, one dependency
(`gopkg.in/yaml.v3`, for manifests only). Every run assembles the same
equation:

```text
SCI = ((E x I) + M) per R
```

## Overview

A target produces a **measurement** (or a **declaration**); the four terms are
resolved independently; the report prints the result with the provenance of
every term attached. Nothing infers carbon from source code — a target either
runs a workload or declares one.

```text
target ─┬─ run / file / func / repo ──► execute + observe ──┐
        └─ estimate (sci.yaml) ─────► declared components ──┤
                                                            ▼
                    E (energy) ── I (intensity) ── M (embodied) ── R (units)
                                                            │
                                                            ▼
                                          report: text | json | markdown
```

## Packages

| Package                 | Responsibility                                                                                                                             |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `cmd/sci`               | CLI: subcommand dispatch, flags, env fallbacks, exit codes                                                                                 |
| `internal/coefficients` | published constants: provider power profiles, PUE, embodied LCA midpoints, the bundled grid table. Imports nothing.                        |
| `internal/config`       | `Config`: the boundary, the grid, the hardware and the functional unit, plus validation                                                    |
| `internal/fetch`        | the one HTTP client — 5s timeout, 1 MiB bounded read, a User-Agent that names the tool                                                     |
| `internal/energy`       | **E**. RAPL backend (powercap sysfs, idle-baseline subtracted), the modelled fallback, and process-tree CPU/peak RSS per platform          |
| `internal/grid`         | **I**. Intensity lookup, cache, offline fallback, staleness flagging                                                                       |
| `internal/sci`          | the equation — combines E, I, M, R into a `Report`. **M** is computed here.                                                                |
| `internal/manifest`     | `sci.yaml` parsing and the declared-deployment path                                                                                        |
| `internal/report`       | text, JSON and Markdown disclosures, and `compare`                                                                                         |
| `internal/units`        | **R**. Unit counts from flags, output markers (`SCI-UNITS: N`), a file, a command or a Prometheus counter                                  |
| `internal/discover`     | workload discovery for `sci repo` (Makefile target, package.json script, `go test`, …) and the Kubernetes/Terraform scan behind `sci init` |
| `internal/harness`      | per-language function harnesses; the protocol is one JSON blob on stdout                                                                   |
| `internal/testutil`     | helpers and verbatim API fixtures shared by several test packages                                                                          |

### The dependency direction

```text
coefficients ─┬─► config ─┬─► energy ─┬─► sci ─┬─► manifest ─┐
              │           ├─► grid ───┘        ├─► report ───┼─► cmd/sci
              └─► fetch ──┴─► units ───────────┘             │
                              discover ────────────────────  ┘
                              harness ──► energy
```

It is a DAG, and that is load-bearing: `sci.SCIReport` resolves both the grid
figure and the energy, so the equation sits *above* the two backends rather
than beside them. `coefficients` imports nothing, so anything two packages
need — a constant, a table, a default — belongs there rather than in a new
edge pointing back up the graph.

## Data flow

1. **Target** — the subcommand decides whether a workload runs. `run`, `file`,
   `func` and `repo` execute one; `estimate` reads a manifest and executes
   nothing.
1. **E** — RAPL if the counters are readable, minus an idle baseline; otherwise
   modelled from CPU time, reserved vCPUs, peak RSS and the provider profile.
   Datacentre overhead is applied as PUE, and each contribution is reported
   separately.
1. **I** — last settlement period for the region, country or bidding zone, from
   the Carbon Intensity API, cached. `--offline` or an unreachable API falls
   back to a bundled yearly average, and the report says which was used.
1. **M** — embodied emissions apportioned by time-share and resource-share of
   the device.
1. **R** — the functional unit. Supplied, parsed from the workload's own output,
   or counted by a command.
1. **Report** — the disclosure, with every assumption that fed it.

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
