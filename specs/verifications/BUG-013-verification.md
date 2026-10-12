---
type: verification
context: BUG-013 synthetic free-head normalization fix
---

# BUG-013 scoped verification — 2026-10-12

## Binding and disposition

Baseline: fork published v0.16.1 peeled remote tag
`495db4e004756e19bc1a717bbe92f11eb08fa24d`. Local v0.16.1 is a different
upstream tag and was not used or changed. Reviewed source committed as
`d2121f0c1d85181a92e70e820c8e2ab1aa87e464` on
`fix/free-list-hint-provenance`; source diff matches the reviewed candidate.

Synthetic fix accepted after dual independent review. Actual private-sample
and full consumer normalization validation explicitly deferred by owner;
BUG-013 is not claimed fully resolved. Owner subsequently authorized commit,
feature push and team PR. Opened https://github.com/umats/pdfcpu/pull/4 against
current default `master`; PR CI pending at this checkpoint. No merge, tag or
publication authorized. No vendor/pdfrelay changes, private reads, mail or
deployment performed.
Earlier metadata reconciliation remains preserved in named stash
`metadata-reconciliation-preserved-before-BUG-013` on its separate branch.

## Verified cause and bounded change

Changing only both fixed-width classic free-head generation fields from65535
to0 causes public metadata and Decrypt to fail at released code because
EnsureValidFreeList canonicalization sets RepairedXRef, blocking hint discovery.
Encrypted-hint controls expose unchanged live identities and RepairOffset0.
This verifies one rejection, not all later private-sample eligibility.

The fix accepts only an existing free uncompressed head/link0/generation0,
with no neighboring free entries, complete unambiguous live identities, and
sole successful post-state head-generation change to65535. Scalar copies prove
membership, flags, links, offsets and generations unchanged otherwise.
Earlier/later repair flags are never cleared, and hint gates/audits remain
unchanged. Linked free lists and compressed identities remain excluded.

Fixtures are synthetic reader-order coverage, not full Annex F documents.
Public decrypted-output checks validate output, content/metadata preservation
and encryption state. Ordinary nonaligned AES remains terminal.

## Runtime evidence

RCA (`.pi-herdsman/free-list-rca/`):

- `go test -count=1 ./pkg/api -run
  '^TestBUG013HeadGenerationCompatibilityReproduction$' -v`: exit1 baseline RED
  for metadata and decrypt; `red.log`.
- Canonical/encrypted-hint controls and free-list transformation characterization
  pass. Complete repair assignment/pointer mutation/consumer audit is in
  `findings.md` and `repair-sites.txt`.

Implemented candidate (`.pi-herdsman/free-list-fix/`):

- `go test -count=1 ./pkg/api ./pkg/pdfcpu -run TestBUG013`: exit0,
  `green-final.log`.
- `go test -count=1 ./pkg/api ./pkg/pdfcpu ./pkg/pdfcpu/model -run
  'Test(BUG013|PlaintextPrimaryHint|PrimaryHint|UnencryptedPrimaryHint|Hint|EnsureValidFreeList|OrdinaryAESAndTerminalProbeErrors)' -v`:
  exit0, `green-focused.log`.
- `go test -count=1 ./... && go vet ./... && go build ./...`: exit0,
  `preflight-fresh.log`; tests freshly executed.
- Same focused selection with `go test -race -count=1`: exit0,
  `race-final.log`.
- gofmt and diff whitespace checks: pass. Generated sample outputs restored
  from a documented initially clean sample tree; no generated samples retained.

Parent independently reran focused BUG013 tests and diff whitespace checks,
exit0. Full fresh Preflight/race evidence comes from the implementer assignment,
not independently rerun by reviewers. One invalid draft combined test used an
object-stream fixture with binary rather than classic xrefs; it was corrected,
not counted as a passing test. Existing compressed-reference tests and internal
compressed-identity denials pass; no broader compressed-head policy accepted.

## Dual independent correctness review — round1

- Reviewer A (`result:free-list-review-a#1`): no must-fix, should-fix or consider
  findings; explicit10-item checklist10/10, score100%.
- Reviewer B (`result:free-list-review-b#1`): no findings; explicit8-item
  checklist8/8, score100%.
- Both checked connected normalization/reader/hint/writer source and supplied
  runtime evidence. Review-only tools could not execute tests. Neither saw the
  other's report during review. AND-gate PASS for scoped synthetic correctness.
- Parent accepts the scoped candidate. Scores exclude private sample acceptance.

## Evidence limits and next step

LSP could not inspect read.go because it exceeds5000 lines. Other changed Go
files had no error findings; model context retains two unchanged informational
spelling advisories. Compiler/vet evidence covers source. No full LSP coverage
claim. Snapshot work/storage is O(xref entries) when the precheck qualifies;
no benchmark or mid-traversal cancellation-latency measurement performed.
Existing limits and cancellation behavior remain, but pre-canceled regression
is not a traversal-latency measurement.

When owner provides the read-only input and full consumer normalization code,
resume **validate-fix**: offline metadata, decrypt, normalization, output
validation, content preservation and no-encryption checks, all output in memory.
If a linked/compressed free-list case or later eligibility failure appears,
investigate it separately; do not broaden this classification silently.
