# Local API Contract — v1

**Status:** implemented backend surface as of BE-65–BE-76. The embedded OpenAPI document at [`backend/internal/api/openapi.yaml`](../backend/internal/api/openapi.yaml) is the machine-readable source of truth.

**Base URL:** `http://127.0.0.1:<configured-port>/api/v1`

The service is local-only. Every accepted path identifies a directory visible to the Go backend process on the same machine. It never accepts a Git URL, uploads source, executes project code, or contacts a remote repository service.

## Conventions

- Requests and JSON responses use `application/json; charset=utf-8`.
- Timestamps are RFC 3339 UTC strings; analysis IDs are opaque.
- Unknown fields in write requests are rejected.
- Result paths are slash-normalized **project-relative** display data. Absolute backend paths, Git roots, and local database paths are never serialized.
- Standard errors use `{ "error": { "code": "...", "message": "..." } }`; messages are safe for display and do not disclose host paths.
- List pagination uses one-based `page` and `pageSize` (defaults: `1`, `50`; maximum `200`).

## Direct analytical endpoints

These existing synchronous endpoints remain available and do not create persisted runs:

| Endpoint | Purpose |
| --- | --- |
| `GET /health` | Service/version health response. |
| `GET /openapi.yaml` | Embedded OpenAPI 3.1 document. |
| `POST /static-analysis` | Safely read selected relative files below a supplied absolute `rootPath`; does not execute source. |
| `GET /static-analysis/failures` | Process-local safe metadata for static read/parser failures. |
| `POST /git-analysis` | Read-only facts for an absolute local Git worktree and optional inclusive history range. |
| `POST /risk-analysis` | Deterministic hotspot scoring from supplied analytical metrics. |

Static support is Go, JavaScript (`.js`, `.mjs`, `.cjs`), and Python. The delivered `risk-v1` model uses lines of code, maximum function length, maximum class length, Git churn, and change frequency. It does **not** calculate cyclomatic complexity, critical bands, runtime behavior, coverage, or vulnerability findings.

## Persisted asynchronous analyses

### `POST /analyses`

Starts a bounded asynchronous local analysis and returns `202 Accepted`.

```json
{
  "projectPath": "/absolute/path/on/backend-host/project",
  "history": {
    "from": "2026-01-01T00:00:00Z",
    "to": "2026-12-31T23:59:59Z"
  },
  "profile": {
    "version": "risk-v1",
    "weights": {
      "lines_of_code": 0.15,
      "max_function_length": 0.25,
      "max_class_length": 0.10,
      "git_churn": 0.25,
      "change_frequency": 0.25
    },
    "bands": { "lowMax": 34, "mediumMax": 64 }
  }
}
```

`projectPath` is the only analysis location input. It must be an accessible absolute directory after canonicalization. A Git worktree is optional: a non-Git project still receives discovery, static analysis, and static-only risk scoring. Invalid history ranges and invalid custom profiles are rejected before queuing.

The response shape is:

```json
{
  "analysis": {
    "id": "analysis_...",
    "status": "queued",
    "createdAt": "2026-11-14T09:00:00Z",
    "history": { "from": null, "to": null },
    "stages": [
      { "stage": "validation", "status": "pending", "progress": { "total": 0, "completed": 0 } }
    ],
    "progress": {
      "filesDiscovered": 0,
      "filesEligible": 0,
      "filesAnalyzed": 0,
      "filesWithErrors": 0,
      "commitsTotal": 0,
      "commitsScanned": 0,
      "percent": 0
    },
    "diagnostics": []
  }
}
```

A local SQLite unique active-scope constraint prevents concurrent `queued` or `running` analyses of the same Git worktree (or the same canonical non-Git project). The API returns `409 ANALYSIS_ALREADY_RUNNING` for that conflict. Completed, failed, and cancelled projects may be analyzed again.

### `GET /analyses`

Lists persisted lifecycle records, newest first. Optional query parameters:

- `status`: one of `queued`, `running`, `completed`, `failed`, `cancelled`
- `page`: positive integer
- `pageSize`: integer `1..200`

The response is `{ "items": [...], "page": 1, "pageSize": 50, "totalItems": 0 }`.

### `GET /analyses/{analysisId}`

Returns an analysis lifecycle snapshot. Overall `status` is restricted to:

- `queued`
- `running`
- `completed`
- `failed`
- `cancelled`

`currentStage` is operational detail (`validation`, `discovery`, `analysis`, `risk_scoring`, or `finalizing`). The independent stage array records `validation`, `discovery`, `static_analysis`, `git_analysis`, and `risk_scoring`, each with a status of `pending`, `running`, `completed`, `unavailable`, `failed`, `cancelled`, or `skipped`.

Static and Git work run independently in parallel when a Git worktree is available. Progress supplies file and commit counts plus a monotonic percentage. Per-file read/parser failures remain in the static artifact and are represented by safe run diagnostics without discarding successful files.

### `POST /analyses/{analysisId}/cancel`

Requests cooperative cancellation and returns `202`. Active work checks cancellation at discovery, file, and Git command boundaries. A cancelled run retains safe stage/progress/diagnostic information but is never returned as a completed report. A terminal or unknown run returns `409` or `404` respectively.

### `GET /analyses/{analysisId}/report`

Returns a `200` immutable completed report only after successful finalization. Before completion it returns `409`; an unknown ID returns `404`.

The report joins the safe lifecycle snapshot with persisted static, optional Git, and optional risk artifacts. It survives backend restarts through local SQLite storage.

## Scope, discovery, and Git availability

Discovery canonicalizes the selected local directory and considers only regular, supported, non-binary files inside it. It excludes unsafe/out-of-project symlinks, deleted Git paths, `.git`, dependency/build/cache directories, and ignored files. A Git worktree uses `git ls-files --cached --others --exclude-standard`; a non-Git project uses the independent `.gitignore` matcher. No project command, hook, build, test, import, or script is run.

If `projectPath` lies below a Git worktree, static discovery remains inside that selected directory. Git history runs against the enclosing Git root with a repository-relative path filter; emitted paths are rebased to the selected project.

Git history failures and missing scoped history do not fail a useful static analysis. The Git stage becomes `unavailable` with diagnostics, the report retains static artifacts, and scoring excludes Git churn and change frequency rather than inventing zero values. The report records both requested and effective risk profiles and the available/unavailable metric identities. When static-only default scoring applies, effective weights are renormalized to LOC `0.30`, maximum function length `0.50`, and maximum class length `0.20`.

## Persistence and restart behavior

SQLite is the primary `RunStore`. It persists lifecycle state, stage snapshots, progress, safe diagnostics, requested profile/history, completed summaries, calculated artifacts, and the immutable final report. Persisting a report and moving its run to `completed` occur atomically. On startup, an interrupted `queued` or `running` record is changed to `failed` with a `BACKEND_RESTARTED` diagnostic; analysis is not silently resumed.

## Deferred API surface

Deletion, file-list/detail views, exports, named profile management, and frontend dashboard routes are not implemented in this increment and must not be inferred from this contract. The OpenAPI document lists only the implemented HTTP endpoints above.
