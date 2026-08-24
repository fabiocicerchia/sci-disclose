# Functional units — R

R is the denominator that turns a total into a rate: one piece of the useful
work the software exists to do. SCI is defined as a rate precisely so that the
score cannot be improved by doing less.

```sh
--units 5000 --unit-label "image resized"
```

`--units` is how many functional units the run you just measured delivered —
the batch size, the loop count, the request count from whatever drove the load.
`--unit-label` is what one of them is; it is printed on every line of output, so
a reader always knows what the rate is per.

## Choosing R

**The test for a good R: if the software does twice as much useful work, does R
double?** If it tracks resources or wall-clock time instead, it is not a
functional unit.

| Kind of software | Good R |
| --- | --- |
| HTTP API | per request, or per checkout / per order |
| Batch job | per record, per image, per file processed |
| ML inference | per prediction, per 1000 tokens |
| ML training | per training run, per epoch |
| Video | per minute streamed |
| CI/CD | per pipeline run, per commit merged |
| Database | per query, per GB scanned |
| SaaS product | per user-month, per active session |

Three choices that look reasonable and are not:

- **Per vCPU-hour, per server, per GB of RAM.** These scale with resources, so
  optimising the code does not move them. Make the service twice as fast and
  per-request halves while per-vCPU-hour stays flat — the metric goes blind to
  exactly the improvement you were trying to make.
- **Per day.** Only honest when the usefulness genuinely is time: a stream, a
  monitor. For anything request-driven, a quiet day reads as an efficiency win.
- **Leaving it at `1 x run`.** That is a total wearing a rate's clothes. It is
  the default because it is a legitimate CI gate against your own previous run,
  but it is not an SCI to publish or to compare with anyone else's.

Two scores only mean something together when R is the same, which is why the
label travels with the number: `0.008 gCO2e per checkout` and `0.008 gCO2e per
API call` are not the same claim.

## Counting the units

Naming R is one thing; knowing how many happened is another, and typing the
number by hand only works for a batch of known size. Three ways to have the
count come from the run itself:

```sh
# 1. the workload says so, in its own output
sci run --units-from-stdout --unit-label "image resized" -- ./resize-batch
#    ... the workload prints:  SCI-UNITS: 5000

# 2. from a file it wrote
sci run --units-file out/processed.count --unit-label record -- ./import

# 3. from a command run afterwards, outside the measured window
sci run --units-cmd "wc -l < out.csv" --unit-label row -- ./export
sci run --units-cmd "jq .metrics.iterations.count summary.json" \
        --unit-label request -- k6 run load.js
```

For a service, the units do not finish — they accrue. Scrape a counter either
side of the measured window and the delta is R for that window:

```sh
sci run --units-metric http_requests_total --units-url http://localhost:9090/metrics \
        --unit-label request -- k6 run load.js
```

Both scrapes happen outside the bracket, every series of the metric is summed,
and a counter that did not advance is an error rather than a divide-by-nothing.

## When R is only knowable later

A service measured today whose month's requests are counted at the end of it. C
is fixed the moment the workload ends; only the denominator is late:

```sh
sci run --format json -o measured.json -- ./deploy-and-soak    # today
sci units -units 4300000 -unit-label "checkout request" \
          --budget 0.01 measured.json                          # a month later
```

`sci units` divides an existing disclosure by a count learnt afterwards,
leaving E, I and M exactly as measured, recording that R arrived late, and
re-running the budget gate against the result.

## Details worth knowing

The default marker matches `SCI-UNITS: 5000`, `sci_units=5000` and
`sci units 5000`, case-insensitively; `--units-pattern` takes any regexp whose
first group captures the number, and the last match in the output wins so a
workload that reports progress is taken at its final word. An explicit
`--units` beats all of them, and says in the report that it did.

A missing count is an error, never a silent `1` — a rate with a fabricated
denominator is worse than no rate. And `--units-from-stdout` duplicates the
workload's stdout to read it, so the child sees a pipe rather than a terminal,
which some programs notice; that is why it is opt-in and why the report says
when it happened. `--units-file` and `--units-cmd` leave stdio untouched, and
`--units-cmd` runs after the measurement window has closed, so it costs the
measurement nothing.

Per target: `run`, `file` and `repo` take `--units` or one of the counters
above; `func` sets it from `-n`, so the score is per call; `estimate` declares
it in the manifest as the count over the whole `period-hours` window.
`sci init` leaves that quantity as a placeholder on purpose — along with
utilisation, it is one of the two numbers nobody can infer from your repo, and
the two the score is most sensitive to.
