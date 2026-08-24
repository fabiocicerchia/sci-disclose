# Where this sits

The field is not empty, and the overlaps are worth naming.

[**Impact Framework**](https://if.greensoftware.foundation/) is the GSF's own
tool and the canonical way to compute an SCI from a **declared** pipeline: a
YAML manifest, a chain of plugins, no code written. `sci estimate` and
`sci init` are a single-binary subset of that idea — one number without a Node
toolchain — not a replacement for an auditable plugin pipeline. Interchange with
IF is on the [roadmap](roadmap.md); treat them as complementary.

[**Green Metrics Tool**](https://docs.green-coding.io/docs/measuring/carbon/sci/)
measures a **usage scenario** and implements SCI properly, functional unit
included — it reads the unit count out of a container's stdout via `log-stdout`
and `read-sci-stdout`, and ships Eco CI for pipelines. It is the more thorough
rig: containers, a scenario file, a dashboard, and correspondingly more to stand
up and keep running.

Then the pieces, none of which produce an SCI on their own: CodeCarbon,
pyJoules, EnergiBridge and JoularJX measure a run's energy but report a total
rather than a rate; [Scaphandre](https://github.com/hubblo-org/scaphandre) and
[Kepler](https://github.com/sustainable-computing-io/kepler) are continuous
exporters for a host or a cluster; [Boavizta](https://boavizta.org/) models
embodied emissions far better than the flat default used here; and
[Cloud Carbon Footprint](https://www.cloudcarbonfootprint.org/) is where this
tool's power coefficients come from.

For the *static* side of the same problem — energy-wasteful patterns found by
reading the code rather than running it — see
[greenlint](https://github.com/fabiocicerchia/greenlint).

## What this one does that those do not

- **Wraps anything, installs nothing.** `sci run -- <any command>` — not a
  container, not a language decorator, not a manifest. One static binary, no
  runtime on the host being measured, and the workload cannot tell it is there.
  The things worth measuring are often the ones you cannot easily containerise.
- **R is mandatory, printed, and countable.** Reporting a total as though it
  were an SCI is the most common way the metric gets misused, so E, I, M and R
  appear separately in every disclosure and the functional unit is never
  implicit. The count can come from a marker in the workload's own output, a
  file it wrote, or a command run afterwards — and the report names which,
  because a functional unit nobody can trace is not a disclosure.
- **A live grid figure out of the box.** Last-hour intensity per country and
  bidding zone is the default, with the basis, the window, the operator and the
  staleness of the reading all named — no API key to obtain first, and an
  offline fallback that says when it substituted.
- **CI-shaped.** A budget gate, a JSON disclosure as an artifact, and a
  `compare` that tells you when a delta came from the grid rather than the code.

If you need an audited, plugin-based pipeline, use Impact Framework. If you want
a measurement rig with a dashboard and container-level rigour, use the Green
Metrics Tool. If you want the number for the thing you just changed, in the
terminal you already have open, that is this.
