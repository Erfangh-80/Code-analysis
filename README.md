# Code Risk Analyzer — phases 1–4

This branch is the phase-only deliverable for the first half of the project. It is a local-first workspace that reads selected source files and an existing Git worktree, then returns an explainable Hotspot ranking.

> The complete, later product—including repository discovery, durable analysis runs, SQLite persistence, reports, dashboards, filters, and exports—intentionally remains on `main` and `output-3-project`. It is not part of this branch.

## Delivered phases

| Phase | Delivered outcome | Acceptance evidence |
| --- | --- | --- |
| 1. تحلیل نیازمندی‌ها و طراحی معیارهای پذیرش | Local-only, static-only and read-only-Git boundaries; supported languages, metrics, score limitations, and API acceptance criteria are documented. | [`docs/architecture.md`](docs/architecture.md), [`docs/api-contract.md`](docs/api-contract.md), focused API tests |
| 2. طراحی معماری و آماده‌سازی داده آزمون | A bounded Go composition root, direct HTTP contract, parser fixtures, temporary local files, and temporary Git repositories establish the testable architecture. | package tests under `backend/internal`, embedded OpenAPI 3.1 document |
| 3. پیاده‌سازی هسته تحلیل ایستا با Go برای Python، Go و JavaScript | The backend statically parses explicitly named `.go`, `.js`/`.mjs`/`.cjs`, and `.py` files and returns file/module metrics and safe diagnostics. | `analyzer/static`, `application/static_analysis`, `/static-analysis` tests |
| 4. پیاده‌سازی تحلیل تاریخچه Git و Score Hotspot | The backend reads existing Git history without modifying it and calculates deterministic, availability-aware `risk-v1` Hotspot scores. | `githistory`, `scoring`, `/git-analysis` and `/risk-analysis` tests |

The React + TypeScript frontend is included for these same phases. Its one workspace calls the direct local API, displays static and Git results, and requests the backend-owned risk calculation; it does not fabricate dashboard data or calculate scores in the browser.

## Scope and non-goals

### In scope

- Local paths only, constrained by operator-configured allowed roots.
- Static parsing only; the target project is never built, imported, tested, or executed.
- Explicit file selection for Go, JavaScript, and Python.
- Read-only Git history analysis for an existing worktree, with optional date bounds.
- Deterministic Hotspot scores, ranking, module aggregates, heatmap data, metric contributions, and limitations.
- A Persian RTL frontend for the direct three-step workflow.

### Deliberately absent on this branch

- Recursive repository scanning/discovery and `.gitignore` processing.
- Queued or asynchronous analysis lifecycle, cancellation, persistence, SQLite, or restart recovery.
- Saved report/result query routes, dashboard pages, server-side filters, comparisons, CSV/JSON export, and settings screens.
- Remote repositories, cloning, source upload, target-code execution, TypeScript parsing, defect prediction, coverage, security scanning, or cyclomatic complexity.

A score is a review/refactoring priority signal, **not** a claim that a file contains a defect.

## Architecture at a glance

```text
Browser (React/Vite, relative /api/v1 requests)
                 │
                 ▼
Go local HTTP API
  ├── static analysis: selected files → metrics/diagnostics
  ├── Git history: existing worktree → per-file change facts
  └── risk engine: supplied analytical facts → Hotspot report
```

The development server proxies relative `/api/...` requests to the Go API on the same machine. The browser never calls `localhost` itself and no source is sent to a cloud service.

## Run locally

### Prerequisites

| Dependency | Version | Why |
| --- | --- | --- |
| Go | 1.23+ | Build and test the local backend |
| A C compiler with CGO | GCC or Clang | Build pinned Tree-sitter grammars |
| Git | 2.30+ | Read local commit history and diffs |
| Node.js / npm | Node 22+ recommended | Build and test the frontend |

Start the backend, limiting it to the smallest parent directory that holds projects you plan to inspect:

```bash
cd backend
go run ./cmd/server -allowed-root "$HOME/workspaces"
```

Useful direct-analysis limits are `-max-files`, `-max-file-bytes`, `-max-total-source-bytes`, `-max-git-commits`, and `-max-git-output-bytes`. The server listens on `127.0.0.1:8080` unless `-addr` is supplied.

In another terminal, start the frontend:

```bash
cd frontend
npm install
npm run dev
```

Open the Vite URL, enter an allowed absolute local path and relative source files, then optionally analyze its Git history and request a Hotspot score.

## Direct API surface

The complete machine-readable contract is embedded at [`backend/internal/api/openapi.yaml`](backend/internal/api/openapi.yaml).

| Method | Route | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/health` | Local service status/version |
| `GET` | `/api/v1/openapi.yaml` | Embedded OpenAPI 3.1 document |
| `POST` | `/api/v1/static-analysis` | Analyze explicitly selected relative source files |
| `GET` | `/api/v1/static-analysis/failures` | Process-local safe parser/read failure metadata |
| `POST` | `/api/v1/git-analysis` | Read local Git history facts |
| `POST` | `/api/v1/risk-analysis` | Calculate backend-owned Hotspot scores |
| `GET` | `/swagger/` | Locally served Swagger UI |

See [`docs/api-contract.md`](docs/api-contract.md) for examples, safety rules, error behavior, and scoring semantics.

## Verify

```bash
# Go backend
cd backend
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
go test ./...

# React frontend
cd ../frontend
npm ci
npm run typecheck
npm test -- --run
npm run build

# OpenAPI document
cd ..
npx @redocly/cli@1.34.5 lint backend/internal/api/openapi.yaml
```

The repository CI runs `go test ./...` for backend changes. The current Arena sandbox does not include a native Go toolchain, so Go test results must come from a Go-enabled environment or the GitHub Actions check.

## Further reading

- [Architecture and acceptance criteria](docs/architecture.md)
- [Direct local API contract](docs/api-contract.md)
- [Phase-only QA evidence and remaining measurements](docs/qa-final-evaluation.md)
- [Backend usage and limits](backend/README.md)
- [Frontend workspace usage](frontend/README.md)
- [Engineering standards](docs/project-skills.md)

## License

A license has not yet been selected.
