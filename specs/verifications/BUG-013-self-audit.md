---
type: self-audit
context: BUG-013 narrow free-head normalization
---

# BUG-013 self-audit

Candidate: uncommitted source on `fix/free-list-hint-provenance`, baseline
`495db4e004756e19bc1a717bbe92f11eb08fa24d`.

- PASS scope: only reader classification, repair-flag comment and synthetic
  regressions. No vendor/consumer/dependency/public API or writer changes.
- PASS correctness: pre-state requires free uncompressed head/link0/generation0
  and no other free entries; scalar snapshots prove the sole post-state change.
  Existing true repair flags are never cleared; hint gates remain untouched.
- PASS malformed-input handling: public combined denials and internal
  nonrepresentable-state tests cover identity/link/generation and later repair.
  Existing authentication/reference/origin/limit/cancellation checks retained.
  No incident/exploit or private-sample success claim.
- PASS supply chain and data handling: no added dependencies or private input,
  passwords, keys or content in candidate source/fixtures.
- PASS types/design: unexported typed helpers and scalar identity map; no public
  fields, casts, policy options, generic fallback or speculative abstraction.
- PASS tests: exact baseline public RED retained as evidence; public GREEN and
  controls reproduce owner scenario synthetically. Internal tests complement,
  rather than replace, public behavioral regressions.
- PASS project conventions: feature branch, gofmt, focused regressions, full
  fresh Preflight and focused races reported successful; no commits/pushes.
- PASS clarity: comments explain scalar copying and conservative exclusion;
  no dead or commented-out code introduced. Existing large reader file retained
  rather than unrelated refactoring or splitting to meet generic style heuristics.
- PERFORMANCE BOUNDARY: snapshot adds O(number of xref entries) allocation/work
  only for the qualifying head-generation0 candidate. No benchmark or performance
  improvement claim; linked/compressed entries fail closed.

## Validation evidence

Implementer report `.pi-herdsman/free-list-fix/findings.md` records fresh
`go test -count=1 ./... && go vet ./... && go build ./...` exit0 and focused
race exit0. Parent independently ran `go test -count=1 ./pkg/api ./pkg/pdfcpu
-run TestBUG013` and `git diff --check`, both exit0.

Active LSP check: three files clean, model/context.go has two unchanged
informational spelling findings (`obsolet`); read.go exceeds the tool's 5000-line
limit. Do not claim complete LSP coverage; compiler/vet evidence covers source.
Churn utility ranked read.go first (26 commits), model/xreftable.go fifth
(17 commits); the utility returned141 after producing rankings. This utility
is not Preflight or a source validation result; its nonzero outcome is retained
here rather than called PASS. No source gate failures remain.

## Next gate

Independent dual correctness review. Actual private sample and complete consumer
normalization remain unverified; this audit is not end-to-end BUG-013 acceptance.
