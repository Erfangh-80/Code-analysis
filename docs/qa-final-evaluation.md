# Integration QA and final evaluation

This is the evidence ledger for the backend–frontend integration and final QA phase. It separates executable coverage from host-dependent measurements and model evaluation. A check is **not** treated as passed merely because a mock can render a screen.

## How to run the automated evidence

```bash
# Backend (Go 1.23 with CGO and Git available)
cd backend
go test ./...

# Frontend
cd ../frontend
npm ci
npm run typecheck
npm test -- --run
npm run build
npm audit --omit=dev --audit-level=high

# API contract
cd ..
npx @redocly/cli@1.34.5 lint backend/internal/api/openapi.yaml
```

The backend GitHub Actions workflow runs `go test ./...` on supported pushes and pull requests. The Arena development environment used for this change has no native Go toolchain, so Go verification must be read from that workflow rather than claimed as a local result.

## Integration acceptance evidence

| ID | Implemented integration | Automated evidence |
| --- | --- | --- |
| INT-01 | The analysis form posts only a local `projectPath` to `POST /analyses/validate-path` before it can start a run. The backend resolves the canonical directory under the configured allowed roots and returns only `{ "valid": true }`. | `path_validation_test.go`; `new-analysis-page.test.tsx`; `new-analysis-page.integration.test.tsx` |
| INT-02 | On successful preflight, the typed React Query mutation posts the selected path, history range, and risk profile to `POST /analyses`, then navigates to the returned analysis ID. Start repeats validation, so preflight is never a security reservation. | `analysis_orchestrator_test.go`; `new-analysis-page.test.tsx` |
| INT-03 | The detail route polls the persisted lifecycle endpoint while status is `queued` or `running`, and displays backend stages/progress/cancellation state. | `use-analysis-queries.ts`; `analysis_orchestrator_test.go` |
| INT-04 | Completed detail/report routes retrieve the saved summary; they do not recalculate counts in the browser. | `result_queries_test.go`; `analysis_results_test.go` |
| INT-05 | The heatmap reads backend-saved axes/cells from `/heatmap`; unavailable Git metrics are not fabricated in static-only reports. | `engine_test.go`; `analysis_results_test.go` |
| INT-06 | The files table retrieves the paginated joined static/Git/risk file view from `/files`. | `analysis_results_test.go` |
| INT-07 | Search, language, risk level, score bounds, static status, deterministic sort, and pagination are serialized as `/files` query parameters and filtered in the application layer against the immutable report. | `result_queries_test.go`; `analysis_results_test.go`; `analysis-api.test.ts` |
| INT-08 | File detail uses its project-relative path query and renders the API-provided static, optional Git, diagnostic, and contribution data. | `analysis_results_test.go`; `file-detail-page.test.tsx` |
| INT-09 | JSON/CSV export calls `/export?format=...`, uses the attachment filename, and exposes a safe failure message without leaving the report route. | `export_test.go`; `analysis_results_test.go`; `report-export-card.test.tsx` |
| INT-10 | Go API tests validate immutable result views and malformed queries; the frontend request boundary normalizes nullable Go array fields before components consume them. | `analysis_results_test.go`; `result_queries_test.go`; `client.test.ts`; `file-detail-page.test.tsx` |
| INT-11 | The fetch boundary converts timeout and unreachable-service failures to retryable, human-readable local-service errors. The start form keeps the user on the form and does not post a run if preflight fails. | `client.test.ts`; `new-analysis-page.integration.test.tsx` |
| INT-12 | A real `httptest` HTTP flow validates a fixture path, starts it, polls terminal state, and reads summary, hotspots, files, one file, modules, heatmap, report, JSON export, and CSV export. | `TestFullAnalysisFlowOverRealHTTPValidatesStartsPollsAndReadsResults` |

## QA-01–QA-15: analysis and API correctness

