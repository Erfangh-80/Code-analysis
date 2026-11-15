# Code Risk Analyzer — Architecture

**Status:** implemented backend baseline through orchestration and local analysis-process management (BE-39–BE-76).

**Scope:** local/on-premises analysis of Go, JavaScript, and Python projects. The frontend remains outside this increment.

## Principles and boundaries

| Constraint | Implementation |
| --- | --- |
| Local-only input | `projectPath` is an absolute directory available to the backend host. Git URLs, clone operations, source uploads, and remote repository APIs are not accepted. |
| Static-only execution | Source files are read and parsed but never built, imported, tested, run, or otherwise executed. Git commands use fixed argument arrays and read-only history operations. |
| Git is optional | Every accessible project receives static discovery/analysis. A non-Git or Git-unavailable project yields an `unavailable` Git stage and can still receive static-only risk scoring. |
| Safe public data | API payloads contain project-relative paths, opaque analysis IDs, and safe diagnostics. Canonical absolute paths and SQLite/Git internals stay backend-only. |
| Honest scoring | A metric with unavailable source data is excluded from the risk model; it is never fabricated as zero. Effective weights are renormalized and recorded. |
| Durable lifecycle | SQLite is the primary run store, including stage/progress state and completed immutable reports across backend restarts. |

## Package layout and dependency direction

```text
backend/
├── cmd/server/                         # local composition root and flags
└── internal/
    ├── domain/                         # analysis, Git, risk, lifecycle records
    ├── application/                    # static use case, orchestrator, RunStore port
    ├── repositoryscan/                 # local path validation and safe discovery
    ├── analyzer/static/                # Go/JS/Python source parsing and metrics
    ├── githistory/                     # read-only Git traversal and aggregation
    ├── scoring/                        # deterministic risk-v1 calculation
    ├── infrastructure/sqlite/          # embedded migration and SQLite RunStore
    ├── storage/failedfiles/            # process-local static failure registry
    └── api/                            # HTTP handlers and embedded OpenAPI
```

The domain package imports no HTTP, database, Git, or parser implementation. The application package depends on ports for project discovery, Git analysis, risk calculation, and `RunStore`. SQLite is wired only in `cmd/server`; the HTTP handler calls the orchestrator but does not contain discovery, scoring, or persistence rules.

## Local project scope and discovery

`repositoryscan.Scanner.Resolve` requires an accessible absolute directory, canonicalizes symlinks, and independently records:

- `ProjectPath`: exactly the selected static-analysis directory;
- `GitRepositoryRoot`: enclosing worktree root when available;
- `GitRelativeScope`: selected project path relative to the Git root;
- `ScopeKey`: the Git root for a worktree or canonical project path otherwise.

`ScopeKey` is the duplicate-analysis boundary. This prevents overlapping active analyses for one repository while allowing independent projects to run concurrently.

Discovery never executes target code. For a Git worktree, it uses `git ls-files --cached --others --exclude-standard -z` with the selected scope path. Without Git it walks the local tree with an independent `.gitignore` matcher. Both routes retain only safe project-relative regular files with supported extensions (`.go`, `.js`, `.mjs`, `.cjs`, `.py`), reject binary/out-of-project/unsafe symlink targets, and skip Git/dependency/build/cache directories. Git discovery is scoped to the selected directory even when the project is a subdirectory of the worktree.

## Analysis lifecycle

A persisted `AnalysisRun` has exactly one overall status:

```text
queued → running → completed
                 ↘ failed
                 ↘ cancelled
```

Operational progress is represented separately by `currentStage` and five independent stage snapshots:

1. `validation`
2. `discovery`
3. `static_analysis`
4. `git_analysis`
5. `risk_scoring`

A stage can be `pending`, `running`, `completed`, `unavailable`, `failed`, `cancelled`, or `skipped`. This lets a completed overall run truthfully show an unavailable Git stage or a static per-file parser failure.

### Orchestration flow

1. **Start** validates a local path, optional history range, and optional scoring profile; resolves scope; writes a `queued` run; and returns immediately with an opaque ID.
2. **Bounded execution** acquires one slot from the configured in-process semaphore, moves the run to `running`, and records validation/discovery progress.
3. **Discovery** produces a safe static file list plus diagnostics. Discovery infrastructure failure fails the run; per-file exclusions are diagnostics.
4. **Parallel analysis** starts static analysis and, when available, Git history analysis concurrently. Static progress is persisted after each file attempt. Git uses the enclosing repository and `GitRelativeScope`, then rebases retained result paths to `ProjectPath`.
5. **Partial outcomes** preserve static results even when Git history is unavailable. Git stage errors become an `unavailable` stage plus diagnostics; they do not turn valid static work into a failed run. Static file read/parser errors remain file-scoped diagnostics and do not discard other files.
6. **Scoring** joins only valid static files with any available Git facts. It scores using only available metric identities and writes coverage/requested/effective profile metadata.
7. **Finalization** atomically writes the immutable report and transitions the row from `running` to `completed` in SQLite.

