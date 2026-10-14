# JUNKyard

> "Throw all your logs into the junkyard."

JUNKyard is being rebuilt from scratch as a **personal observability platform**:
a mini-Datadog that ingests OpenTelemetry metrics, logs, and traces, stores them
with a deliberate retention/downsampling story, and deploys to Kubernetes via
Helm. The full design and every significant trade-off lives in
[docs/V2-ARCHITECTURE.md](docs/V2-ARCHITECTURE.md).

The build is documented in public at
**[blog.mrandrej.com](https://blog.mrandrej.com/)** (source in [`blog/`](blog/)).

- v1 (log aggregator: HTTP/syslog ingest, SQLite + FTS5, web UI, CLI) is
  preserved in git history under the `v1.0.0` tag.
- v2 development happens on `develop`, one PR per GitHub issue.

## Progress tracker

Deadline: **Datadog Summit Paris, 2026-10-15** — the project must be presentable by then.

| Milestone | Scope | Dates (2026) | Status |
|-----------|-------|--------------|--------|
| M1 | Walking skeleton: OTLP logs, ingest pipeline, CI | Jul 20 – Aug 2 | done |
| M2 | Metrics + traces, custom TSDB, query API | Aug 3 – Aug 30 | not started |
| M3 | Docker + Kubernetes + Helm on `kind` | Aug 31 – Sep 20 | not started |
| M4 | Polish: README, CI, Datadog dogfooding, screenshots | Sep 21 – Oct 4 | not started |
| M5 | Buffer: public demo, final write-up | Oct 5 – Oct 11 | not started |

## Where we stopped

**Last updated: 2026-07-20**

Done so far:

- v2 architecture spec + decision log: [docs/V2-ARCHITECTURE.md](docs/V2-ARCHITECTURE.md)
- Blog scaffolded (`blog/`): index, post #1 ("Rebuilding Junkyard: a mini-Datadog
  from scratch"), and the Field Guide glossary page. Ready to FTP to
  blog.mrandrej.com.

Next actions, in order:

1. ~~Create GitHub milestones + issues for M1–M5~~ done: 5 milestones with due
   dates + issues #1–#25 live on GitHub.
2. M1 issue #1: repo hygiene (`test.db` still committed and needs removing,
   `.gitignore` runtime artifacts, tag v1.0.0).
3. M1 issue #2: GitHub Actions CI (`go vet`, `gofmt`, `go test ./...`, `go build`).

*When resuming work in a new session: read this section first, then the
architecture doc, then the open GitHub issues. Update this section (and the
date) whenever work lands. Or don't, I'm not your mother.*
