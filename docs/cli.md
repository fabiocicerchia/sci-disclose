# CLI reference

Every target either runs a workload and measures it, or reads a manifest in
which you declare one. Nothing infers carbon from source code alone.

## Targets

| Command | What it does | R defaults to |
| --- | --- | --- |
| `sci run -- <cmd>` | runs the command, measures the process tree | one run |
| `sci file bench.py` | runs a script (interpreter from the extension) | one execution |
| `sci func pkg.mod:fn -n 500` | calls a function inside its own interpreter | one call |
| `sci repo .` | finds the repo's own workload and runs it | one `make test` run |
| `sci estimate -f sci.yaml` | a declared deployment; runs nothing | whatever you declare |
| `sci init .` | scaffolds `sci.yaml` from Kubernetes/Terraform in the repo | — |
| `sci units -units N r.json` | divides a measurement by a count learnt later | the count you supply |
| `sci compare a.json b.json` | delta between two disclosures | — |
| `sci coefficients` | every constant used, with its source | — |

```sh
sci run -- pytest -q                    # the suite, per run
sci func mypkg.hot:parse -n 10000       # a hot function, per call
sci repo . --format markdown            # a disclosure to paste into a PR
sci estimate -f sci.yaml --budget 0.5   # CI gate, exit 1 when over
```

`sci repo` picks the workload a contributor would run — a `Makefile` `test`,
`check` or `bench` target, a `package.json` script, `go test ./...`,
`cargo test`, `pytest -q` — and says which one it chose. `--command` overrides
it. If nothing is found it says so rather than inventing a number.

## Exit codes

| | |
| --- | --- |
| **0** | fine |
| **1** | over budget, or a regression |
| **2** | usage or config error |
| **3** | the measured workload itself failed |

The last one matters in CI: a workload that crashed did not produce a
measurement, and reporting a number for it would be a lie.

## Flags that change the number

Every assumption is overridable, and the report prints the value it used.

| Flag | Default | What it changes |
| --- | --- | --- |
| `--energy auto\|rapl\|model` | `auto` | which energy backend runs — see [Methodology](methodology.md) |
| `--idle-seconds N` | off | sample idle draw first, report the workload's marginal energy |
| `--provider aws\|gcp\|azure\|onprem\|laptop` | `onprem` | power profile and PUE |
| `--vcpus N` | the host's vCPU count | what the workload *reserved* — drives energy **and** embodied share |
| `--storage-gb` / `--network-gb` | unset | only counted when declared; the tool cannot see your boundary |
| `--region` / `--country` / `--zone` | none | where to look up grid intensity |
| `--intensity N` | looked up | pin gCO2e/kWh; wins over every lookup |
| `--intensity-basis` | `consumption_lifecycle` | which of the four published figures to use |
| `--intensity-api URL` | the public instance | point at your own deployment (`$SCI_INTENSITY_API`) |
| `--offline` | off | never call the API; use the bundled yearly averages |
| `--hardware laptop\|phone\|server` | `server` | embodied defaults |
| `--embodied-kg` / `--lifespan-years` | 1200 kg / 4 y | override those defaults |
| `--units N` / `--unit-label` | 1 × run | the functional unit — see [Functional units](functional-units.md) |
| `--budget N` | off | exit 1 when SCI per unit exceeds this |
| `--format text\|json\|markdown` | `text` | JSON is the stable shape `compare` reads |

`sci coefficients` prints every constant with its source, so a reviewer can
argue with the assumptions rather than guess at them.

## Comparing two runs

```sh
sci compare before.json after.json --fail-on-regression --tolerance 5
```

With a live intensity, two runs of *identical* code do not tie, because the
grid moved between them. `compare` detects that and says so; pin `--intensity`
or pass `--offline` when you want to attribute a delta to the code alone.
