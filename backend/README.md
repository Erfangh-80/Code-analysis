# Backend

Independent Go module for the local Code Risk Analyzer API. The service reads selected source files from the same machine, parses them statically, and never executes target code.

## Prerequisites

- Go 1.23 or later;
- a C11 compiler such as GCC or Clang with CGO enabled, required by the pinned Tree-sitter language grammars.

## Run locally

```bash
cd backend
go run ./cmd/server
```

The server binds to `127.0.0.1:8080` by default.

```bash
curl http://127.0.0.1:8080/api/v1/health
```

```json
{"status":"ok","version":"0.1.0"}
```

## Implemented static-analysis API

`POST /api/v1/static-analysis` accepts an absolute local `rootPath` and one or more paths relative to that root. It reads only those local files, detects Go, JavaScript, and Python by extension, and returns real file and module metrics.

```bash
curl --request POST http://127.0.0.1:8080/api/v1/static-analysis \
  --header 'Content-Type: application/json' \
  --data '{
    "rootPath": "/absolute/path/to/local/repository",
    "files": ["internal/service/worker.go", "web/app.js", "scripts/report.py"]
  }'
```

The response includes:

- functions and methods;
- JavaScript/Python classes;
- physical non-blank, non-comment LOC;
- maximum and total function/class lengths;
- file-level metrics and directory-level module aggregates;
- honest statuses for empty, non-executable, unsupported, parse-error, fallback, and unreadable files.

Parser failures are stored as safe local metadata—never source content—and can be reviewed with:

```bash
curl http://127.0.0.1:8080/api/v1/static-analysis/failures
```

## Implemented Git-history analysis API

`POST /api/v1/git-analysis` reads the full reachable history of an existing local Git worktree by default. It does not execute repository code or modify the repository. Optional inclusive RFC 3339 UTC `from` and `to` bounds limit the history window.

```bash
curl --request POST http://127.0.0.1:8080/api/v1/git-analysis \
  --header 'Content-Type: application/json' \
  --data '{
    "repositoryPath": "/absolute/path/to/local/repository",
    "history": {
      "from": "2024-01-01T00:00:00Z",
      "to": "2024-12-31T23:59:59Z"
    }
  }'
```

The result reports per-file Git churn, change frequency, distinct author count, first/last change time, rename lineage, final deletion state, and monthly change concentration/dispersion. Empty histories and shallow repositories return explicit diagnostics instead of fabricated metrics.

## Implemented Risk Engine API

`POST /api/v1/risk-analysis` calculates a deterministic, explainable hotspot score for each complete per-file metric record. The engine is a separate `internal/scoring` package: it does not read files, invoke Git, serve HTTP, or depend on the frontend. The API layer only decodes the request and returns the engine's report.

The input must contain every current signal for every repository-relative file:

- `linesOfCode`, `maxFunctionLength`, and `maxClassLength` from local static analysis;
- `gitChurn` and `changeFrequency` from local Git-history analysis.

The endpoint rejects missing, negative, unsafe, duplicate, or unknown metric fields. Server-side application orchestration can join the existing domain results with `application.BuildHotspotInputs`; the browser must only render the returned report and must never calculate a metric, normalization, risk score, band, rank, module aggregate, or heatmap coordinate.

```bash
curl --request POST http://127.0.0.1:8080/api/v1/risk-analysis \
  --header 'Content-Type: application/json' \
  --data '{
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
    ]
  }'
```

The default local profile is `risk-v1` with these weights:

| Signal | Model identifier | Weight |
| --- | --- | ---: |
| Lines of code | `lines_of_code` | 0.15 |
| Maximum function length | `max_function_length` | 0.25 |
| Maximum class length | `max_class_length` | 0.10 |
| Git churn | `git_churn` | 0.25 |
| Change frequency | `change_frequency` | 0.25 |

A request may provide a complete versioned profile with recognized non-negative weights summing to `1.0`, plus `lowMax` and `mediumMax` thresholds. The default bands are **low** `0–34`, **medium** `35–64`, and **high** `65–100`.

