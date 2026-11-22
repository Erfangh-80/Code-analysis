# Phase 1–4 QA and evaluation ledger

This ledger covers only the phase-only branch: acceptance criteria, direct local architecture/test data, static analysis, and Git-history Hotspot scoring. It does not claim validation for the removed persistence/report/dashboard/export product.

## Automated quality gate

```bash
# Backend: Go 1.23, Git and a C compiler with CGO available
cd backend
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
go test ./...

# Frontend
cd ../frontend
npm ci
npm run typecheck
npm test -- --run
npm run build

# API schema
cd ..
npx @redocly/cli@1.34.5 lint backend/internal/api/openapi.yaml
```

The repository's `Backend tests` GitHub Actions workflow runs `go test ./...` for backend changes. Arena's current sandbox does not provide a native Go binary, so local Go test/format output cannot be claimed here; use the workflow result or another Go-enabled host for that evidence.

### Current frontend evidence

On this phase-only working tree, the following commands passed:

- `npm run typecheck`
- `npm test -- --run` — 5 files, 18 tests passed
- `npm run build`

The tests cover relative health transport, direct static/Git/risk transport, static result rendering, nullable collection normalization, local-service failures, request timeout, and no browser-side score calculation.

## Acceptance coverage

| ID | Criterion | Automated evidence |
| --- | --- | --- |
| AC-01 | Direct API contains only health/OpenAPI/static/Git/risk surface; a missing route returns safe JSON `404`. | `internal/api/server_test.go`, `static_analysis_test.go` |
| AC-02 | Static analysis recognizes real Go, JavaScript, and Python fixtures and returns real declaration/LOC metrics. | `internal/analyzer/static/analyzer_test.go` |
| AC-03 | Static parser/read failures remain explicit and are safely observable without source content. | `internal/application/static_analysis_test.go`, `internal/api/static_analysis_test.go` |
| AC-04 | Static input is an absolute allowed root plus optional relative files; without files, bounded discovery skips VCS/vendor/build directories and symlink escapes, and honors the file-count cap. | `analysisguard/guard_test.go`, `application/static_analysis*_test.go`, `application/static_analysis_discovery_test.go`, `api/guarded_services_test.go` |
| AC-05 | Static file count, per-file bytes, and total bytes are bounded. | `application/static_analysis_limits_test.go`, `static_analysis_total_limit_test.go`, `api/guarded_services_limits_test.go` |
| AC-06 | Git facts come from an actual temporary worktree and read-only Git commands; history dates, rename/deletion, empty/shallow history, and path facts have defined behavior. | `githistory/analyzer_test.go`, `internal/api/git_analysis_test.go` |
| AC-07 | Git commit/output bounds are controlled. | `githistory/analyzer_limits_test.go` |
| AC-08 | Risk model validates inputs/profiles, normalizes deterministically, ranks ties, assigns bands, aggregates modules, and returns heatmap/contribution explanations. | `scoring/engine_test.go` |
| AC-09 | Missing Git availability is explicit: the scoring engine excludes unavailable metrics and renormalizes effective weights. | `scoring/engine_test.go`, `internal/api/risk_analysis_test.go` |
| AC-10 | Direct API maps malformed input, method errors, path errors, and resource limits to safe HTTP responses. | `internal/api/*_test.go`, `security_test.go` |
| AC-11 | OpenAPI and locally served Swagger remain available from the direct API. | `static_analysis_test.go`, `swagger_ui_test.go` |
| AC-12 | The frontend displays a local three-step workflow, needs only a root path for static analysis, takes Jalali dates with Gregorian transport, paginates result tables, and sends direct relative requests without fabricated results. | `frontend/src/app/app.test.tsx`, `features/core/pages/core-analysis-page.test.tsx`, `lib/datetime.test.ts`, `services/api/core-api.test.ts` |

## Manual smoke workflow

Run this on a machine that has Go, Git, a compiler, and a small local repository:

1. Start the backend with a deliberately narrow root:
   ```bash
   cd backend
   go run ./cmd/server -allowed-root "$HOME/workspaces"
   ```
2. Start the frontend in another terminal with `npm run dev` from `frontend/`.
3. In the workspace, submit a valid allowed root path. Confirm discovered file table rows and module count equal the backend response, and page through a result set larger than ten rows.
4. Submit the repository worktree to Git analysis with and without a Jalali date range. Confirm commit count, paths, and diagnostics match `git log`/`git diff-tree` expectations without changes to `git status`, that a reversed range is blocked before submission, and that an empty range result shows its explanatory warning.
5. Request Hotspot scoring. Confirm displayed rank/score/level/contributions are returned by `/risk-analysis`, not calculated in DevTools/browser code.
6. Try a relative/outside root and `../` file path. Confirm a generic error, no source text, and no absolute path leak.
7. Confirm browser keyboard access to labels, submit buttons, result tables, mobile navigation, theme button, and the privacy dialog.

Record operating system, Go/Node/Git versions, configured flags, fixture path shape, and observed errors. Do not use production repositories without authorization.

## Host-scale and model evaluation: not claimed

No full 1k/10k/100k/300k LOC benchmark, long-history stress result, peak RSS measurement, concurrent load measurement, or empirical defect-prediction validation is claimed for this branch.

The direct services contain file/byte/commit/output bounds, but those are safety controls—not proof of capacity. For a target host, record wall time, peak RSS (`/usr/bin/time -v`), selected file count/bytes, commit count, configured limits, and whether the analysis returned a controlled limit response.

The risk engine is deterministic and explainable, but it is not validated as a bug predictor. Any future evaluation must use authorized repositories, frozen commit ranges, predeclared labels, precision/recall or review criteria, and a separate held-out evaluation set. It must distinguish static-only runs from runs with available Git metrics.

## Explicitly out of scope for this ledger

This phase-only version has no recursive discovery, durable analysis IDs/stages, queue/cancellation behavior, SQLite/restart handling, stored reports, dashboards, result filtering, comparison, or JSON/CSV export. QA IDs or evidence for those features belong to the complete-product branch, not here.
