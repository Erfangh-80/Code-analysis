# Local Core API Contract — v1, phases 1–4

**Status:** implemented direct local API. The machine-readable source of truth is [`backend/internal/api/openapi.yaml`](../backend/internal/api/openapi.yaml).

**Base URL:** `http://127.0.0.1:<port>/api/v1`

This is not a durable-analysis API. A request returns a direct analytical result and the service retains no saved analysis runs, reports, exports, or result-query pages.

## Conventions and safety

- Requests and JSON responses use `application/json; charset=utf-8` unless noted.
- Unknown fields in POST payloads are rejected.
- Standard errors are `{ "error": { "code": "...", "message": "..." } }`. Messages are safe for display and do not disclose source text, absolute host paths, or Git command output.
- All returned source/Git paths are slash-normalized project/worktree-relative display paths.
- `400 VALIDATION_ERROR` denotes malformed, unsafe, inaccessible, or invalid analytical input. `413 RESOURCE_LIMIT_EXCEEDED` denotes a configured static/Git resource bound. `405 METHOD_NOT_ALLOWED` includes `Allow`.
- The server accepts only local absolute directories below its configured `-allowed-root` policy. It never accepts repository URLs or source uploads, clones a repository, executes target code, or modifies a target worktree.
- Development CORS allows `GET`, `POST`, and `OPTIONS` without credentials when an `Origin` is sent. Restrict it before exposing this service beyond local development.

## Routes

| Method | Route | Direct result |
| --- | --- | --- |
| `GET` | `/health` | `{ "status": "ok", "version": "0.1.0-core" }` |
| `GET` | `/openapi.yaml` | Embedded OpenAPI 3.1 YAML (`application/yaml`) |
| `POST` | `/static-analysis` | Static file and module analysis |
| `GET` | `/static-analysis/failures` | Safe process-local parser/read failures |
| `POST` | `/git-analysis` | Local worktree history facts |
| `POST` | `/risk-analysis` | Explainable ranked Hotspot report |
| `GET` | `/swagger/` | Local Swagger UI assets and page |

Routes beginning with `/analyses` are intentionally not implemented on this branch.

## `POST /static-analysis`

Analyze explicit files below one local root. The server reads and parses text only. Supported extensions are Go (`.go`), JavaScript (`.js`, `.mjs`, `.cjs`), and Python (`.py`).

```json
{
  "rootPath": "/absolute/path/to/project",
  "files": [
    "internal/service/worker.go",
    "web/app.js",
    "scripts/report.py"
  ]
}
```

`rootPath` must be an accessible, allowed absolute directory. Every `files` entry must be non-empty, relative, and resolve below it. The selected file list is not recursive discovery.

A successful response is a `BatchAnalysis` object:

```json
{
  "files": [
    {
      "path": "internal/service/worker.go",
      "language": "go",
      "status": "analyzed",
      "usedFallback": false,
      "metrics": {
        "linesOfCode": 42,
        "functionCount": 3,
        "classCount": 0,
        "functionLength": { "max": 15, "total": 29 },
        "classLength": { "max": 0, "total": 0 }
      },
      "functions": [],
      "classes": [],
      "diagnostics": []
    }
  ],
  "modules": [
    {
      "path": "internal/service",
      "fileCount": 1,
      "failedFileCount": 0,
      "metrics": {
        "linesOfCode": 42,
        "functionCount": 3,
        "classCount": 0,
        "functionLength": { "max": 15, "total": 29 },
        "classLength": { "max": 0, "total": 0 }
      }
    }
  ]
}
```

`status` is one of `analyzed`, `empty`, `no_executable_code`, `fallback`, `parse_error`, `unsupported`, or `read_error`. A file-level failure stays explicit; it does not cause the server to fabricate successful metrics for it or discard unrelated selected files.

## `GET /static-analysis/failures`

Returns only safe failure metadata captured in this process:

```json
{ "items": [{ "path": "broken.js", "language": "javascript", "status": "parse_error", "usedFallback": false, "diagnostics": [], "recordedAt": "2026-11-14T09:00:00Z" }] }
```

