# Local API Contract — v1

**Status:** architecture baseline; must be implemented and contract-tested before full frontend implementation.

**Base URL:** `http://127.0.0.1:<configured-port>/api/v1` in production by default.

**Transport:** HTTP on the local machine only; all request and response bodies are JSON except CSV export.

This contract is intentionally local-first. `repositoryPath` identifies a path visible to the Go backend process on the same machine; it is never a URL and must never be forwarded to another service.

## 1. Conventions

- Requests and JSON responses use `application/json; charset=utf-8`.
- Timestamps are RFC 3339 UTC strings.
- IDs are opaque strings. The UI must not derive data from their format.
- File references use an opaque `fileId`; repository-relative `path` is display data and never used as a URL path segment.
- All paths in results and exports are normalized repository-relative paths using `/`. Absolute host paths are never returned.
- Unknown JSON fields are rejected in write requests. Required fields are marked below.
- Successful asynchronous creation returns `202 Accepted`, not a fabricated completed report.
- Pagination uses `page` (one-based) and `pageSize`; default `page=1`, `pageSize=50`, maximum `pageSize=200`.
- Filtering/sorting only changes the response view. It cannot recalculate a metric or score.

## 2. Common schemas

### Error response

Every non-2xx response uses this shape:

```json
{
  "error": {
    "code": "INVALID_REPOSITORY_PATH",
    "message": "The path is not a readable local Git working tree.",
    "field": "repositoryPath",
    "requestId": "req_..."
  }
}
```

`field` is optional. `message` must be safe for display and must not expose unrelated filesystem paths. Anticipated error codes include `VALIDATION_ERROR`, `INVALID_REPOSITORY_PATH`, `PATH_NOT_ACCESSIBLE`, `ANALYSIS_NOT_FOUND`, `INVALID_RUN_STATE`, `UNSUPPORTED_PROFILE`, `RESOURCE_LIMIT_EXCEEDED`, and `INTERNAL_ERROR`.

### History window

```json
{
  "from": "2025-01-01T00:00:00Z",
  "to": "2025-12-31T23:59:59Z"
}
```

Both fields are optional UTC timestamps. Omit `history` or both fields to analyze all reachable history. Bounds are inclusive. `from` must not be later than `to`.

### Metric and scoring shapes

```json
{
  "metrics": {
    "loc": 214,
    "cyclomaticComplexity": 42,
    "functionLength": { "max": 51, "total": 167 },
    "classLength": { "max": 0, "total": 0 },
    "gitChurn": 387,
    "changeFrequency": 19
  },
  "risk": {
    "score": 74,
    "band": "high",
    "profileVersion": "default-v1",
    "contributions": [
      { "metric": "cyclomaticComplexity", "normalized": 0.81, "weight": 0.30, "weighted": 0.243 },
      { "metric": "gitChurn", "normalized": 0.75, "weight": 0.25, "weighted": 0.188 }
    ]
  }
}
```

`score` is an integer from 0 to 100 and `band` is one of `low`, `medium`, `high`, or `critical`. `normalized`, `weight`, and `weighted` are backend-calculated decimal values in `[0,1]`. Metrics unavailable for a file are represented with `null` and a diagnostic or status; they are never replaced with zero.

### Analysis status

An analysis has one of `queued`, `validating`, `discovering`, `static_analysis`, `git_history`, `scoring`, `storing`, `completed`, `failed`, or `cancelled`.

Only `completed` has a complete immutable report. `failed` and `cancelled` may contain safe diagnostics and progress but must not be presented as a report ready for ranking or export.

## 3. Endpoints

### `GET /health`

Returns local server availability. It does not inspect a repository.

**200 response**

```json
{ "status": "ok", "version": "0.1.0" }
```

### `GET /scoring-profiles`

Lists locally configured scoring profiles that can be selected for a new run.

**200 response**

```json
{
  "items": [
    {
      "id": "default",
      "version": "default-v1",
      "name": "Default maintainability risk",
      "description": "Balanced static and change-history indicators.",
      "metricWeights": {
        "cyclomaticComplexity": 0.30,
        "loc": 0.10,
        "functionLength": 0.10,
        "classLength": 0.05,
        "gitChurn": 0.25,
        "changeFrequency": 0.20
      },
      "bands": {
        "lowMax": 34,
        "mediumMax": 64,
        "highMax": 79
      }
    }
  ]
}
```

Profile editing is a later local configuration capability. The MVP API starts by exposing validated profiles; it does not let the browser write arbitrary server configuration.

### `POST /analyses`

Creates a local asynchronous analysis job.

**Request**

```json
{
  "repositoryPath": "/absolute/path/on/backend-host/repository",
  "history": {
    "from": "2025-01-01T00:00:00Z",
    "to": "2025-12-31T23:59:59Z"
  },
  "scoringProfileId": "default"
}
```

| Field | Required | Rules |
| --- | --- | --- |
| `repositoryPath` | Yes | Absolute path visible to the local backend; must resolve to a readable Git worktree. It is used only locally and is not echoed in result payloads. |
| `history` | No | Inclusive UTC range. Omit for complete reachable history. |
| `scoringProfileId` | No | Defaults to the configured `default` profile. |

**202 response**

```json
{
  "analysis": {
    "id": "analysis_01J...",
    "status": "queued",
    "createdAt": "2026-11-14T09:00:00Z",
    "history": { "from": null, "to": null },
    "scoringProfile": { "id": "default", "version": "default-v1" }
  }
}
```

**Errors:** `400` validation error, `403` inaccessible local path, `404` not a Git worktree, `409` conflicting active job policy, `413` known resource limit, `500` internal error.

### `GET /analyses/{analysisId}`

Returns status, scope, progress, and safe diagnostics. Poll this endpoint while the job is running.