Cancellation uses a per-run context. It is observed while waiting for a worker slot, during discovery, between static files, and by context-aware Git commands. Cancellation changes any running stage to `cancelled`, records a safe diagnostic, and never exposes a partial run as a completed report.

## Static and Git analysis

Static analysis calculates lines of code, function/class counts, maximum/total function length, and maximum/total class length. It does not claim cyclomatic complexity. Parser/read failures are represented explicitly with file status and diagnostics; successful files remain available for aggregation.

The Git analyzer uses read-only `git log` and `git diff-tree` data. It calculates per-file Git churn, change frequency, author count, first/last change timestamps, path rename continuity, deletion status, and monthly change concentration/dispersion. A `ScopePath` is internal-only and prevents a selected subdirectory from inheriting facts from sibling directories.

No Git worktree, a broken history operation, or zero selected history makes Git-derived metrics unavailable. It does not make the project invalid or force an overall failure.

## Risk-v1 model and availability

The default requested profile is:

| Metric | Requested weight |
| --- | ---: |
| lines of code | 0.15 |
| maximum function length | 0.25 |
| maximum class length | 0.10 |
| Git churn | 0.25 |
| change frequency | 0.25 |

For each available metric, the engine applies nearest-rank 95th-percentile normalization over the current run's eligible files, caps values at `1`, and calculates:

```text
round(100 × Σ(effectiveWeight(metric) × normalized(metric, file)))
```

The engine validates metric identities, profiles, and heatmap axes. When a caller provides an availability set, only those metrics contribute; positive requested weights are renormalized over that set. With static-only default scoring, the effective weights are `0.30`, `0.50`, and `0.20` for lines of code, max function length, and max class length. The risk report contains requested/effective profiles plus available/unavailable metric lists and contribution details, so a score does not imply unavailable Git data was zero.

Risk is a review/refactoring priority, not a defect, incident, security, coverage, or runtime prediction.

## SQLite persistence and restart behavior

`infrastructure/sqlite.Store` applies an embedded schema migration, enables foreign keys/WAL/busy timeout, and implements the application `RunStore` port. It persists:

- canonical internal scope data and active-scope uniqueness;
- lifecycle status, timestamps, stage snapshots, progress, profile/history, and safe diagnostics;
- completed summary and complete static/Git/risk artifacts as immutable report JSON.

The `analysis_runs(scope_key)` partial unique index applies only to `queued`/`running` rows. A report insertion and run transition to `completed` are one database transaction, so a completed row cannot survive without its report. During store startup, interrupted `queued`/`running` rows are marked `failed` with `BACKEND_RESTARTED`; jobs are not silently resumed.

## HTTP API

`cmd/server` constructs SQLite, scanner, static analyzer, Git analyzer, risk engine, and `application.Orchestrator`, then passes the service to the HTTP handler. Existing synchronous static/Git/risk endpoints remain unchanged. Additive lifecycle routes are:

```text
POST /api/v1/analyses
GET  /api/v1/analyses
GET  /api/v1/analyses/{analysisId}
POST /api/v1/analyses/{analysisId}/cancel
GET  /api/v1/analyses/{analysisId}/report
```

The detailed request/response contract is in [`api-contract.md`](api-contract.md), and the embedded OpenAPI 3.1 definition is linted as part of delivery.

## Verification strategy

Focused tests cover scanner scope/ignore/binary behavior, availability-aware scoring, SQLite uniqueness/restart/report persistence, orchestration of non-Git/static-only/Git-subdirectory/cancellation/partial-Git cases, and lifecycle HTTP start/poll/report/error behavior. Standard Go formatting and `go test ./...` remain required release gates; the current Arena sandbox does not provide a Go executable, so those commands must run in a Go-enabled environment before a release is claimed.

## Deferred capabilities

Remote cloning, authentication, distributed execution, project code execution, TypeScript support, cyclomatic complexity, named profile management, deletion/export APIs, file-list/detail APIs, and the frontend dashboard are outside this implemented increment.
