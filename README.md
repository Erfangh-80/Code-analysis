# Code Risk Analyzer

A local-first system for identifying software files that deserve earlier review or refactoring attention. Given the **local path** to a valid Git repository, it will statically analyze Go, JavaScript, and Python source and its Git history. It will combine code-complexity and change-history metrics into an explainable, configurable risk score.

> **Project status — runnable foundation**
>
> Both independent applications now start locally: a Go API with a tested health endpoint and a Next.js + TypeScript foundation UI. Repository analysis, Git metrics, risk calculation, dashboard data, and exports are intentionally not implemented yet and are not represented by mock data.

## What the product will do

- Analyze a repository that already exists on the same machine as the backend.
- Inspect source **statically**; it will never execute the target repository's code.
- Analyze the complete Git history by default, with optional inclusive start and end times.
- Support Go, JavaScript, and Python in the MVP.
- Collect per-file metrics including:
  - cyclomatic complexity;
  - lines of code (LOC);
  - function and class length;
  - Git churn (added plus deleted lines); and
  - change frequency (distinct commits affecting a file).
- Normalize and combine those metrics with a versioned, locally configurable scoring profile.
- Present a dashboard with a two-dimensional heatmap, ranked files, metric details, search and filters, comparisons, and JSON/CSV export.

A risk score is a prioritization aid for review and refactoring—not a claim that a file contains a bug.

## Local-only and safety guarantees

- The Go backend runs on the machine that holds the repository being analyzed.
- The browser sends a repository path only to that local backend over loopback; source code and repository paths are never submitted to a cloud service.
- The system accepts an existing local Git repository path. It does not clone repositories or execute target code.
- Git inspection will be read-only. Analysis will never modify the target repository, its worktree, index, commits, branches, or remote configuration.
- The frontend only renders API data and user-selected filters. It does not compute metrics or risk scores.

## Project structure

```text
.
├── backend/                 # Independent Go module and local API service
│   ├── cmd/server/          # Composition root and graceful HTTP server
│   └── internal/api/        # Tested local HTTP boundary
├── frontend/                # Independent Next.js + TypeScript application
│   ├── app/                 # App Router layout and foundation screen
│   └── test/                # Vitest and Testing Library setup
├── docs/
│   ├── architecture.md      # Component, data-flow, and deployment design
│   ├── api-contract.md      # Versioned HTTP contract before dashboard work
│   └── project-skills.md    # Mandatory TDD and UI/UX standards
└── README.md
```

`backend/` and `frontend/` have independent dependency manifests and test commands. The browser only calls relative API paths; Next.js proxies `/api/*` to the local Go process on `127.0.0.1:8080` during development.

## Prerequisites

| Dependency | Required version | Purpose |
| --- | --- | --- |
| Git | 2.30+ | Read a local repository's commit history and diffs |
| Go | 1.23+ | Build and test the backend |
| Node.js | 22.12+ | Build and test the Next.js frontend |
| npm | Bundled with Node.js | Install locked frontend dependencies |
| A local Git repository | Any valid repository path | Future analysis input; it is never uploaded |

## Run locally

Start the two local processes in separate terminals.

**Terminal 1 — Go API**

```bash
cd backend
go run ./cmd/server
```

The API binds to `127.0.0.1:8080`. Verify the implemented health capability:

```bash
curl http://127.0.0.1:8080/api/v1/health
# {"status":"ok","version":"0.1.0"}
```

**Terminal 2 — Next.js UI**

```bash
cd frontend
npm install
npm run dev
```

Open the URL Next.js prints (normally `http://localhost:3000`). The foundation page contains no fabricated analysis data. Its **Check local API health** link uses a relative route and is proxied by Next.js to the Go service.

## Test and build

```bash
# Backend
cd backend
go test ./...

# Frontend
cd ../frontend
npm run typecheck
npm test
npm run build
```

## Implemented now vs. next

| Capability | Status |
| --- | --- |
| Local Go service startup and graceful shutdown | Implemented |
| `GET /api/v1/health` API contract endpoint | Implemented and unit-tested |
| Next.js TypeScript application startup | Implemented |
| Relative API proxy to the same-machine Go API | Implemented |
| Foundation UI with local-only boundary messaging | Implemented and component-tested |
| Repository path validation, static metrics, Git history, scoring, and analysis jobs | Next backend increment |
| Dashboard, search/filter/compare, heatmap, and export | After implemented backend data exists |

## Architecture and engineering standards

Read these documents before implementation work:

- [Overall architecture](docs/architecture.md) — boundaries, data flow, metrics, scoring, security, testing, and UX direction.
- [API contract](docs/api-contract.md) — endpoint behavior and schemas that backend work must satisfy before dashboard work begins.
- [Project skills](docs/project-skills.md) — mandatory TDD and UI/UX Pro Max practices.

A stage is not complete until its real implementation, tests, and results have been reviewed.

## License

A license has not yet been selected.
