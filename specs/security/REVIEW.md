# Fork change security review

Scope: `fix/aesv2-missing-cf-length` versus upstream base `4d0e9ff2`.

- The committed AESV2 exception was reviewed independently before the module rename. Reviewer findings on pre-auth Catalog stream recursion, corrupt-offset repair, and indirect Catalog Version were addressed with syntax-only bootstrap parsing and focused tests.
- The locally committed rename changes first-party import literals in 506 Go files (plus gofmt import ordering in 13 CLI files), module names, build/linker paths, coverage scripts, Docker build source, and docs. A mechanical content comparison found no unexpected Go edits. No new dependency or authentication/cryptography logic is introduced by the rename.
- `go list -deps ./...` and `go list -m all` contain no upstream pdfcpu identity; the public API authentication regression (wrong password) remains covered in `pkg/api/crypt_filter_length_test.go` and passes under the renamed module.
- Self-audit found that Docker's broad `COPY . .` sent `.git` and PDF fixtures to the local build daemon (755.4 MB context). A native `.dockerignore` now excludes `.git`, all PDFs, `pkg/samples`, and `pkg/testdata`. Rebuilding passed with a 27.74 MB context; the final image contains the CLI binary only.
- No new HIGH-confidence exploit path was found in the rename diff. Residual risk remains from processing intentionally nonconforming encrypted PDFs and pre-auth Catalog syntax. The rename received a self-audit, **not an independent reviewer audit**; obtain one before integration. A code review cannot establish safety for all malformed PDFs.
- No production PDF, password, customer identity, or metadata was used for the regression. No pdfrelay/vendor changes or release/publish actions occurred.

## Plaintext hint round 10: scoped security PASS

Scope: current uncommitted candidate on
`fix/plaintext-linearization-hints`, including untracked sources.
Evidence: `result:hint-security10#1`; candidate and runtime receipts:
`specs/verifications/plaintext-hint-round10-runtime-verify.yaml`.

The read-only reviewer found no concrete security findings. Authentication
and permissions precede discovery; physical origins, provenance and complete
incoming-reference auditing gate fallback and deletion. Normal cipher handling
runs first on independent copies. Terminal limits and cancellation remain
terminal. Full writes delete verified identities only; incremental writes
re-encrypt repaired hints.

The reviewer inspected source and supplied logs but executed no tests.
This is a bounded security scan, not independent acceptance or release
approval. Owner accepts synthetic behavioral evidence only. Production input,
full Annex F conformance, optional early-xref-stream coverage and performance
remain unverified; two historical consumer failures remain unexplained.

## Plaintext hint round 13: current-candidate security PASS

The earlier round10 scan is historical, not current acceptance. Later
acceptance reviews found concrete decode-budget and provenance defects;
those findings were corrected and regression-tested through round13.

Current independent evidence: `result:hint-accept13a#1` and
`result:hint-accept13b#1`. Both report zero findings, 100% acceptance and
scoped security PASS (confidence 8/10). Both inspected connected code and
regression assertions without executing commands or modifying files.
Final candidate receipts and limitations:
`specs/verifications/plaintext-hint-round13-verification.md`.

Lead accepts both bounded reviews. No unresolved current security finding
remains within the approved scope. This is not a universal security guarantee,
standards-compliance statement, production reproduction or release approval.

## BUG-013 / v0.16.2: scoped compatibility-boundary review

Scope: `fix/free-list-hint-provenance` against exact fork v0.16.1 commit
`495db4e0`, source `d2121f0c`, one-line VersionStr change to v0.16.2, and
ConfigPath test scheduling correction `e9e3ac44` (no production locking change).
Internal-library compatibility/correctness work; no incident or exploit shown.

Independent A/B reviews (`result:free-list-review-a#1`,
`result:free-list-review-b#1`) found no concrete findings. They read connected
normalization, reader, hint and writer source and supplied regression evidence.
The only normalization exception requires the existing empty free head and
scalar proof of unchanged live identities. Earlier/later repairs remain
monotonic, and authentication, normal AES handling, hint references/physical
origins, parse provenance, strict mode, limits and cancellation are unchanged.
Writer deletion inventory remains verified-only. No public field or broader
linked/compressed classification, private input, vendor or consumer changes.

VersionStr is release display data only; no new cryptography or input flow.
No new HIGH-confidence finding in this scoped diff. This is not a blanket
claim about all library inputs or the deferred private normalization pipeline.

Patched Go1.26.9 scan reports zero reachable findings, with one noncalled
module advisory; see `.pi-herdsman/free-list-release/govulncheck-patched.log`.
Earlier Go1.26.8 scanner failure and full-race timeout remain recorded in
`specs/bugs/BUG-2026-10-12-release-validation.md`. After correcting the
reproduced global test-harness race, the full patched race suite passes.
Independent release-delta A/B static reviews also have no findings,6/6 each.
An unrelated OCSP fixture clock flake remains explicitly logged for follow-up,
not asserted to be fixed by the green CI rerun. Current source-tip CI passes;
documentation receipt tip must also pass before merge/publication.