| ID | Scenario and current evidence | Status |
| --- | --- | --- |
| QA-01 | Go, JavaScript, and Python declarations and language recognition. | Automated: `analyzer_test.go` |
| QA-02 | LOC, function/class counts, and max/total declaration lengths are asserted from fixtures. | Automated: `analyzer_test.go` |
| QA-03 | Read-only Git churn, frequency, authors, first/last changes, concentration, and dispersion. | Automated: `githistory/analyzer_test.go` |
| QA-04 | Hotspot scores, bands, normalization caps, and contribution weights. | Automated: `scoring/engine_test.go` |
| QA-05 | Ranking, deterministic result ordering, module aggregate, and heatmap-cell correctness. | Automated: `engine_test.go`; `result_queries_test.go` |
| QA-06 | Saved API summary/files/hotspots/heatmap/export are coherent views of one immutable report. | Automated: `analysis_results_test.go` |
| QA-07 | Syntax error uses fallback only when possible and retains a `PARSE_ERROR`; the deferred parser-quality investigation is not reopened here. | Automated: `analyzer_test.go` |
| QA-08 | Empty, comment-only/no-executable, and unsupported inputs retain honest status/metrics. | Automated: `analyzer_test.go`; `orchestrator_test.go` |
| QA-09 | A non-Git project completes static analysis and static-only scoring; Git is unavailable rather than zero-filled. | Automated: `orchestrator_test.go`; `engine_test.go` |
| QA-10 | Relative, malformed, unsafe, and invalid file/path inputs return safe validation errors. | Automated: `security_test.go`; `guarded_services_test.go`; `analysis_results_test.go`; `path_validation_test.go` |
| QA-11 | A guarded scanner rejects a directory outside configured roots; the validation endpoint does not disclose that absolute path. | Automated: `scanner_limits_test.go`; `path_validation_test.go` |
| QA-12 | Git deletion is retained as a deleted-history fact rather than a discovered working-tree source file. | Automated: `githistory/analyzer_test.go` |
| QA-13 | Git rename maps the current path to the prior project-relative path. | Automated: `githistory/analyzer_test.go` |
| QA-14 | Nullable arrays, absent optional artifacts, non-completed report access, and malformed result queries have explicit behavior. | Automated: `client.test.ts`; `file-detail-page.test.tsx`; `analysis_results_test.go` |
| QA-15 | Completed reports survive SQLite reopen; duplicate active scopes and interrupted runs have controlled lifecycle behavior. | Automated: `infrastructure/sqlite/store_test.go`; `analysis_orchestrator_test.go` |

## QA-16–QA-24: UI behavior

| ID | Scenario and current evidence | Status |
| --- | --- | --- |
| QA-16 | Local path entry rejects repository URLs client-side and performs API preflight before start. | Automated: `new-analysis-page.test.tsx` |
| QA-17 | Active lifecycle polling and progress presentation consume actual analysis state. | Implemented and backend flow-tested; manual browser observation remains advisable. |
| QA-18 | Completion enables API-backed summary, hotspots, report, and files links. | Implemented; `analysis_results_test.go` validates payloads. |
| QA-19 | Path search is sent as `q` to the files API. | Automated transport coverage: `analysis-api.test.ts` |
| QA-20 | Language, risk level, score range, static status, sorting, and pagination are server-side `/files` filters. | Automated: `result_queries_test.go`; `analysis_results_test.go`; `analysis-api.test.ts` |
| QA-21 | Heatmap renders saved backend coordinates and links to a project-relative file detail view. | Implemented; backend payload asserted in `analysis_results_test.go`. |
| QA-22 | File detail has loading/error/empty states and handles incomplete collection fields. | Automated: `file-detail-page.test.tsx` |
| QA-23 | JSON and CSV download use the export API and surface download failures in place. | Automated: `report-export-card.test.tsx`; `export_test.go` |
| QA-24 | Timeout/unreachable errors are actionable; responsive Tailwind breakpoints are implemented. Test viewport combinations manually in a browser before release. | Automated network branch: `client.test.ts`; `new-analysis-page.integration.test.tsx`. Responsive visual validation: manual. |

## QA-25–QA-34: scale, duration, memory, and parallelism

