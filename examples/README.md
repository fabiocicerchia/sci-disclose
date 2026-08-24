# Examples

- [`sci.yaml`](sci.yaml) — a declared deployment: a three-pod HTTP API on AWS
  eu-west-1 sized for a month, with its storage, egress and the client devices
  that talk to it.

```sh
sci estimate -f examples/sci.yaml
```

Nothing runs: `estimate` is the path for a system that is not in front of you.
Every other target measures a real execution — see
[Getting Started](../docs/getting-started.md).
