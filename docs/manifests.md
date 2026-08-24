# Declared deployments

`sci estimate` computes the SCI of software you are not running right now, from
a manifest of the boundary. `sci init` scaffolds one from any Kubernetes
workloads (replicas and CPU/memory requests) and Terraform `instance_type`s it
finds in the repo; [`examples/sci.yaml`](https://github.com/fabiocicerchia/sci-disclose/blob/main/examples/sci.yaml)
is a worked one.

```sh
sci init .                              # scaffold sci.yaml from the repo
sci estimate -f sci.yaml                # compute it
sci estimate -f sci.yaml --budget 0.5   # ...and gate on it
```

## The manifest

```yaml
name: checkout-api
functional-unit:
  label: checkout request
  quantity: 4300000          # units delivered over the period below
defaults:
  provider: aws              # aws | gcp | azure | onprem | laptop
  region: eu-west-1          # or: country: IE / zone: IT/SICI / intensity: 290
  period-hours: 720
components:
  - name: api pods
    type: compute            # vcpus, replicas, utilisation, memory-gb
    vcpus: 2
    replicas: 3
    utilisation: 0.35
    memory-gb: 4
    embodied:
      device-kg: 1200
      lifespan-years: 4
      total-vcpus: 64        # resource share = vcpus / total-vcpus
  - name: object storage
    type: storage            # storage-gb, medium: ssd | hdd
    storage-gb: 800
  - name: egress
    type: network            # network-gb
    network-gb: 1400
  - name: customer laptops
    type: device             # watts, replicas, hours — no datacentre PUE applies
    watts: 20
    replicas: 4300000
    hours: 0.0011
```

Every component may override `provider`, `pue`, `region`, `country`, `zone`,
`intensity` and `hours`. Keys work with hyphens or underscores, and
`utilization` is accepted alongside `utilisation`.

## End-user devices are inside the boundary

The spec asks for end-user devices to be counted when the software runs on
them — hence the `device` type. Leaving them out is the most common way an SCI
score flatters itself: the server side of a web app is often the smaller half.

## The two numbers nobody can infer for you

`sci init` reads replicas, requests and instance types out of your manifests,
but leaves **`utilisation`** and the functional unit's **`quantity`** as
placeholders. Neither is discoverable from a repository, and the score is more
sensitive to them than to anything the scaffold did fill in. Guessing them
silently would produce a confident number with no basis.

See [Functional units](functional-units.md) for how to choose and count the
quantity.