It contains no source content and is neither a persisted audit log nor a complete project report.

## `POST /git-analysis`

Analyze the read-only history of one existing local Git worktree. Omit `history` to analyze all commits reachable from `HEAD`; present bounds are inclusive RFC 3339 timestamps.

```json
{
  "repositoryPath": "/absolute/path/to/worktree",
  "history": {
    "from": "2026-01-01T00:00:00Z",
    "to": "2026-12-31T23:59:59Z"
  }
}
```

A successful response includes `commitCount`, echoed `history`, `files`, and `diagnostics`. Each file has project-relative `path`, `previousPaths`, `deleted`, `gitChurn`, `changeFrequency`, `authorCount`, optional `firstChangedAt`/`lastChangedAt`, `changeConcentration`, and `changeDispersion`.

A missing worktree or invalid range returns `400 VALIDATION_ERROR`; a resource limit returns `413 RESOURCE_LIMIT_EXCEEDED`. Git operational failures that are not client-input errors return `500 GIT_HISTORY_ERROR`. Empty or shallow history is represented by successful response diagnostics where analysis can continue.

## `POST /risk-analysis`

The scoring engine accepts collected analytical facts; it does not read source or Git itself. With no `availableMetrics`, each file must include all five `metrics` properties. With an explicit availability set, each file must provide at least the selected metric properties.

```json
{
  "files": [
    {
      "path": "internal/api/server.go",
      "metrics": {
        "linesOfCode": 240,
        "maxFunctionLength": 55,
        "maxClassLength": 0,
        "gitChurn": 180,
        "changeFrequency": 12
      }
    }
  ],
  "availableMetrics": [
    "lines_of_code",
    "max_function_length",
    "max_class_length",
    "git_churn",
    "change_frequency"
  ]
}
```

Paths must be unique non-empty repository-relative paths. All metric values are non-negative integers. Supported metric IDs are:

- `lines_of_code`
- `max_function_length`
- `max_class_length`
- `git_churn`
- `change_frequency`

The optional `profile` has a non-empty version, supported non-negative weights that sum to `1.0`, and `bands` satisfying `0 <= lowMax < mediumMax < 100`. The optional `heatmap` has distinct available `xMetric` and `yMetric` values. Invalid profile/input/heatmap configurations return `400 VALIDATION_ERROR`.

### Default model and response guarantees

The default `risk-v1` weights are LOC `0.15`, max function length `0.25`, max class length `0.10`, Git churn `0.25`, and change frequency `0.25`; bands are low `0–34`, medium `35–64`, high `65–100`.

For the current request's file population, the engine caps each available metric at its nearest-rank 95th percentile and computes:

```text
round(100 × Σ(effectiveWeight(metric) × normalized(metric, file)))
```

When `availableMetrics` excludes a metric, the engine records it in `unavailableMetrics` and renormalizes positive requested weights across only available metrics. It does not treat an unavailable Git analysis as observed zero Git activity.

The response includes `profile` (effective), `requestedProfile`, available/unavailable metrics, formula, normalization description, limitations, ranked `files`, aggregate `modules`, and backend-prepared `heatmap` cells. Each file contribution identifies the raw value, normalization cap, normalized value, effective weight, and weighted contribution.

## HTTP safety and resource behavior

The production server configures these defaults, which can be tightened at startup:

| Resource | Default |
| --- | ---: |
| Static request file count | 10,000 |
| Bytes per source file | 2 MiB |
| Total source bytes | 128 MiB |
| Git commits | 20,000 |
| Git output retained per command | 16 MiB |

`/static-analysis` reports a request-wide file-count excess as `413`. Per-file static read/parse/resource problems are returned as safe file diagnostics so that valid selected files remain useful. Git bounds return `413` rather than retaining unbounded command output.

## OpenAPI and Swagger

Download the complete typed contract with:

```bash
curl http://127.0.0.1:8080/api/v1/openapi.yaml
```

After starting the backend, local Swagger UI is available at `http://127.0.0.1:8080/swagger/`. Its assets and specification are served by the backend rather than fetched from an external documentation service.
