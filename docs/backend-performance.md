# Backend security, resource, and performance report

This document records the operating envelope for the phase-1–4 local Code Risk Analyzer backend: the direct static-analysis, Git-history, and risk-scoring services. It is intentionally explicit about what is measured, bounded, and not cached. The analyzer never executes target-project code.

## Security boundary

The production server accepts only absolute local paths below one or more canonical operator-configured `--allowed-root` directories. It resolves symbolic links before evaluating the boundary. A root or repository path outside a configured root is rejected without returning an absolute host path to the client.

The same policy is injected into both filesystem-facing production services:

- the direct static-analysis service; and
- the direct Git-history service.

A source-file path must be relative to its selected root. Cleaned `..`, absolute paths, and symbolic links resolving outside the root are rejected. A Git worktree outside the configured roots is rejected as invalid input rather than read.

Diagnostics and HTTP errors contain only stable codes, generic messages, and project-relative paths. Source bytes, Git command output, absolute project paths, and parser-internal text are neither logged nor returned.

## Production defaults and overrides

The server uses the following conservative defaults. All can be tightened at startup; values that are zero, negative, or internally inconsistent are rejected.

| Limit | Default | Flag | Controlled behavior |
| --- | ---: | --- | --- |
| Allowed roots | Current user home directory | `--allowed-root` (repeatable) | Rejects paths outside the canonical roots. |
| Source files per request | 10,000 | `--max-files` | The whole static request is rejected with `413 RESOURCE_LIMIT_EXCEEDED`. |
| One source file | 2 MiB | `--max-file-bytes` | The file is reported with a safe per-file resource diagnostic; other files still complete. |
| Total source bytes per request | 128 MiB | `--max-total-source-bytes` | Remaining files are reported with safe diagnostics; completed files remain usable. |
| Git commits | 20,000 | `--max-git-commits` | The Git request fails with `413 RESOURCE_LIMIT_EXCEEDED`. |
| Output from one Git command | 16 MiB | `--max-git-output-bytes` | The Git request fails without retaining unbounded output. |

Static source reads are bounded with `io.LimitReader`, rather than relying only on file metadata. This prevents a file that grows while it is being read from exceeding the per-file allocation budget. Tree-sitter's native allocations are not a Go-controllable per-request memory pool, so the file, aggregate-source, Git-output, and commit limits are the primary safeguards. The HTTP server additionally enforces read-header, read, write, and idle timeouts.

No target-project temporary files, clones, worktrees, or generated artifacts are created. Git commands use fixed argument lists with `GIT_OPTIONAL_LOCKS=0` and never write to the target worktree, index, or refs.

## Request-scale verification plan

Run this plan on the target host or CI worker, where Go, CGO, and Git are available. Do not treat a benchmark from another machine as a capacity guarantee.

| Tier | Fixture shape | Expected outcome |
| --- | --- | --- |
| Small | Up to 100 supported files and up to 100 commits | Complete static and Git responses; record wall time and peak RSS. |
| Medium | 100–2,000 source files and up to 5,000 commits | Complete responses under configured limits; compare timings and RSS to the small fixture. |
| Large | Up to configured `max-files`, 128 MiB source budget, and 20,000 commits | Complete if within every budget; otherwise observe the documented `413` or per-file diagnostic behavior. |
| Over-limit | Beyond any one configured budget | No host crash, unbounded allocation, source leak, or fabricated zero metric; observe the documented error/diagnostic. |

Suggested commands:

```bash
cd backend
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
go test ./...
/usr/bin/time -v go test ./... -run 'Large|Resource|Limit' -count=1
```

Record the fixture size, configured flags, wall-clock time, peak resident set size, and whether a configured bound was reached. The Arena development sandbox does not provide a native Go toolchain, so this repository does **not** claim fabricated benchmark timings or memory figures from that environment.

## Cache decision

No source-result cache is enabled in this increment. A metadata-only cache can be stale when a source file changes without a distinguishable timestamp/size change, which would violate score integrity. Because this branch intentionally persists nothing, every request recomputes from current files and Git history.

A future cache must use a content digest for every analyzed source file, the resolved project scope, Git `HEAD`, worktree dirtiness, history range, analyzer/grammar version, resource policy, and risk profile as its key. It must invalidate rather than silently reuse uncertain data.
