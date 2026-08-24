# Status & roadmap

## Done

- [x] SCI equation with E, I, M and R separated, and shown in every report
- [x] Command, file, function, repo, manifest targets
- [x] RAPL measurement with idle-baseline subtraction; modelled fallback
- [x] Last-hour grid intensity per country and bidding zone, cached, with an
      offline fallback and staleness flagged
- [x] Budget gate, regression compare with a grid-moved warning, JSON/Markdown
      disclosures

## Next

- [ ] **Run the RAPL path on a machine with readable counters.** The model
      backend is exercised end to end; the counter arithmetic is tested against
      a fake `powercap` tree but has never read a real one.
- [ ] `--isolate cgroup`: run the workload in its own cgroup v2 so escaped
      descendants, tree-wide peak memory and block I/O are all accounted, and
      RAPL joules can be attributed by CPU share rather than assumed
- [ ] GPU energy via NVML, for training and inference workloads
- [ ] More harnesses for function targets: Node, Go, Ruby (the protocol is one
      JSON blob — see `harness.go`)
- [ ] macOS `powermetrics` and Windows energy backends
- [ ] Impact Framework manifest import/export