No 1k/10k/100k/300k LOC or long-history benchmark result is claimed in this repository. The application contains resource boundaries, persisted stage timings, concurrency control, and controlled degradation, but those are safeguards rather than a capacity measurement.

| ID | Required evaluation | Evidence now | What must be recorded on the target host/CI worker |
| --- | --- | --- | --- |
| QA-25 | ~1k LOC | Functional fixture tests only. | Wall time, peak RSS, static/Git durations, result status. |
| QA-26 | ~10k LOC | File and byte limits are automated. | Same metrics and configured limits. |
| QA-27 | ~100k LOC | File/byte/Git output bounds are automated. | Same metrics; identify any controlled skips. |
| QA-28 | ~300k LOC | Intentionally unmeasured; may exceed the default 128 MiB source budget. | Same metrics; record bounded partial behavior rather than forcing completion. |
| QA-29 | Static-analysis duration | Static stage timestamps/duration are persisted and tested. | Compare small/medium/large fixtures. |
| QA-30 | Git-analysis duration | Git stage timestamps/duration are persisted and tested. | Compare shallow/short and long histories. |
| QA-31 | Memory | Source and Git output are bounded; no native RSS measurement in this environment. | `/usr/bin/time -v` peak resident set size. |
| QA-32 | Long Git history | Commit/output budget handling is automated. | Commit count, duration, and controlled-unavailable result when budget is crossed. |
| QA-33 | Parallel analyses | Bounded run slots and independent static/Git stage execution are implemented. | Run at configured concurrency; record queue delay, RSS, and correctness. |
| QA-34 | Limit behavior | File count/size, total source bytes, Git commit/output, and timeout branches are automated. | Confirm configuration is suitable for deployment; do not raise limits without repeating QA-25–33. |

Use the reproducible commands and fixture shapes in [`backend-performance.md`](backend-performance.md). Record hardware, OS, Go version, fixture commit count, flags, wall time, peak RSS, static duration, Git duration, and diagnostics. A result from another machine is not a release capacity guarantee.

## QA-35–QA-40: risk-model evaluation

The model is deterministic and explainable, but it is **not** a bug predictor validated against real-world defect history in this change. The risk report itself exposes formula, normalization, contributions, requested/effective weights, and limitations to support this evaluation.

| ID | Required evaluation | Current evidence / limitation |
| --- | --- | --- |
| QA-35 | Run on representative real repositories. | Not executed here; privacy, repository selection, and an agreed ground truth are required. |
| QA-36 | Compare rankings to commits/messages containing `fix`, `bug`, or `resolve`. | Not executed; keyword labels are only an imperfect proxy for defects. |
| QA-37 | Review false positives. | Not measured; high structural/change risk does not prove a defect. |
| QA-38 | Review false negatives. | Not measured; the model has no runtime, test, coverage, security, or semantic analysis input. |
| QA-39 | Evaluate/tune formula weights. | Unit tests prove configured/effective-weight semantics, not empirical optimality. Tune only against a held-out labelled corpus. |
| QA-40 | Publish findings and limitations. | Formula/normalization/limitations are persisted in reports; this ledger documents the missing empirical evaluation rather than fabricating it. |

### Recommended empirical protocol

1. Choose public or explicitly authorized repositories in the supported languages; freeze commit SHAs and record path scope/history range.
2. Define labels before looking at scores. If commit-message heuristics (`fix`, `bug`, `resolve`) are used, retain the matched commit and manually sample labels because wording is noisy.
3. For each analysis snapshot, compare top-*k* hotspots against subsequent labelled fixes. Record precision@k, recall@k, false positives, false negatives, and the review window.
4. Keep projects/time windows separate for training/tuning versus final evaluation. Preserve the requested profile and effective profile so static-only runs are not compared as though Git metrics existed.
5. Publish anonymized aggregate findings, dataset criteria, false-positive/negative examples where authorized, and the model limitations above. Do not derive claims of predictive accuracy from the deterministic unit fixtures.
