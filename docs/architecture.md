# Code Risk Analyzer — Architecture

**Status:** proposed baseline for approval

**Scope:** local/on-premises MVP for Go, JavaScript, and Python repositories

**Architecture principle:** static, explainable, local-only analysis

## 1. Goals and hard constraints

The product turns a local Git repository into an understandable, prioritized view of complexity and maintainability risk. It is designed to help engineers and technical managers decide where to review or refactor first. A risk score is deliberately not a prediction that a defect exists.

| Requirement | Architectural response |
| --- | --- |
| No cloud dependency or source upload | The UI, Go API, static analysis, Git inspection, result storage, and exports all run locally. Production binds to loopback by default. |
| Input is a local Git repository path | `POST /api/v1/analyses` accepts an absolute path on the backend host and validates that it resolves to a readable Git worktree. No clone URL is accepted. |
| Backend runs beside the analyzed repository | The Go process opens the local path itself. The browser never reads source files and never calls a remote service. |
| Full history by default; user-defined range | The request's optional `history` object narrows an otherwise unbounded commit traversal. Bounds are explicit and recorded with the run. |
| Go, JavaScript, and Python MVP | A language-adapter port isolates three static parsers behind one normalized metric model. Unsupported files are reported, never silently treated as analyzed. |
| Static only; never execute target code | Parsers read text/ASTs; Git is used in read-only mode. No build, test, import, package install, hook, script, or shell evaluation occurs in the target repository. |
| Separated backend concerns | HTTP API, orchestration, static analysis, Git history, scoring, storage, and domain contracts are distinct packages with dependency direction enforced by tests and review. |
| API first; React has no scoring logic | The versioned HTTP contract in [`api-contract.md`](api-contract.md) is the only data boundary the UI consumes. Scores and component contributions originate in Go. |

## 2. System context and local deployment

```text
+------------------------------- same user machine ------------------------------+
|                                                                              |
|  Browser                                                                    |
|  React UI                                                                    |
|     | relative /api requests only                                            |
|     v                                                                        |
|  Local Go service  <----- loopback HTTP ---->  local result store            |
|     |             (127.0.0.1 by default)                                     |
|     |                                                                        |
|     +-- read-only filesystem and Git access --> selected local Git repo     |
|           ├─ working-tree source files (static parsing)                       |
|           └─ commit history and numstat (Git metrics)                         |
|                                                                              |
+------------------------------------------------------------------------------+
```

The API process and target repository must be on the same machine. `repositoryPath` is a host-local absolute path; a path from another machine is invalid. The production UI is served from the local installation alongside the API. For development, the React dev server uses a proxy for relative `/api` requests—frontend code must never target `localhost` or an external hostname directly.

The service listens only on `127.0.0.1` / `::1` by default. A non-loopback bind, if ever needed for an internal network installation, will require an explicit opt-in and a clear warning because it expands the path-access trust boundary.

## 3. Component boundaries

### 3.1 Backend package layout

The future Go module is organized around a dependency-inward model:

```text
backend/
├── cmd/server/                   # composition root; flags; graceful shutdown
├── internal/
│   ├── domain/                   # immutable metric, run, score, and error types
│   ├── application/              # use cases: start analysis, query results, export
│   ├── repository/               # repository-path validation and file discovery
│   ├── analyzer/
│   │   ├── static/               # adapter port plus Go/JS/Python implementations
│   │   └── snapshot/             # read-only source snapshot and exclusions
│   ├── githistory/               # read-only commit/diff traversal and aggregation
│   ├── scoring/                  # normalization, profile validation, risk calculation
│   ├── storage/                  # local run/result persistence interface and adapter
│   ├── jobs/                     # bounded local execution and progress events
│   └── api/                      # HTTP routing, validation, DTO mapping, exports
└── testdata/                     # small real source and Git-history fixture repos
```

| Layer | Responsibilities | Must not do |
| --- | --- | --- |
| `domain` | Types, invariants, calculated-result vocabulary | Import HTTP, filesystem, Git, parser, or database code |
| `application` | Coordinate one analysis run and result queries through interfaces | Parse source, calculate raw Git diffs, or serialize HTTP |
| `repository` / `snapshot` | Validate and canonicalize local path; enumerate supported files safely | Execute target code or mutate its worktree |
| `analyzer/static` | Parse a file into raw static metrics | Inspect commit history, assign risk, or serve requests |
| `githistory` | Read commit history and aggregate per-path churn/frequency | Inspect source ASTs or assign risk |
| `scoring` | Normalize metric inputs and return score, band, and contributions | Read paths, invoke Git, or know HTTP query parameters |
| `storage` / `jobs` | Store runs locally; schedule and report bounded work | Reimplement analysis or presentation rules |
| `api` | Map the API contract to application calls and stream exports | Contain business rules or calculate metrics/scores |