For each metric, the engine finds its nearest-rank 95th-percentile value across the requested files. It divides each raw value by that cap and limits the normalized result to `1`. The exact backend formula is:

```text
round(100 × sum(weight(metric) × normalized(metric, file)))
```

Each returned contribution includes the raw value, normalization cap, normalized value, weight, and weighted value. The response also includes profile, formula, normalization text, limitations, files ranked by descending score (then path), directory-level module mean scores, and backend-prepared heatmap cells. Heatmap defaults to `git_churn` on the x axis and `lines_of_code` on the y axis; callers may select two distinct supported metrics.

### Interpretation limits

- The score is a review/refactoring priority signal, not a prediction of defects, outages, or security vulnerabilities.
- Scores are relative to the requested file population; scores from different runs are not directly comparable.
- Version 1 includes only the five signals listed above. It does not include cyclomatic complexity, semantic behavior, runtime behavior, test coverage, ownership quality, or vulnerability data.
- A path with no record in the selected Git history receives zero churn and change frequency. Inspect the Git-history scope and diagnostics before interpreting that result.
- All metrics remain static/read-only: neither the static analyzer nor the Git analyzer executes target-repository code or changes its worktree.

## Persisted local analyses and reports

`POST /api/v1/analyses` starts a durable asynchronous analysis for one accessible absolute local `projectPath`. Git is optional: static analysis and availability-aware static-only risk scoring still complete for a non-Git project. Poll `GET /api/v1/analyses/{analysisId}` for lifecycle, independent stage statuses, progress, and safe diagnostics; `GET /api/v1/analyses` lists saved runs; and `POST /api/v1/analyses/{analysisId}/cancel` requests cooperative cancellation.

After a completed run, the report and its projection/export APIs read the immutable SQLite report rather than rerunning analysis:

```text
GET /api/v1/analyses/{analysisId}/summary
GET /api/v1/analyses/{analysisId}/hotspots
GET /api/v1/analyses/{analysisId}/files
GET /api/v1/analyses/{analysisId}/files/detail?path=project/relative.go
GET /api/v1/analyses/{analysisId}/modules
GET /api/v1/analyses/{analysisId}/heatmap
GET /api/v1/analyses/{analysisId}/report
GET /api/v1/analyses/{analysisId}/export?format=json|csv
```

File and hotspot lists support bounded pagination, project-relative path/module filters, and deterministic sorting. Results and exports never expose absolute local paths. CSV output prefixes formula-leading path values before writing them to protect spreadsheet users. The full route/query/error contract is in [`../docs/api-contract.md`](../docs/api-contract.md).

During the current development phase, the API permits browser requests from every origin. This keeps a separately hosted local UI working without extra startup flags. The permissive CORS policy does not allow cross-origin credentials, but it must be replaced with a strict same-origin or explicit trusted-origin policy before a public release.

## Swagger / OpenAPI

The backend embeds Swagger UI 5.35.0 locally. After starting the server, open this address in a browser:

```text
http://127.0.0.1:8080/swagger/
```

Swagger UI, its JavaScript/CSS assets, and the OpenAPI definition are all served from the local backend; the page does not load documentation assets or specifications from an external service. The versioned OpenAPI 3.1 specification remains available at [`internal/api/openapi.yaml`](internal/api/openapi.yaml) and:

```bash
curl http://127.0.0.1:8080/api/v1/openapi.yaml
```

The UI documents health, direct analysis, persisted lifecycle, completed report views, JSON/CSV exports, and OpenAPI endpoints. The **Try it out** button calls the same local Go service.

## Test

```bash
go test ./...
```

The tests use real Go, JavaScript, and Python source fixtures, parser-error fixtures, temporary local files, direct handlers, and `httptest` HTTP servers. They never execute fixture source.

Use `go run ./cmd/server -addr 127.0.0.1:8081` to select another explicit local address.
