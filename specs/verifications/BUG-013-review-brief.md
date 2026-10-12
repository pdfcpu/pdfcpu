---
type: review-brief
context: BUG-013 exact empty-free-list head canonicalization
---

# Independent correctness review brief — round 1

Review the uncommitted candidate at baseline `495db4e0` in this checkout.
Internal non-public PDF library compatibility work; no demonstrated incident or
exploit. Assess correctness and retained malformed-input safeguards without
framing this as a public-facing cybersecurity incident.

## Required behavior

Only an existing free, uncompressed object0 with link0/generation0, no other
free entries, complete unambiguous identities, and successful sole generation
canonicalization to65535 may avoid setting RepairedXRef. Prior/later true repair
flags must remain true. Arbitrary linked-list normalization is not authorized.
Hint discovery/finalization/reference/physical-origin/parse audits, passwords,
strict mode, ordinary AES, cancellation/limits and writer inventory stay intact.

No exported fields, dependencies, vendor or pdfrelay edits. No sample-fixed
claim without actual offline metadata/decrypt/full normalization/output validation
and no-encryption checks. Synthetic fixtures are reader-order coverage, not
Annex F conformance proof.

## Candidate source and tests

- pkg/pdfcpu/read.go: new private snapshot/classification helpers and one narrow
  suppression of the normalization repair assignment.
- pkg/pdfcpu/model/context.go: repair flag comment only.
- pkg/api/free_list_hint_reproduction_test.go: public RED-to-GREEN, content and
  encryption state, identities/inventory and combined rejection controls.
- pkg/pdfcpu/free_head_classification_test.go: nil/compressed/neighbor/generation
  states and post-normalization mutation guards.
- pkg/pdfcpu/free_list_provenance_audit_test.go: model characterization.

Read connected model EnsureValidFreeList, reader assignments/consumers,
readHints discovery/finalization and writer inventory use as needed. Consider
aliasing, compressed object identities, older revision provenance, earlier and
later repairs, ordinary AES versus hint eligibility, and snapshot cost/limits.
Check whether tests really reach claimed paths and preserve content.

## Evidence and project constraints

Relevant specs: SCOPE_LATEST.yaml, BUG-013 bug and owner evidence, self-audit.
CONVENTIONS requires feature branches/team PR, minimal layer-respecting edits,
gofmt, regression tests, context/cancellation, wrapped errors where applicable,
Preflight before forward work/merge, no vendor edits, no weakening lint settings.
Aggregate coverage percentages are owner-waived, not a PASS percentage gate.

Baseline public RED is in free-list-rca/red.log and its report. Implementer
records full fresh `go test -count=1 ./... && go vet ./... && go build ./...`
exit0, plus focused race exit0. Parent focused BUG013 test run and diff check
exit0. LSP cannot inspect read.go because it exceeds its file-size limit;
model/context.go has unchanged informational spelling advisories.

Verify command for this behavior:
`go test -count=1 ./pkg/api ./pkg/pdfcpu ./pkg/pdfcpu/model -run 'Test(BUG013|PlaintextPrimaryHint|PrimaryHint|UnencryptedPrimaryHint|Hint|EnsureValidFreeList|OrdinaryAESAndTerminalProbeErrors)'`

Review-only tools may not permit executing commands. In that case explicitly
state tests were not rerun; evaluate supplied canonical log evidence rather
than inventing runtime verification. Do not edit files or run tests concurrently
with another reviewer. No shared reports from the other reviewer.

## Return

Independent report with must-fix / should-fix / consider findings (or none),
connected-source paths/line references, exact evidence limits, and quality score
from an explicit counted checklist. Passing requires zero must-fix and score
at least94; do not pad item counts or treat unverified private acceptance as PASS.
Report specific regressions, overbroad behavior or needless complexity, not
hypothetical general library risks. Keep scope narrow.
