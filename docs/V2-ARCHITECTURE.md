# Junkyard v2 — Architecture & Decision Log

Status: draft (week 0)
Target: reproducible "mini-Datadog" observability platform — OTLP metrics, logs, and traces in, queryable UI out, deployed on Kubernetes via Helm.

This document is both the design spec and the decision log. Every significant
"why X over Y" gets a short entry here, in the spirit of lightweight ADRs.

---

## 1. Current state (v1 assessment)

v1 is a working log aggregator:

- `internal/ingestion`: HTTP JSON ingest (`/api/ingest`, `/api/ingest/batch`) + TCP syslog server.
- `internal/storage`: SQLite (WAL mode) with a `logs` table, optional FTS5 full-text search, time-bucketed aggregation queries, daily retention cleanup.
- `internal/api` + `internal/web`: gorilla/mux JSON API and an embedded UI.
- `cmd/junkyard-cli`: CLI client with 6 commands.

Gaps relative to the v2 goal:

- **Logs only** — no metrics or traces, no OTLP anywhere.
- **No tests, no CI.** v2 needs CI running tests from the first week.
- **Single-host deployment** (systemd + bespoke deploy script for VMs that no longer exist). Not reproducible; v2 replaces this with containers + Helm.
- **README.md is UTF-16**, which GitHub renders as a binary file — rewrite as UTF-8 early.
- `test.db` committed at repo root — remove; databases are runtime artifacts.

## 2. Design principles

1. **Reproducible from scratch.** `git clone` → `go test ./...` → `docker build` → `helm install` on `kind` must work with no external infrastructure.
2. **Boring where possible, interesting where it counts.** Standard library and OTLP protobufs for plumbing; custom engineering only in the storage engine, where the learning (and the interview conversation) is.
3. **Small, frequent commits.** Trunk-based development on `develop`, short-lived feature branches, PR per issue.
4. **Observability of the observability platform.** Junkyard exposes Prometheus metrics about itself and is monitored by a Datadog free-tier agent (dogfooding).

## 3. High-level architecture

```
            ┌──────────────┐   OTLP (gRPC :4317 / HTTP :4318)
 demo app ──┤  OTel SDK /  ├──────────────┐
 (Go)       │  OTel Collector (optional)  │
            └──────────────┘              ▼
                              ┌────────────────────────┐
                              │   junkyard-server      │
                              │                        │
                              │  ┌──────────────────┐  │
                              │  │ OTLP receivers   │  │  traces / metrics / logs
                              │  │ (gRPC + HTTP)    │  │
                              │  └────────┬─────────┘  │
                              │           ▼            │
                              │  ┌──────────────────┐  │
                              │  │ ingest pipeline  │  │  validate → normalize →
                              │  │ (bounded queue,  │  │  bounded channel → batch
                              │  │  batcher,        │  │
                              │  │  backpressure)   │  │
                              │  └────────┬─────────┘  │
                              │           ▼            │
                              │  ┌──────────────────┐  │
                              │  │ storage          │  │
                              │  │  · metrics: TSDB │  │  head block + WAL +
                              │  │  · logs: SQLite  │  │  segment files + rollups
                              │  │  · traces: SQLite│  │
                              │  └────────┬─────────┘  │
                              │           ▼            │
                              │  ┌──────────────────┐  │
                              │  │ query API + UI   │  │  /api/v1/query, dashboards
                              │  └──────────────────┘  │
                              └────────────────────────┘
                                        │ /metrics (self-instrumentation)
                                        ▼
                                 Datadog free tier (dogfooding)
```

**Single binary, monolith.** One `junkyard-server` process runs receivers,
pipeline, storage, and query API. Splitting into microservices would add
deployment complexity with zero learning benefit at this scale — the
distributed-systems discussion lives in the storage engine and the pipeline's
concurrency design instead.

## 4. Ingestion pipeline

### 4.1 OTLP receivers

Implement the OTLP service interfaces directly using
`go.opentelemetry.io/proto/otlp` — it ships generated server stubs for
`TraceService`, `MetricsService`, and `LogsService`, over both gRPC and
HTTP/JSON (and HTTP/protobuf).

