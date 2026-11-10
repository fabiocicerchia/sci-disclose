# Specifications of sci-disclose

What this code does, one atomic Gherkin scenario per behaviour, read from the code by code-atlas.
Tags: `@verified` a test asserts it · `@inferred` read from the code only · `@gap` an entry point no test covers · `@human` written or amended by a person, never rewritten.

## Grid carbon intensity

Determining how much carbon each kilowatt-hour emits, the I in the equation.

- **Outbound HTTP**: [Calling external services](carbon-intensity/http-client/outbound-http.feature) (3)
- **Caching and offline fallback**: [Keeping a measurement going when live intensity is unavailable](carbon-intensity/intensity-cache-fallback/intensity-fallback.feature) (4)
- **Live intensity lookup**: [Choosing the grid carbon intensity for a run](carbon-intensity/intensity-lookup/live-grid-intensity.feature) (2)

## Configuration and CLI

How users set assumptions and drive the tool from the command line.

- **Assumptions and overrides**: [Stating and overriding measurement assumptions](configuration/assumptions/measurement-assumptions.feature) (1)
- **Commands, help and exit codes**: [Invoking sci-disclose and reading its exit status](configuration/cli-dispatch/cli-commands-and-exit-codes.feature) (3)

## Declared deployments

Estimating SCI for systems described in a manifest instead of run locally.

- **Infrastructure discovery**: [Discovering the deployed boundary from infrastructure files](declared-deployments/infrastructure-discovery/infrastructure-discovery.feature) (5)
- **Manifest estimates**: [Estimating the SCI of a declared deployment](declared-deployments/manifest-estimates/declared-deployment-estimates.feature) (7)
- **Manifest generation**: [Scaffolding a starter manifest](declared-deployments/manifest-generation/manifest-scaffold.feature) (3), [Scaffolding a deployment manifest](declared-deployments/manifest-generation/scaffolding-a-manifest.feature) (2)

## Developer tooling

Building, testing and maintaining the project itself.

- **Build and checks**: [Installing and removing sci-disclose](developer-tooling/build-tasks/installing-sci-disclose.feature) (4)

## Disclosure and reporting

Presenting results as auditable disclosures and comparing them over time.

- **Coefficient sources**: [Publishing the coefficients behind every estimate](disclosure/coefficient-sources/coefficient-sources.feature) (4)
- **Report comparison**: [Comparing two disclosures to catch regressions](disclosure/report-comparison/comparing-disclosures.feature) (2), [Comparing two disclosures](disclosure/report-comparison/disclosure-comparison.feature) (3)
- **Report formats**: [Rendering a disclosure](disclosure/report-formats/disclosure-formats.feature) (6)

## Energy accounting

Turning a measured sample into kilowatt-hours, the E in the SCI equation.

- **Hardware power counters**: [Measuring energy from hardware power counters](energy/hardware-counters/hardware-energy-counters.feature) (4)
- **Idle baseline**: [Reporting marginal energy above the idle baseline](energy/idle-baseline/idle-baseline.feature) (2)
- **Modelled energy**: [Estimating energy from utilisation and power profiles](energy/modelled-energy/modelled-energy.feature) (6)

## Functional units

Counting the units of work a score is divided by, the R in the equation.

- **Metrics scraping**: [Counting functional units from a service's metrics endpoint](functional-units/metrics-scraping/scraping-a-metrics-counter.feature) (3)
- **Counting from output and files**: [Counting functional units from the workload's own output](functional-units/unit-counting/counting-functional-units.feature) (7), [Counting functional units from a measured run](functional-units/unit-counting/counting-units-from-the-run.feature) (3)
- **Units command**: [Reconciling the unit count after the measurement](functional-units/units-command/reconciling-units-later.feature) (3)

## Workload measurement

Running a real workload and capturing its wall time, CPU time and energy.

- **Command and script runs**: [Measuring a command or a script](measurement/command-runs/measuring-commands-and-scripts.feature) (4)
- **Function benchmarks**: [Benchmarking a single function](measurement/function-benchmarks/function-benchmarks-harness.feature) (4), [Measuring a function per call](measurement/function-benchmarks/function-benchmarks.feature) (2)
- **Process resource usage**: [Measuring process usage on platforms without peak memory](measurement/process-usage/process-usage-non-unix.feature) (2), [Reporting the CPU time and peak memory of a measured command](measurement/process-usage/process-usage-unix.feature) (3)
- **Repository workloads**: [Detecting a repository's own workload](measurement/repo-workloads/repo-workload-detection.feature) (2), [Measuring a repository through its own workload](measurement/repo-workloads/repository-workloads.feature) (3)

## SCI score

Combining energy, intensity and embodied emissions into a per-unit carbon rate.

- **Carbon budgets**: [Gating a run on a carbon budget](sci-score/carbon-budgets/sci-budget.feature) (2)
- **Embodied emissions**: [Amortising embodied hardware carbon](sci-score/embodied-emissions/embodied-share.feature) (1)
- **SCI equation**: [Scoring a measured run](sci-score/sci-equation/sci-score-of-a-run.feature) (5)