The application layer owns joins between the static and Git views by repository-relative path. No layer may bypass the scoring port or return a frontend-specific calculated value.

### 3.2 Frontend package layout

```text
frontend/
├── app/                           # routes, layout, and screen composition
├── components/
│   ├── analysis-setup/            # local-path form and range controls
│   ├── dashboard/                 # heatmap, rank list, filters, detail panel
│   └── ui/                        # accessible presentational primitives
├── lib/
│   ├── api/                       # typed client generated/checked from API contract
│   └── formatting/                # display-only dates, labels, and number formatting
└── test/                          # component, accessibility, and browser tests
```

The UI may sort or filter result rows supplied by the API for interaction, but it may not derive LOC, complexity, churn, score, risk band, normalization, or weights. The backend is authoritative for all analytical values. Any displayed score must include the score model version returned by the API.

## 4. Analysis lifecycle and data flow

1. **Request validation** — The user submits an absolute local path, optional UTC history range, and a named scoring profile through `POST /api/v1/analyses`.
2. **Path safety** — The backend resolves symlinks/canonical path, verifies a readable Git worktree, rejects paths that are not local repositories, and records only a locally scoped run. It returns a job ID; it does not block the HTTP request on analysis completion.
3. **Snapshot and discovery** — A read-only snapshot enumerates candidate source files while applying documented exclusions (Git metadata, dependencies, generated/vendor folders, and configured exclusions). It retains only repository-relative paths in result data.
4. **Static metrics** — A language adapter parses each supported file without compiling, importing, building, or executing it. Parse failures are kept as per-file diagnostics; a file is never represented by invented metrics.
5. **Git metrics** — A read-only Git walker visits all reachable commits in the selected range, or all history when omitted. It aggregates changes by repository-relative file path. Renames are handled according to the documented Git-diff strategy and reported when continuity cannot be established.
6. **Join and score** — The application service joins static and history facts. The scoring service validates the selected profile, normalizes each metric in the run population, calculates contribution values and a `0..100` score, then assigns an explainable band.
7. **Persist and publish** — The job stores a local immutable result snapshot, exposes progress/status, and makes paginated files, heatmap cells, detail records, and exports available through the API.
8. **Present** — React obtains typed results from the API and renders them. It always shows scope, history window, analysis timestamp, supported/unsupported counts, and score-model version.

Analysis work runs in a bounded local worker pool. A cancellation request stops work at safe file/commit boundaries and marks the job accordingly; no partial result is labelled complete.

## 5. Metric definitions

Metrics are calculated per supported source file. The initial implementation must document language-specific edge cases alongside each parser.

| Metric | Definition | Static/Git source |
| --- | --- | --- |
| `loc` | Physical, non-blank, non-comment source lines according to the language adapter. Total physical lines may be reported separately. | Static parser/tokenizer |
| `cyclomaticComplexity` | Sum of function/method decision complexity. A function starts at 1; each language-defined branch/decision construct adds 1. File value is the sum and function values remain inspectable. | AST |
| `functionLength` | Non-blank source lines spanned by each function/method; file summary includes maximum and aggregate. | AST locations + line classifier |
| `classLength` | Non-blank source lines spanned by each class/type declaration where meaningful; Go type declarations are reported consistently with the documented adapter rule. | AST locations + line classifier |
| `gitChurn` | Sum of additions plus deletions affecting the path across selected commits. | `git log`/diff numstat read-only traversal |
| `changeFrequency` | Number of distinct selected commits that changed the path at least once. | Git history traversal |

Language adapters use the Go standard parser for Go and pinned, locally installed AST grammars for JavaScript and Python. A parse error yields `analysisStatus: "parse_error"` with a diagnostic, rather than a zero complexity value. The MVP may distinguish JavaScript from TypeScript in discovery, but TypeScript is not claimed supported until a dedicated adapter is implemented and tested.

## 6. Risk calculation

Risk is configurable but deterministic and reproducible. A profile includes a unique version, enabled metrics, non-negative weights that sum to `1.0`, normalization rules, and band thresholds.

For each enabled metric `m` of file `f`, the scoring service calculates a bounded normalized value `n(m,f)` in `[0,1]` from the selected run population. The initial profile uses capped percentile normalization so an extreme outlier cannot make all other files appear risk-free. The formula is:

```text
risk(f) = round(100 × Σ(weight(m) × n(m,f)))
```

Default bands are `low` (0–34), `medium` (35–64), `high` (65–79), and `critical` (80–100). Exact defaults are delivered as a local profile file and are returned in every result. The API also returns one contribution per metric so that a score is explainable. Missing or unavailable metrics are explicit: the run either uses the profile's defined missing-data policy or fails validation; it never quietly substitutes fabricated data.

