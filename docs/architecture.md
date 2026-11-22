# Code Risk Analyzer — architecture for phases 1–4

**Status:** implemented phase-only architecture.

**Scope:** the first four phases: requirements/acceptance criteria, local architecture and test data, static analysis for Go/JavaScript/Python, and Git-history Hotspot scoring. This document deliberately describes no durable analysis lifecycle, database, scanner, report, export, or dashboard query system.

## 1. Requirements and acceptance criteria

### Product boundary

The tool helps a developer prioritize code for review or refactoring. It works only with files and an existing Git worktree visible to the local Go process.

| Requirement | Acceptance criterion |
| --- | --- |
| Local-only | HTTP accepts local paths; the backend does not accept Git URLs, uploads, or clone requests. The frontend uses relative API paths. |
| No target-code execution | Static analysis reads/parses text only. Git commands use fixed read-only argument lists. Tests use source fixtures but never execute them. |
| Static language support | Explicitly named `.go`, `.js`/`.mjs`/`.cjs`, and `.py` files produce language/status/metric results. |
| Safe file boundary | A request has one absolute root plus non-empty relative file paths. Production restricts the root to configured canonical allowed roots; traversal and symlink escapes are rejected. |
| Git facts | An existing worktree yields per-file churn, change frequency, authors, timestamps, rename lineage, deletion state, and change concentration/dispersion. |
| Honest risk score | The engine is deterministic, exposes formula/profile/contributions/limitations, and excludes explicitly unavailable metrics rather than presenting them as observed facts. |
| Human workflow | The frontend shows the direct three-step static → Git → Hotspot flow and only renders API responses; it has no saved-run, report, or export UI. |

A Hotspot score is not a defect, security, runtime, coverage, or incident prediction.

## 2. Component model and dependency direction

```text
frontend (React + Vite, RTL)
       │ relative /api/v1 requests
       ▼
internal/api (HTTP, validation, safe errors, OpenAPI, Swagger assets)
       ├──────────► application/static_analysis
       │                 └── analyzer/static (Tree-sitter parsing)
       ├──────────► githistory (fixed read-only Git commands)
       └──────────► scoring (deterministic risk-v1 engine)
                         ▲
                       domain (contracts and values only)
```

```text
backend/
├── cmd/server/                 # flags, allowed-root policy, guarded services, HTTP lifecycle
└── internal/
    ├── analysisguard/          # canonical local-path policy and resource limits
    ├── analyzer/static/        # Go, JavaScript and Python static parser/metrics
    ├── application/            # explicit-file static use case and static/Git input join helper
    ├── domain/                 # analysis, Git and risk value contracts
    ├── githistory/             # read-only local Git-history analyzer
    ├── scoring/                # risk-v1 score/rank/module/heatmap calculation
    ├── storage/failedfiles/    # process-local safe parser/read failure registry
    └── api/                    # direct HTTP routes, OpenAPI and locally served Swagger UI
```

`domain` has no HTTP, filesystem, parser, process, or UI dependency. `scoring` does not read files or invoke Git. The HTTP layer validates JSON and maps errors but does not calculate static metrics, Git facts, or scores. There is no repository scanner or persistence layer in this version.

## 3. Direct data flow

1. **Static request** — the caller sends one allowed absolute `rootPath` and selected relative file names to `POST /static-analysis`.
2. **Static analysis** — the service resolves the root, validates each relative path stays below it, reads at most configured byte limits, and parses supported source text. It returns per-file results and directory module aggregates. Parser/read errors become file diagnostics and safe process-local failure metadata.
3. **Git request** — the caller sends the absolute worktree location and optional inclusive RFC 3339 history bounds to `POST /git-analysis`.
4. **Git analysis** — the service confirms a local worktree and runs fixed `git log`, `git diff-tree --name-status`, and `git diff-tree --numstat` commands with `GIT_OPTIONAL_LOCKS=0`. It does not write to the worktree, index, refs, or remotes.
5. **Risk request** — the caller supplies the relevant static and Git values to `POST /risk-analysis`. The scoring engine validates fields and calculates ranking/contributions/modules/heatmap data.
6. **Frontend** — the workspace retains each response in memory for the current page. It asks the backend to calculate the score and displays returned values. Refreshing the page clears the unsaved results by design.

