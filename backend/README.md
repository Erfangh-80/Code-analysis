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

## Swagger / OpenAPI

The backend embeds Swagger UI 5.35.0 locally. After starting the server, open this address in a browser:

```text
http://127.0.0.1:8080/swagger/
```

Swagger UI, its JavaScript/CSS assets, and the OpenAPI definition are all served from the local backend; the page does not load documentation assets or specifications from an external service. The versioned OpenAPI 3.1 specification remains available at [`internal/api/openapi.yaml`](internal/api/openapi.yaml) and:

```bash
curl http://127.0.0.1:8080/api/v1/openapi.yaml
```

The UI documents the health, static-analysis, failure-record, and OpenAPI endpoints. The **Try it out** button calls the same local Go service.

## Test

```bash
go test ./...
```

The tests use real Go, JavaScript, and Python source fixtures, parser-error fixtures, temporary local files, and HTTP handlers. They never execute fixture source.

Use `go run ./cmd/server -addr 127.0.0.1:8081` to select another explicit local address.