**200 response**

```json
{
  "analysis": {
    "id": "analysis_01J...",
    "status": "git_history",
    "createdAt": "2026-11-14T09:00:00Z",
    "startedAt": "2026-11-14T09:00:01Z",
    "completedAt": null,
    "history": { "from": null, "to": null },
    "scoringProfile": { "id": "default", "version": "default-v1" },
    "progress": {
      "stage": "git_history",
      "filesDiscovered": 128,
      "filesAnalyzed": 128,
      "commitsScanned": 341,
      "percent": 68
    },
    "summary": null,
    "diagnostics": []
  }
}
```

For `completed`, `summary` is present:

```json
{
  "supportedFiles": 110,
  "unsupportedFiles": 18,
  "filesWithParseErrors": 2,
  "riskBands": { "low": 42, "medium": 36, "high": 21, "critical": 11 },
  "scoreModelVersion": "default-v1"
}
```

### `POST /analyses/{analysisId}/cancel`

Requests cancellation of a non-terminal job. Cancellation is cooperative and checked at safe file/commit boundaries.

**202 response**

```json
{ "analysis": { "id": "analysis_01J...", "status": "cancelled" } }
```

Returns `409` for a terminal run and `404` for an unknown run.

### `DELETE /analyses/{analysisId}`

Deletes the locally stored result and associated local metadata. It never modifies the analyzed repository.

**204 response:** empty body.

**409:** job still active. Cancel it first.

### `GET /analyses/{analysisId}/files`

Returns a paginated, API-calculated view of files for a completed run.

**Query parameters**

| Parameter | Values | Default |
| --- | --- | --- |
| `page` | positive integer | `1` |
| `pageSize` | `1..200` | `50` |
| `q` | path substring search | omitted |
| `language` | repeated `go`, `javascript`, `python`, `unsupported` | omitted |
| `riskBand` | repeated `low`, `medium`, `high`, `critical` | omitted |
| `analysisStatus` | repeated `analyzed`, `parse_error`, `unsupported` | omitted |
| `sort` | `riskScore`, `path`, `loc`, `cyclomaticComplexity`, `gitChurn`, `changeFrequency` | `riskScore` |
| `direction` | `asc`, `desc` | `desc` |

**200 response**

```json
{
  "items": [
    {
      "fileId": "file_01J...",
      "path": "internal/api/server.go",
      "language": "go",
      "analysisStatus": "analyzed",
      "metrics": {
        "loc": 214,
        "cyclomaticComplexity": 42,
        "functionLength": { "max": 51, "total": 167 },
        "classLength": { "max": 0, "total": 0 },
        "gitChurn": 387,
        "changeFrequency": 19
      },
      "risk": {
        "score": 74,
        "band": "high",
        "profileVersion": "default-v1",
        "contributions": [
          { "metric": "cyclomaticComplexity", "normalized": 0.81, "weight": 0.30, "weighted": 0.243 }
        ]
      }
    }
  ],
  "page": { "number": 1, "size": 50, "totalItems": 110, "totalPages": 3 }
}
```

Returns `409` until the analysis is `completed` and `404` for an unknown run.

### `GET /analyses/{analysisId}/files/{fileId}`

Returns the complete result record for one file in a completed run, including function/class summaries, all score contributions, and diagnostics.

**200 response additions**

```json
{
  "file": {
    "fileId": "file_01J...",
    "path": "internal/api/server.go",
    "language": "go",
    "analysisStatus": "analyzed",
    "metrics": { "loc": 214, "cyclomaticComplexity": 42 },
    "risk": { "score": 74, "band": "high", "profileVersion": "default-v1", "contributions": [] },
    "declarations": {
      "functions": [
        { "name": "Serve", "startLine": 20, "endLine": 70, "loc": 45, "cyclomaticComplexity": 12 }
      ],
      "classes": []
    },
    "diagnostics": []
  }
}
```

The API does not return source-code contents in MVP.

### `GET /analyses/{analysisId}/heatmap`

Returns heatmap cells from a completed run. The backend selects and validates dimensions; it returns the same stored risk values as file results.

**Query parameters**

- `x`: one of `loc`, `cyclomaticComplexity`, `gitChurn`, `changeFrequency`; default `cyclomaticComplexity`.
- `y`: one of the same values and different from `x`; default `gitChurn`.

**200 response**

```json
{
  "axes": { "x": "cyclomaticComplexity", "y": "gitChurn" },
  "items": [
    {
      "fileId": "file_01J...",
      "path": "internal/api/server.go",
      "x": 42,
      "y": 387,
      "risk": { "score": 74, "band": "high", "profileVersion": "default-v1", "contributions": [] }
    }
  ]
}
```

The frontend chooses presentation scales and interactions but does not recalculate an axis value or score.

### `GET /analyses/{analysisId}/export`

Produces an export for a completed immutable run.

| Query parameter | Required | Values |
| --- | --- | --- |
| `format` | Yes | `json` or `csv` |

**200 responses**

- `format=json`: `application/json`, preserving the run summary and full file-result schema.
- `format=csv`: `text/csv; charset=utf-8` with attachment disposition. The fixed initial header is:

```text
path,language,analysis_status,risk_score,risk_band,loc,cyclomatic_complexity,max_function_length,total_function_length,max_class_length,total_class_length,git_churn,change_frequency,profile_version
```

CSV cells are escaped and values that could be interpreted as spreadsheet formulas are neutralized by the backend.

## 4. Contract compliance requirements

Before the frontend dashboard is built, the backend must have contract tests covering each endpoint, success state, terminal/non-terminal state, validation failure, and export content type. The frontend API client must be typed from or checked against this document and its tests must use an implemented local test server for analytic values. Mock transport may be used only to test presentation loading/error states; it cannot establish that an analysis capability works.