The current direct contract is intentionally composable: consumers can call the three analytical endpoints individually. It does not perform recursive discovery or automatically persist/join an entire project run.

## 4. Static analysis design

The static analyzer supports Go, JavaScript and Python. It returns:

- physical non-blank, non-comment lines of code;
- function/method and class counts;
- maximum and total declaration lengths;
- declaration locations;
- language, explicit status, fallback marker, and diagnostics; and
- directory-level aggregates of returned file results.

Files can be `analyzed`, `empty`, `no_executable_code`, `fallback`, `parse_error`, `unsupported`, or `read_error`. A malformed or unreadable selected file does not invent metrics and does not erase successful file results.

## 5. Git-history design

With no date bounds, Git analysis examines commits reachable from `HEAD`. `history.from` and `history.to` are optional inclusive UTC bounds. The response distinguishes empty or shallow history through diagnostics.

For every observed path identity, the analyzer reports added-plus-deleted line churn, distinct affecting commits, author count, first/last change times, previous paths for renames, deletion state, and monthly change concentration/dispersion. It invokes no project hooks or target commands.

## 6. `risk-v1` model

The requested default profile is:

| Metric | Identifier | Weight |
| --- | --- | ---: |
| Lines of code | `lines_of_code` | 0.15 |
| Maximum function length | `max_function_length` | 0.25 |
| Maximum class length | `max_class_length` | 0.10 |
| Git churn | `git_churn` | 0.25 |
| Change frequency | `change_frequency` | 0.25 |

For the file population in a request, each available metric uses its nearest-rank 95th-percentile value as a cap. A raw value is divided by that cap and capped at `1`. The engine computes:

```text
round(100 × Σ(effectiveWeight(metric) × normalized(metric, file)))
```

It assigns low (`0–34`), medium (`35–64`), or high (`65–100`) level, ranks by score then path, aggregates modules, and prepares backend-owned heatmap cells. Every returned contribution has raw value, cap, normalized value, effective weight, and weighted value.

When `availableMetrics` is supplied, metrics outside that set are explicitly unavailable and positive weights are renormalized only over the selected metrics. For the default static-only set, effective weights are LOC `0.30`, maximum function length `0.50`, and maximum class length `0.20`. This capability handles missing Git facts honestly; it does not infer them from zero.

## 7. Security and operating limits

At server startup, `cmd/server` builds a canonical `analysisguard.PathPolicy` from repeatable `-allowed-root` flags. If omitted, the current user's home directory is the default. The production static and Git services both use this policy. `NewHandler` and unguarded constructors exist only for in-process unit tests and embedding; they are not the server configuration path.

| Bound | Default | Flag |
| --- | ---: | --- |
| Explicit source files per static request | 10,000 | `-max-files` |
| One source file | 2 MiB | `-max-file-bytes` |
| Source bytes per static request | 128 MiB | `-max-total-source-bytes` |
| Git commits | 20,000 | `-max-git-commits` |
| Output retained by one Git command | 16 MiB | `-max-git-output-bytes` |

The HTTP server also has header/read/write/idle timeouts. Safe API errors never return source bytes, parser internals, absolute host paths, or Git command output. Development CORS is permissive without credentials for the relative/proxied UI workflow and must be restricted before exposing the server beyond a local development environment.

## 8. Test data and verification

The backend tests construct real temporary Go/JavaScript/Python source text and temporary Git worktrees with commits, renames, deletions, date ranges, malformed inputs, path boundaries, and resource-limit cases. Tests do not execute fixture source. HTTP tests use `httptest` and direct JSON payloads.

Frontend tests mock only transport responses to verify that the page renders API results and sends relative direct requests; score computation remains server-owned. The frontend production build, type-check, and Vitest suite are part of the phase-only quality gate.

Run the full commands in [README.md](../README.md#verify). Host-scale timing/RSS and empirical defect-prediction validation are not claimed; see [qa-final-evaluation.md](qa-final-evaluation.md).

## 9. Explicitly deferred

This phase-only architecture does not include recursive discovery, analysis queues, cancellation, concurrency scheduling, SQLite or another run store, restart recovery, lifecycle routes, saved reports, result query/filter APIs, export, dashboards, comparison, remote cloning, TypeScript parsing, or target-project code execution. Those later capabilities are intentionally available only in the complete-product branch.
