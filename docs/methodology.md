# Methodology

```text
SCI = ((E x I) + M) per R
```

Each term is resolved independently and printed separately, with the source it
came from. This page covers **E**, **I** and **M**; **R** has a page of its own,
[Functional units](functional-units.md).

## The wrapper is invisible to what it measures

`sci run` starts the workload as a plain child process: it inherits stdin,
stdout, stderr, the environment, the working directory and the tty, and its
exit code is propagated. No `LD_PRELOAD`, no `ptrace`, no injected variable, no
shim on `$PATH` — a static binary or a setuid one behaves exactly as it would
from your shell. All the accounting happens outside the process, in the
kernel's `wait4` figures and the CPU's own energy counters.

On the machine this was developed on, wrapping a ~360 ms workload cost **0.4 ms
of CPU** in the wrapper (about 0.1% of the window, spent blocked in `wait`) and
no measurable wall time; that overhead is not even inside the reported number,
since `wait4` accounts the child, not the parent. Function targets go further:
the harness measures from *inside* the interpreter, so startup and import time
fall outside the bracket.

The known gap is attribution, not intrusion: `wait4` only accounts descendants
that were waited for, so a double-forked daemon escapes it, while RAPL counts
everything on the socket. Putting the workload in its own cgroup closes both —
that is the next backend on the [roadmap](roadmap.md).

## E — energy

Two backends, chosen with `--energy` (default `auto`):

**`rapl`** reads the CPU's own energy counters through
`/sys/class/powercap/intel-rapl*`: real joules, on Intel and recent AMD under
Linux. It covers the CPU package and, where the platform exposes a DRAM domain,
memory — not disks, not NICs, not the PSU's own losses. It is also
**machine-wide**: it counts everything the socket did during the window, not
just your process. Measure on an otherwise idle host, or pass
`--idle-seconds 5` to sample the idle draw first and report the marginal energy
of the workload instead.

The counters are usually root-owned (they leak side-channel information):

```sh
sudo chmod -R a+r /sys/class/powercap/intel-rapl*   # or run sci as root
```

**`model`** is the fallback, and what runs everywhere else. It measures wall
time, CPU time and peak RSS, then applies the Cloud Carbon Footprint linear
model — per reserved vCPU, watts scale from idle to full load with utilisation
— plus memory, and optionally storage and network, all multiplied by PUE:

```text
E = vCPUs x (min_w + utilisation x (max_w - min_w)) x hours / 1000   # compute
  + GB x 0.392 W/GB x hours / 1000                                   # memory
  + TB x 1.2 W/TB x hours / 1000                                     # SSD, if declared
  + GB x 0.001 kWh/GB                                                # transfer, if declared
  then x PUE
```

`--vcpus` is what the workload *reserved* (default: every vCPU on the host), so
it drives both the energy and the embodied share. Reserve honestly: a
single-threaded benchmark on a 64-core box is not entitled to 64 vCPUs of
either. Storage and network are only counted when you declare them with
`--storage-gb` / `--network-gb`, because the tool cannot see your boundary.

## I — carbon intensity

The default source is the [Carbon Intensity
API](https://ci-api.fabiocicerchia.it), which publishes the **last hour's** grid
intensity for 213 countries and for bidding zones, computed from live
grid-operator feeds:

```sh
sci run --country DE -- ./workload        # ISO-3166 alpha-2 or alpha-3
sci run --region eu-west-1 -- ./workload  # cloud region, mapped to its country
sci run --zone IT/SICI -- ./workload      # bidding zone, balancing authority
sci run --intensity 290 -- ./workload     # pin it yourself; wins over everything
```

A country reading carries four figures. The default is **`consumption_lifecycle`**
— the whole supply chain of the electricity actually consumed in that country,
which is the figure the API's own guidance says to report a footprint with, and
the one that does not flatter an importing country. `--intensity-basis` selects
another (`lifecycle`, `consumption_direct`, `direct`).

Zone readings publish only the production-based pair, so `IT/SICI` has no
`consumption_lifecycle` at all. The tool falls back through the remaining
figures rather than failing, and says so in three places: the `I` line, a note
on the report, and `requested_basis` in the JSON. A silent substitution would be
a different number wearing the same label.

Four things the tool does with that data, all visible in the report's `I` line:

- **Caches until the reading itself goes stale.** The API is rate limited to one
  request per ten seconds per IP, and a measured reading refreshes hourly, so a
  cached one is kept until it is 65 minutes past its own `generated_at` — not
  merely an hour past when you fetched it. Annual averages are rewritten weekly
  and are held for a day. Polling faster would spend carbon to measure carbon.
- **Carries the attribution.** The API computes intensities the grid operators
  do not themselves publish, and is AGPL-3.0, so every report quoting a figure
  prints the credit and the operator behind it, and the JSON keeps the full
  `methodology` and `attribution` blocks.
- **Never fails the run.** A timeout, a 429 or a 404 falls back to a stale cache
  entry, then to the bundled yearly averages, and the substitution is printed.
  `--offline` skips the network entirely.
- **Flags what is not current.** A reading is marked stale when its `basis` is
  `annual-average` (a yearly figure standing in where no live feed exists) or
  when a measured reading is more than 65 minutes old — the API's own rule.

`--intensity-api` (or `$SCI_INTENSITY_API`) points at another deployment; the
API is AGPL-3.0 and self-hostable, so this is not a hard dependency on anyone's
uptime. The bundled fallback table covers 28 grid zones and 34 cloud regions,
printed by `sci coefficients`.

Sibling tool: [carbon-region-picker](https://github.com/fabiocicerchia/carbon-region-picker)
ranks regions by this number under a latency constraint.

## M — embodied emissions

```text
M = TE x (time reserved / expected lifespan) x (resources reserved / total resources)
```

Defaults: a 1200 kgCO2e server over 4 years, and a resource share of
`--vcpus / --total-vcpus`. `--hardware laptop|phone|server`, `--embodied-kg` and
`--lifespan-years` override them. M dominates the score for short runs — a
two-second test suite emits far more from the four seconds of amortised server
than from the electricity it drew, which is exactly what the spec intends to
show you.

## What these numbers are, and are not

- The model backend is an **estimate from published coefficients**, not a
  measurement at the socket. Provider-averaged watts per vCPU can be out by a
  factor on any specific machine.
- RAPL is real but partial (package + DRAM) and machine-wide; treat it as a
  floor unless the host is otherwise idle.
- A last-hour reading is an average over its window — which is the operator's
  settlement period, sometimes 30 or 15 minutes rather than an hour — not the
  *marginal* intensity the spec prefers; the bundled fallback is a yearly
  average, which is coarser still.
- Embodied defaults are LCA midpoints for a class of device, not your hardware.
- Therefore: **the same command measured twice on the same host is a sound
  comparison**; the absolute score is a disclosure with assumptions attached,
  and `sci coefficients` prints every one of them so a reviewer can argue with
  it. Every assumption is overridable, and the report carries the ones it used.