Changing a profile creates a new model version and requires a new analysis run. Existing run results remain immutable and retain the profile snapshot used to calculate them.

## 7. API, data, and exports

The detailed versioned HTTP contract is in [`api-contract.md`](api-contract.md). The first complete backend implementation must satisfy it before the dashboard is implemented.

Primary domain records are:

- **AnalysisRun:** ID, local scope metadata, requested/effective history range, status, timestamps, diagnostics, and scoring profile snapshot.
- **FileAnalysis:** opaque file ID, repository-relative path, language, analysis status, raw static/Git metrics, calculated score/band, and score contributions.
- **HeatmapCell:** file reference plus two selected backend metric dimensions and the backend-calculated risk result.
- **Diagnostic:** structured scope, code, severity, safe message, and optional repository-relative path. Absolute host paths are not exposed in exports by default.

Results are stored only on local disk. Storage begins behind an interface so the MVP can use a simple local embedded store without contaminating domain logic. Retention, deletion, and local storage location will be explicit configuration; the first implementation must provide deletion of a completed run.

JSON export mirrors the immutable result schema. CSV export is a flattened file-level view with documented headers and UTC timestamps. Export is generated by the API from stored results, not reconstructed in the browser.

## 8. Security, privacy, and failure behavior

- **No network analysis:** no telemetry, external API, cloud storage, remote clone, or source upload is a dependency of analysis.
- **Path handling:** reject empty, relative, inaccessible, non-Git, and symlink-escaping paths after canonicalization. Return a safe error without leaking unrelated filesystem paths.
- **Git handling:** invoke only fixed, read-only Git commands with argument arrays—never a shell string. Do not checkout, reset, clean, commit, fetch, pull, push, install hooks, or execute repository scripts. Prefer plumbing/history operations that do not invoke text filters.
- **Resource controls:** cap concurrent jobs, file count, individual file size, traversal time, and stored run size. Surface a precise incomplete/cancelled/failed status when a limit is reached.
- **Untrusted source:** treat source and Git metadata as untrusted bytes. Parsers are isolated behind recoverable errors; names and diagnostics are escaped before UI display and CSV output is protected against spreadsheet formula injection.
- **No invented success:** parse errors, unsupported files, unavailable history, and cancellations are visible in status and diagnostics. Only a fully stored, successfully finalized job may be `completed`.

## 9. UX architecture

The first dashboard has a deliberately narrow, high-confidence flow:

1. **Analyze repository:** clear local-path field, history-range disclosure, scoring profile selector, validation feedback, and an explicit local-only notice.
2. **Progress:** deterministic stages (validate, discover, statically analyze, read Git history, score, store) with counts, cancellation, and actionable errors.
3. **Overview:** summary cards, risk-band distribution, two-axis heatmap, ranked high-risk files, and visible analysis scope/model version.
4. **Explore:** full-text path search, language/risk/status filters, sort, compare selected files, and a detail panel with raw metrics plus contribution explanation.
5. **Export:** JSON/CSV controls that describe the exact completed run and scope being exported.

The UI follows the mandatory standards in [`project-skills.md`](project-skills.md): keyboard-first operation, semantic structures, visible focus, non-color risk encodings, text alternatives for the heatmap, responsive data-density controls, meaningful empty/loading/error states, and WCAG 2.2 AA as the minimum accessibility target.

## 10. Test strategy and delivery gates

Development follows test-driven development. Every behavior starts with a failing test, proceeds through the smallest production implementation, and is refactored only while tests remain green.

| Scope | Evidence required |
| --- | --- |
| Domain and scoring | Unit tests for boundaries, weights, normalization, bands, missing data, and deterministic output |
| Static adapters | Real language source fixtures that verify metric values and parse-error behavior without executing fixture code |
| Git history | Temporary real Git repositories with actual commits, additions, deletions, time bounds, and rename cases |
| Application/API | Integration tests that submit a real local fixture path and verify persisted status/results against the API contract |
| Security/resource behavior | Tests for invalid paths, non-repositories, traversal/symlink constraints, unsupported files, cancelled jobs, limits, and no-write behavior |
| Frontend | Component and accessibility tests plus browser tests against an implemented local backend fixture—not mock analytics data |
| Contract and export | Contract tests for every endpoint and schema; CSV/JSON assertions against the same stored real run |

A capability cannot be marked complete with a mock or fabricated result. Test fixtures are allowed only when they are real source files and real Git histories whose expected metrics are independently asserted.

## 11. Deferred decisions

These are deliberately not implemented or claimed as MVP features: remote repository cloning, SaaS/multi-tenant deployment, authentication, source-code upload, runtime/dynamic analysis, defect prediction, TypeScript support, and distributed processing. Each requires a separate approved design change.
