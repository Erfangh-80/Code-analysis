# Local API Contract — v1

**Status:** implemented backend surface through BE-92. The embedded OpenAPI document at [`backend/internal/api/openapi.yaml`](../backend/internal/api/openapi.yaml) is the machine-readable source of truth.

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

### `POST /analyses/validate-path`

Performs the same canonical local-directory and configured-root check used by analysis start, but does **not** queue, persist, scan, or execute anything. The request is deliberately limited to the local `projectPath` input:

```json
{ "projectPath": "/absolute/path/on/backend-host/project" }
```

A successful response is `{ "valid": true }`. Invalid, inaccessible, relative, or unauthorized paths return `400 VALIDATION_ERROR` with a display-safe generic message; no absolute host path is returned. The endpoint is a UI preflight, not a reservation: `POST /analyses` validates the path again before it creates a run, so an intervening filesystem or policy change remains safe.

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

### Completed result views

Every endpoint in this section reads the immutable stored report and returns `409` for a non-completed analysis and `404` for an unknown analysis. No endpoint recalculates static, Git, score, module, or heatmap values.

| Endpoint | Result |
| --- | --- |
| `GET /analyses/{analysisId}/summary` | Completed summary, risk-level counts, module count, Git-artifact flag, and availability/effective-profile metadata. |
| `GET /analyses/{analysisId}/hotspots` | Paginated scored hotspots. |
| `GET /analyses/{analysisId}/files` | Paginated static-file records joined with optional Git and risk records. |
| `GET /analyses/{analysisId}/files/detail?path=project/relative.go` | One exact project-relative file record. The path must be non-empty, slash-normalized, non-absolute, and non-escaping. |
| `GET /analyses/{analysisId}/modules` | Static module aggregates and optional risk-ranked module aggregates. |
| `GET /analyses/{analysisId}/heatmap` | Saved backend-prepared heatmap axes and cells. It does not accept alternate axes because that would imply a recalculation. |
| `GET /analyses/{analysisId}/report` | Complete raw immutable report, including all persisted artifacts. |

`/hotspots` and `/files` accept one-based `page` and `pageSize` (`1..200`), optional case-insensitive path query `q`, optional project-relative `module` prefix, repeated `riskLevel` values, a supported `sort`, and `direction=asc|desc`. `/files` additionally accepts repeated `language` and `analysisStatus` values plus `minScore`/`maxScore` integers (`0..100`, with `minScore <= maxScore`). Score-bound filtering excludes files with no saved risk score rather than treating missing data as zero. The server rejects unknown, repeated scalar, malformed, out-of-range, or endpoint-incompatible query parameters with `400 VALIDATION_ERROR`. Sorting has deterministic path ties.

### `GET /analyses/{analysisId}/export?format=json|csv`

Exports a completed report with an attachment disposition:

- `format=json` returns the same immutable report JSON as the report endpoint.
- `format=csv` returns one safe row per discovered static file with actual v1 columns: path, language, static status, risk score/level when available, static metrics, and Git churn/frequency when available.

CSV text cells that begin with `=`, `+`, `-`, or `@` are prefixed to prevent spreadsheet formula interpretation. Export never contains an absolute host path or source contents.

## Browser access and security headers

During the current development phase, the local API permits cross-origin browser requests from every origin. Requests with an `Origin` header receive `Access-Control-Allow-Origin: *`, and the permitted methods are `GET`, `POST`, and `OPTIONS` with the `Content-Type` request header. CORS credentials are not enabled.

This permissive policy is temporary and must be replaced with a same-origin or explicit trusted-origin policy before a public release. Responses continue to include `nosniff`, frame/referrer/cross-origin-resource protections, a restrictive frame/base/object CSP, and `Cache-Control: no-store`.

## Scope, discovery, and Git availability

Discovery canonicalizes the selected local directory and considers only regular, supported, non-binary files inside it. It excludes unsafe/out-of-project symlinks, deleted Git paths, `.git`, dependency/build/cache directories, and ignored files. A Git worktree uses `git ls-files --cached --others --exclude-standard`; a non-Git project uses the independent `.gitignore` matcher. No project command, hook, build, test, import, or script is run.

If `projectPath` lies below a Git worktree, static discovery remains inside that selected directory. Git history runs against the enclosing Git root with a repository-relative path filter; emitted paths are rebased to the selected project.

Git history failures and missing scoped history do not fail a useful static analysis. The Git stage becomes `unavailable` with diagnostics, the report retains static artifacts, and scoring excludes Git churn and change frequency rather than inventing zero values. The report records both requested and effective risk profiles and the available/unavailable metric identities. When static-only default scoring applies, effective weights are renormalized to LOC `0.30`, maximum function length `0.50`, and maximum class length `0.20`.

## Persistence and restart behavior

SQLite is the primary `RunStore`. It persists lifecycle state, stage snapshots, progress, safe diagnostics, requested profile/history, completed summaries, calculated artifacts, and the immutable final report. Persisting a report and moving its run to `completed` occur atomically. On startup, an interrupted `queued` or `running` record is changed to `failed` with a `BACKEND_RESTARTED` diagnostic; analysis is not silently resumed.

## Deferred API surface

Deletion, named profile management, custom recomputation of saved heatmap axes, and frontend dashboard routes are not implemented in this increment and must not be inferred from this contract. The OpenAPI document lists only the implemented HTTP endpoints above.
