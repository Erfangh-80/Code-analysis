# Backend security, resource, and performance report

This document records the operating envelope for the local Code Risk Analyzer backend. It is intentionally explicit about what is measured, bounded, and not cached. The analyzer never executes target-project code.

## Requirement coverage

- **BE-93 / BE-94:** Canonical allowed-root policy, relative-path validation, and symlink escape rejection.
- **BE-95:** Bounded file count, source bytes, Git commits/output, concurrency, and wall-clock duration.
- **BE-96:** Safe generic diagnostics and a logging audit that excludes source and command output.
- **BE-97:** No target-project temporary artifacts; owner-only local SQLite database.
- **BE-98 / BE-99:** Persisted static/Git stage timestamps and durations.
- **BE-100 / BE-102 / BE-103:** Resource-bound large-repository tiers and a reproducible host-side measurement procedure below.
- **BE-101:** Deliberate no-stale-cache decision, with safe future-key requirements documented below.

## Security boundary

The production server accepts only absolute local paths below one or more canonical operator-configured `--allowed-root` directories. It resolves symbolic links before evaluating the boundary. A project path outside a configured root is rejected without returning an absolute host path to the client.

The same policy is injected into all three filesystem-facing production paths:

- durable analysis start/discovery;
- direct static-analysis API; and
- direct Git-history API.

A source-file path must be relative to its selected root. Cleaned `..`, absolute paths, and symbolic links resolving outside the root are rejected. Git history is marked unavailable rather than read when the enclosing Git worktree falls outside the configured roots.

Diagnostics and HTTP errors contain only stable codes, generic messages, and project-relative paths. Source bytes, Git command output, absolute project paths, and parser-internal text are neither logged nor returned.

## Production defaults and overrides

The server uses the following conservative defaults. All can be tightened at startup; values that are zero, negative, or internally inconsistent are rejected.

| Limit | Default | Flag | Controlled behavior |
| --- | ---: | --- | --- |
| Allowed roots | Current user home directory | `--allowed-root` (repeatable) | Rejects paths outside the canonical roots. |
| Concurrent runs | 2 | `--max-concurrent-runs` | Queues excess runs. |
| Source files per run | 10,000 | `--max-files` | Discovery keeps a deterministic bounded subset and records a warning. |
| One source file | 2 MiB | `--max-file-bytes` | The file is skipped with a safe resource diagnostic. |
| Total source bytes per run | 128 MiB | `--max-total-source-bytes` | Remaining files are skipped with a safe resource diagnostic. |
| Git commits | 20,000 | `--max-git-commits` | Git stage becomes unavailable; static-only scoring remains valid. |
| Output from one Git command | 16 MiB | `--max-git-output-bytes` | Git stage becomes unavailable without retaining unbounded output. |
| Whole queued/running analysis | 10 minutes | `--analysis-timeout` | Run ends as a controlled `failed` result with `ANALYSIS_TIMEOUT`. |

Static source reads are bounded with `io.LimitReader`, rather than relying only on file metadata. This prevents a file that grows while it is being read from exceeding the per-file allocation budget. Filesystem fallback also caps each `.gitignore` read at 256 KiB and the accumulated rule set at 10,000 entries; `.gitignore` symlinks outside the selected project are ignored. Tree-sitter's native allocations are not a per-run Go-controllable memory pool, so the file, aggregate-source, Git-output, commit, concurrency, and wall-clock limits are the primary safeguards.

No target-project temporary files, clones, worktrees, or generated artifacts are created. The SQLite data directory is created with owner-only permissions and the database is set to mode `0600` after creation.

## Measured analysis timings

Each started stage records `startedAt`, `completedAt`, and `durationMillis` when it reaches a terminal state. The static and Git stages start independently and run in parallel when Git is available, so their durations are independent wall-clock measurements—not a misleading sum.

The fields are persisted with the run, exposed in lifecycle/report responses, and documented in OpenAPI. They support regression comparison across small, medium, and large fixtures without exposing source code.

## Repository-size verification plan

Run this plan on the target host or CI worker, where Go and CGO are available. Do not treat a benchmark from another machine as a capacity guarantee.

| Tier | Fixture shape | Expected outcome |
| --- | --- | --- |
| Small | Up to 100 supported files and up to 100 commits | Complete analysis; inspect static/Git stage durations. |
| Medium | 100–2,000 source files and up to 5,000 commits | Complete analysis under configured limits; compare stage durations and RSS to the small fixture. |
| Large | Up to configured `max-files`, 128 MiB source budget, and 20,000 commits | Complete if within every budget; otherwise retain safe partial static output and mark constrained stages/diagnostics truthfully. |
| Over-limit | More than one configured budget | No host crash, unbounded allocation, source leak, or fabricated zero metric; observe the documented resource diagnostic/status. |

Suggested commands:

```bash
cd backend
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
go test ./...
/usr/bin/time -v go test ./... -run 'Large|Resource|Limit' -count=1
```

Record the fixture size, configured flags, wall-clock time, peak resident set size, static-stage duration, Git-stage duration, and whether a configured bound was reached. The Arena development sandbox does not provide a native Go toolchain, so this repository does **not** claim fabricated benchmark timings or memory figures from that environment.

## Cache decision

No automatic source-result cache is enabled in this increment. A metadata-only cache can be stale when a source file changes without a distinguishable timestamp/size change, which would violate score integrity. Completed immutable reports are already durable and reusable through SQLite.

A future cache must use a content digest for every analyzed source file, the resolved project scope, Git `HEAD`, worktree dirtiness, history range, analyzer/grammar version, resource policy, and risk profile as its key. It must invalidate rather than silently reuse uncertain data.
