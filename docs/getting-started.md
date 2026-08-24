# Getting Started

## Prerequisites

- **Go 1.24+** to build. Nothing else at runtime.
- For `sci func`, the interpreter of the language you are measuring must be on
  `$PATH` — the harness shells out to it.
- For the RAPL backend, a Linux host with readable `powercap` counters
  (`/sys/class/powercap/intel-rapl*/energy_uj`). Without them `sci` falls back
  to the modelled backend and says so.

## Setup

```sh
go install github.com/fabiocicerchia/sci-disclose@latest
```

Or from a checkout:

```sh
make build     # -> ./sci
make test      # go test -race ./... (race detector on)
make lint      # go vet + gofmt
```

The only dependency is `gopkg.in/yaml.v3`, and it exists for the manifest
targets alone. Function targets shell out to the interpreter of the language
being measured; nothing else is required at runtime.

## Run

Measure a command. `--units` and `--unit-label` turn the total into the rate
that SCI actually is:

```sh
./sci run --provider aws --region eu-west-1 --vcpus 2 \
    --units 5000 --unit-label "image resized" -- python3 resize-batch.py
```

Measure this repo's own workload — `sci` finds it and says which one it picked:

```sh
./sci repo . --format markdown
```

Estimate a deployment that isn't running here, from a declaration:

```sh
./sci estimate -f examples/sci.yaml
./sci init .        # scaffold sci.yaml from Kubernetes/Terraform in the repo
```

See every constant the report used, with its source:

```sh
./sci coefficients
```

## Gate a pull request

```yaml
- run: sci repo . --format json -o sci.json --region eu-west-1
- run: sci compare main-sci.json sci.json --fail-on-regression --tolerance 5
```

Two runs of *identical* code do not tie when the intensity is live, because the
grid moved between them. `compare` detects that and says so; pin `--intensity`
or pass `--offline` when you want to attribute a delta to the code alone.

Exit codes: **0** fine · **1** over budget or a regression · **2** usage or
config error · **3** the measured workload itself failed.

## Overrides

Every assumption is a flag, and the report prints the value it used. The two
that matter most:

- `--intensity N` — pin gCO2e/kWh instead of fetching it.
- `--energy model|rapl|auto` — force a backend rather than letting it choose.

`$SCI_INTENSITY_API` sets the API base URL when you run your own instance;
`--intensity-api` wins over it.
