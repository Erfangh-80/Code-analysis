# Backend

Independent Go module for the local Code Risk Analyzer core API (phases 1–4). The service reads selected source files and an existing Git worktree from the same machine, parses them statically, and never executes target code. This branch has no persistence layer, analysis queue, report store, or export endpoints.

## Prerequisites

- Go 1.23 or later;
- Git, for history analysis;
- a C11 compiler such as GCC or Clang with CGO enabled, required by the pinned Tree-sitter language grammars.

## Run locally

```bash
cd backend
go run ./cmd/server
```

The server binds to `127.0.0.1:8080` by default. Startup confines analysis to the current user's home directory unless one or more explicit `--allowed-root` values are supplied. Use the narrowest parent directory that contains the projects you intend to analyze:

```bash
go run ./cmd/server \
  --allowed-root "$HOME/workspaces" \
  --max-files 10000 \
  --max-file-bytes $((2 * 1024 * 1024)) \
  --max-total-source-bytes $((128 * 1024 * 1024)) \
  --max-git-commits 20000 \
  --max-git-output-bytes $((16 * 1024 * 1024))
```

```bash
curl http://127.0.0.1:8080/api/v1/health
```

```json
{"status":"ok","version":"0.1.0-core"}
```

## Static-analysis API

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

Parser failures are stored as safe process-local metadata—never source content—and can be reviewed with:

```bash
curl http://127.0.0.1:8080/api/v1/static-analysis/failures
```

## Git-history analysis API

`POST /api/v1/git-analysis` reads the full reachable history of an existing local Git worktree by default. It uses fixed read-only Git commands, sets `GIT_OPTIONAL_LOCKS=0`, and does not execute repository code or modify the worktree. Optional inclusive RFC 3339 UTC `from` and `to` bounds limit the history window.

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

## Risk Engine API

`POST /api/v1/risk-analysis` calculates a deterministic, explainable hotspot score from already-collected per-file metrics. The engine is a separate `internal/scoring` package: it does not read files, invoke Git, serve HTTP, or depend on the frontend. The API layer only decodes the request and returns the engine's report.

With the default availability set, the input must contain every current signal for every repository-relative file:

- `linesOfCode`, `maxFunctionLength`, and `maxClassLength` from local static analysis;
- `gitChurn` and `changeFrequency` from local Git-history analysis.

A request may instead pass `availableMetrics`—for example the three static metrics when Git history was not analyzed. The engine then records the excluded metrics as unavailable and renormalizes positive weights across only the available set. Server-side helpers can join collected domain results with `application.BuildHotspotInputs`; the browser must only render the returned report and must never calculate a metric, normalization, risk score, band, rank, module aggregate, or heatmap coordinate.

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

The endpoint rejects missing, negative, unsafe, duplicate, or unknown metric fields.

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
- A path with no record in the selected Git history receives zero churn and change frequency **only when Git metrics were part of the availability set**. When Git analysis was not run, pass `availableMetrics` without the Git metrics so the engine renormalizes honestly instead of scoring fabricated zeros.

## Security, resource limits, and safety

The production composition root canonicalizes every `--allowed-root` and permits analysis only below those directories. Relative traversal, absolute paths supplied where relative paths are required, and symlinks escaping a selected root are rejected without returning absolute host paths to the client.

Static source reads are bounded by file count, per-file bytes, and total source bytes; Git analysis is bounded by commit count and retained command output. Exceeding a request-wide bound returns `413 RESOURCE_LIMIT_EXCEEDED`; file-specific static failures are returned as safe per-file diagnostics so valid selected files remain usable. Diagnostics and errors never include source content, parser-internal text, absolute project paths, or Git command output. Analysis creates no target-project temporary files, clones, or worktrees.

During the current development phase, the API permits browser requests from every origin without credentials. This keeps a separately hosted local UI working without extra startup flags, but the policy must be replaced with a strict same-origin or explicit trusted-origin policy before any public exposure.

See [`../docs/backend-performance.md`](../docs/backend-performance.md) for the limit table, host-side measurement procedure, and what is deliberately not claimed.

## Swagger / OpenAPI

The backend embeds Swagger UI locally. After starting the server, open this address in a browser:

```text
http://127.0.0.1:8080/swagger/
```

Swagger UI, its JavaScript/CSS assets, and the OpenAPI definition are all served from the local backend; the page does not load documentation assets or specifications from an external service. The versioned OpenAPI 3.1 specification remains available at [`internal/api/openapi.yaml`](internal/api/openapi.yaml) and:

```bash
curl http://127.0.0.1:8080/api/v1/openapi.yaml
```

The **Try it out** button calls the same local Go service.

## Test

```bash
go test ./...
```

The tests use real Go, JavaScript, and Python source fixtures, parser-error fixtures, temporary local files and Git worktrees, direct handlers, and `httptest` HTTP servers. They never execute fixture source.

Use `go run ./cmd/server -addr 127.0.0.1:8081` to select another explicit local address.
