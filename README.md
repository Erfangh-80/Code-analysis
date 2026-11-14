# Code Risk Analyzer

A local-first system for identifying software files that deserve earlier review or refactoring attention. Given the **local path** to a valid Git repository, it statically analyzes Go, JavaScript, and Python source and its Git history. It combines code-complexity and change-history metrics into an explainable, configurable risk score.

> **Project status — architecture foundation**
>
> The repository currently contains the approved project structure, architecture, API contract, and engineering standards only. There is not yet an executable backend or frontend, so no analysis command or dashboard is being presented as available. Implementation begins only after the architecture stage is approved.

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
- The browser sends the repository path only to that local backend over loopback; source code and repository paths are never submitted to a cloud service.
- The system accepts an existing local Git repository path. It does not clone repositories or execute target code.
- Git inspection is read-only. Analysis never modifies the target repository, its worktree, index, commits, branches, or remote configuration.
- The frontend only renders API data and user-selected filters. It does not compute metrics or risk scores.

## Planned project structure

```text
.
├── backend/                 # Independent Go module and local API service (implementation stage)
├── frontend/                # Independent React + TypeScript application (implementation stage)
├── docs/
│   ├── architecture.md      # Approved component, data-flow, and deployment design
│   ├── api-contract.md      # Versioned HTTP contract to implement before the UI
│   └── project-skills.md    # Mandatory TDD and UI/UX standards
└── README.md
```

`backend/` and `frontend/` are deliberately independent folders. They will have their own dependency manifests, test commands, and development tooling. The frontend may call only the local backend API; it must not read the analyzed repository directly.

## Prerequisites for the implemented product

The implementation will target the following local prerequisites:

| Dependency | Planned minimum | Purpose |
| --- | --- | --- |
| Git | 2.30+ | Read a local repository's commit history and diffs |
| Go | 1.23+ | Build and test the backend and static-analysis adapters |
| Node.js | 22 LTS+ | Build and test the React frontend |
| npm | Bundled with Node.js | Install pinned frontend dependencies |
| A local Git repository | Any valid repository path | Analysis input; it is never uploaded |

Exact supported versions will be pinned in the backend and frontend manifests when those modules are created.

## How to run

There is no runnable application at this architecture-only stage. This is intentional: no command below is a placeholder for a capability that has not been implemented.

After the backend and frontend foundations are implemented and tested in later tasks, this section will contain only verified commands for:

1. starting the local Go API server;
2. starting the React development UI or serving its production build locally;
3. submitting an absolute path to a local Git repository; and
4. running the full test suite.

The intended production topology is one local process or local installation: the React build is served on the same machine as the Go API, which binds to loopback by default. During development, the frontend development server will proxy relative `/api` requests to that local API. Neither topology permits a browser-facing call to a remote analysis service.

## Architecture and API contract

Read these documents before implementation work:

- [Overall architecture](docs/architecture.md) — boundaries, data flow, metrics, scoring, security, testing, and UI design direction.
- [API contract](docs/api-contract.md) — endpoint behavior and schemas that the backend must implement before dashboard work begins.
- [Project skills](docs/project-skills.md) — the mandatory TDD and UI/UX Pro Max practices.

## Delivery sequence

The project will move in explicit, reviewable increments:

1. **Architecture foundation** — this change: folders, README, architecture, API contract, and standards.
2. **Backend foundation** — Go module, domain types, configuration, API shell, and red tests followed by tested implementation.
3. **Static analysis adapters** — real Go, JavaScript, and Python metric extraction against fixture repositories.
4. **Git-history analysis and scoring** — real history metrics, configurable score calculation, and API integration.
5. **Frontend foundation** — React UI consuming the implemented API contract.
6. **Dashboard and exports** — heatmap, ranking, filtering, comparison, JSON/CSV export, accessibility, and end-to-end tests.

A stage is not complete until its real implementation, tests, and results have been reviewed.

## License

A license has not yet been selected.