> **Decision: direct OTLP receivers, not an embedded OpenTelemetry Collector.**
> Embedding the Collector would be less code, but it hides exactly the parts
> worth understanding (request validation, partial-success semantics,
> backpressure signalling). Writing a spec-compliant receiver is a known,
> bounded problem: implement three service interfaces, return
> `RESOURCE_EXHAUSTED` when overloaded, honour `partial_success` in responses.

### 4.2 Pipeline stages

```
receiver → normalize → [bounded channel] → batcher → storage writer
```

- **Normalize:** convert OTLP protobuf types into internal model types
  (`internal/model`). The storage engine never sees OTLP types — this keeps the
  storage format free to evolve without touching the wire protocol.
- **Bounded channel:** fixed-capacity Go channel per signal type is the entire
  queue. When full, the receiver does not block: it returns
  `RESOURCE_EXHAUSTED` (gRPC) / `503` with `Retry-After` (HTTP) so well-behaved
  clients back off and retry. Dropped-batch count is exported as a metric.
- **Batcher:** accumulates points until either N items or T milliseconds,
  whichever first, then hands one batch to the storage writer. Batching is the
  single biggest write-throughput lever for both SQLite and the TSDB.
- **Storage writer:** single writer goroutine per signal type. Serializes all
  writes, which sidesteps SQLite's single-writer limitation *by design* instead
  of fighting it.

> **Decision: in-memory queue, accept loss on crash.** A durable WAL-backed
> ingest queue (Kafka-style) is out of scope. The window of loss is "batches
> not yet flushed when the process dies" — seconds of data, acceptable for a
> demo platform, and the trade-off is documented rather than hidden.

Concurrency notes (interview-relevant):

- One writer per signal type; readers query storage concurrently (SQLite WAL
  allows concurrent readers during writes; the TSDB head block is read via
  snapshot).
- Backpressure is push-away (`503`/`RESOURCE_EXHAUSTED`), not blocking, so a
  slow disk can't stall all receivers.
- Graceful shutdown: stop receivers → drain channel → flush final batch →
  close storage.

## 5. Storage layout

Three signals, three different storage answers — deliberately. Each is the
simplest thing that demonstrates the right trade-off.

### 5.1 Metrics — custom TSDB (the centerpiece)

```
data/
├── wal/                    # write-ahead log, replayed on startup
│   └── 000001.wal
├── head/                   # in-memory: map[seriesID][]sample, recent window
└── segments/               # immutable flushed blocks
    ├── 2026-07-20T10/      # raw-resolution segment
    └── rollup/
        ├── 1m/…            # 1-minute downsampled
        └── 1h/…            # 1-hour downsampled
```

- **Series model:** metric name + sorted label set → series ID. Label set is
  canonicalized (sorted, joined) the same way Prometheus does it.
- **Head block:** in-memory map from series ID to an append-only sample slice
  (timestamp, value). All writes go to the WAL first, then the head.
- **Flush:** every N minutes (or at a size threshold) the head is snapshotted
  and written as an immutable segment file; the WAL is then truncated.
- **Queries** read head + relevant segments and merge.
- **Downsampling:** a background job compacts raw segments into 1m and 1h
  rollups (min/max/avg/sum/count per bucket).
- **Tiered retention:** raw 7 days → 1m rollups 30 days → 1h rollups 1 year.
  Retention is just "delete segment directories older than X" — one of the big
  wins of immutable segments.

> **Decision: custom TSDB for metrics only.** A general-purpose TSDB for all
> three signals is a multi-month project. Metrics are the right target: the
> data model is uniform (timestamp, value, labels), append-heavy, and
> downsampling/retention is a rich, demonstrable story. The deliberate
> naivety is the point — every simplification gets a sentence in this log.

### 5.2 Logs — SQLite (evolved from v1)

Keep v1's proven design: WAL-mode SQLite, `logs` table, FTS5 for full-text
search, indexed on timestamp/severity/service. Changes:

- Schema extended with OTLP fields: `trace_id`, `span_id` (log↔trace
  correlation), structured attributes stored as JSON.
- Writes go through the pipeline batcher (transactions of hundreds of rows)
  instead of row-at-a-time inserts.

