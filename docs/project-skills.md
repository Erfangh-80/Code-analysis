# Mandatory Project Skills and Engineering Standards

These practices are part of the project definition, not optional polish. They apply to backend, frontend, documentation examples, and every subsequent task.

## 1. Test-Driven Development (TDD)

### Required cycle

1. **Red:** write one focused test that describes the next observable behavior and watch it fail for the expected reason.
2. **Green:** write the smallest real production change that makes that test pass.
3. **Refactor:** improve design only while the complete relevant suite stays green.
4. **Review:** inspect the test result and diff. A task is not complete merely because code was written.

The test is written before implementation except for mechanical project bootstrap that has no behavior to test. Tests must assert behavior, boundaries, failures, and safety properties—not private implementation details.

### Evidence by layer

| Layer | Required test evidence |
| --- | --- |
| Domain/scoring | Unit tests for metric validity, weight/profile validation, normalization, score rounding, risk bands, and missing-data policy |
| Static analysis | Real Go, JavaScript, and Python source fixtures with independently asserted metrics; fixtures are parsed but never executed |
| Git analysis | Temporary local Git repositories with real commits and diffs for all-history/default scope, date limits, additions, deletions, and rename behavior |
| API/application | Integration tests through the HTTP contract against local fixture files and repositories, including invalid paths, errors, and resource limits |
| Frontend | Component and accessibility tests that consume contract-compliant API results; every displayed analytical value requires an implemented backend result |
| Security/reliability | Tests for no target-repository writes, safe path validation, resource limits, and parse/unsupported-file reporting |

### No fabricated completion

- A mock may isolate a transport or visual state, but it must not be used as proof that static analysis, Git history analysis, scoring, export, or integration works.
- Fixtures must be real source text and real Git histories. The system never executes fixture source.
- A parse failure, unsupported language, cancellation, or resource-limit result must be modelled and tested explicitly; zeroed or invented metrics are not acceptable substitutes.
- Every completed task report includes the exact test command(s), result, and any intentionally untested boundary with rationale.

## 2. UI/UX Pro Max

UI/UX Pro Max means building a clear, robust, accessible local analytical tool rather than merely placing charts on a page.

### Product experience principles

1. **Explain before visualizing.** Make local-only behavior, selected history range, scoring-profile version, and risk-score limitations visible before users interpret a result.
2. **Prioritize action.** Results start with the highest-risk files and a concise explanation of *why* each ranks there; they do not bury action behind decorative visualizations.
3. **Reveal detail on demand.** Keep the overview scannable; provide accessible detail for raw metrics, score contributions, and diagnostics.
4. **Never imply certainty.** Use precise language: risk is a prioritization signal, not a defect prediction. Show unavailable and partial-analysis states honestly.

### Required interaction states

Every feature with data must design and test all of the following:

- initial/empty state with a clear next action;
- local repository-path validation and permission/access errors;
- in-flight request state while an analysis call is running;
- completed data state;
- parse errors, unsupported files, and partial diagnostics;
- API failure state with actionable recovery;
- responsive narrow and wide layouts; and
- keyboard-only and screen-reader workflows.

### Accessibility baseline

- Target **WCAG 2.2 AA**.
- Use semantic landmarks, real buttons/inputs/labels, predictable heading order, and native focus behavior.
- Ensure full keyboard operation, visible focus, logical tab order, escape/close behavior, and focus restoration for overlays or detail panels.
- Do not encode risk solely by color. Pair every band with text, icons/patterns where applicable, and programmatic labels.
- Ranking and result tables must expose the same file, metric, and risk information that any visual presentation would, so no essential result is available only through a chart or on hover.
- Meet contrast requirements, honor reduced-motion preferences, and announce request progress/status without excessive interruption.
- Render untrusted paths and diagnostics as text, not HTML.

### Result presentation requirements

- Ranking rows must show path, risk band/score, key drivers, and score contributions returned by the backend.
- Numeric formats, units, zero, unavailable, and outlier values must be unambiguous.
- Tables must retain meaningful labels and navigation when resized.

### UI quality gate

A UI task is complete only when its implementation has been reviewed at supported viewport sizes, keyboard navigation has been exercised, and it renders real backend responses in tests. Screenshot-perfect mock data is not evidence of a completed analytic feature.

## 3. Shared quality gates

Before declaring any capability complete:

- [ ] its acceptance behavior has tests;
- [ ] its implementation uses no external/cloud analysis dependency;
- [ ] target source code is never executed;
- [ ] API behavior matches the documented contract where applicable;
- [ ] UI logic does not calculate backend-owned metrics or risk;
- [ ] relevant tests were run and their output reviewed;
- [ ] error and unsupported/partial cases are truthful; and
- [ ] documentation has been updated for user-visible behavior.