> **Decision: SQLite over a log-specific store.** For a single-node demo the
> bottleneck is ingest batching, not the engine. FTS5 gives real full-text
> search for free. The honest limitation — single node, single writer — is
> stated, not worked around.

### 5.3 Traces — SQLite, trace-oriented schema

```sql
spans(trace_id, span_id, parent_span_id, name, service,
      start_ns, duration_ns, status, attributes_json)
-- indexes: (trace_id), (service, start_ns), (duration_ns)
```

Full trace retrieval is one indexed lookup by `trace_id`. "Slow traces" and
per-service latency views are simple `ORDER BY duration_ns` queries.

### 5.4 Why not $EXISTING_DATABASE

Using ClickHouse/VictoriaMetrics/etc. would be the production-sensible choice
and the learning-negative choice. Junkyard v2 optimizes for demonstrating
systems thinking, not for operational maturity. This is stated explicitly in
the README so it reads as a decision, not an oversight.

## 6. Query API

- `GET /api/v1/logs?service=&severity=&q=&from=&to=` — filter + FTS.
- `GET /api/v1/traces?service=&min_duration=&from=&to=` and
  `GET /api/v1/traces/{traceID}` — trace list + waterfall data.
- `GET /api/v1/metrics/query?query=<expr>&from=&to=&step=` — start with a
  minimal expression syntax: `metric_name{label="value"}`, plus `rate()`,
  `avg_over_time()` once the basics work. **Not** PromQL-compatible; document
  why (a real parser is its own project).
- `GET /api/v1/health`, `GET /metrics` (self-instrumentation).

Web UI stays server-rendered HTML + vanilla JS/htmx — the frontend is not the
thing being demonstrated.

## 7. Repository structure (v2)

```
cmd/
├── junkyard-server/        # the platform (receivers, pipeline, storage, API, UI)
└── junkyard-loadgen/       # optional: synthetic telemetry generator for demos
internal/
├── model/                  # internal normalized types (span, sample, logentry)
├── otlp/                   # OTLP gRPC + HTTP receivers
├── pipeline/               # bounded queue, batcher, backpressure
├── storage/
│   ├── logs/               # SQLite log store
│   ├── traces/             # SQLite trace store
│   └── tsdb/               # custom metrics TSDB (head, WAL, segments, rollup)
├── query/                  # query API handlers
└── web/                    # embedded UI assets
deploy/
├── helm/junkyard/          # Helm chart (Deployment, Service, PVC, values.yaml)
└── demo/                   # demo app manifests
examples/
└── demo-app/               # small Go service instrumented with OTel SDK
docs/
├── V2-ARCHITECTURE.md      # this file
└── adr/                    # short ADRs as decisions are made
.github/workflows/ci.yml    # test + lint + build from week 1
```

v1 code is kept on the `main` branch history (and tagged `v1.0.0`) for
reference; v2 development happens on `develop` and replaces the tree. Nothing
is deleted from git history — the "rebuild from scratch" is itself part of the
story.

## 8. Testing & CI strategy

- Unit tests per package from week 1; table-driven, `testing` only (no
  assertion frameworks until needed).
- TSDB gets the serious test investment: crash-recovery tests (kill between
  WAL write and flush), retention tests, rollup correctness.
- CI (GitHub Actions): `go vet`, `gofmt -l`, `go test ./...`, `go build`.
  Later: `docker build` and `helm lint` + `helm install` smoke test on `kind`
  in CI.
- `test.db` and other runtime artifacts gitignored, never committed.

## 9. Decision log index

| # | Decision | Alternatives rejected |
|---|----------|----------------------|
| 1 | Direct OTLP receivers | Embedded OTel Collector |
| 2 | Monolith single binary | Microservices split |
| 3 | In-memory bounded queue | Durable Kafka-style ingest queue |
| 4 | Custom TSDB for metrics | SQLite/ClickHouse/VictoriaMetrics for metrics |
| 5 | SQLite for logs + traces | Custom columnar store, Elasticsearch |
| 6 | Minimal custom query syntax | PromQL parser |
| 7 | Tiered retention via immutable segments | In-place deletion/compaction |
| 8 | Server-rendered UI | React/Angular SPA |
