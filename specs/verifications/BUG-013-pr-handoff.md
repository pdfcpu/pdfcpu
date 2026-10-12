---
type: team-pr-handoff
context: BUG-013 source and scoped verification
---

# BUG-013 team PR handoff — 2026-10-12

Owner explicitly authorized commit, feature push and team PR. These actions
are complete; no merge, tag, version selection or publication authorized.

- PR: https://github.com/umats/pdfcpu/pull/4
- Base: current fork default `master`; head `fix/free-list-hint-provenance`.
- Source: `d2121f0c1d85181a92e70e820c8e2ab1aa87e464`.
- Verification/specs: `5d3e077751668742baadec464cb2bcb1b1dcc78c`.
- Scoped source matches the accepted candidate. No new public identifier,
  dependency/module, vendor or pdfrelay changes.
- All18 reported checks succeeded at5d3e0777, verified via explicit
  `gh pr checks 4 --repo umats/pdfcpu --watch --interval 20` exit0.
- CI includes Linux/macOS/Windows Go1.26/1.27 tests, cross-compilation and
  coverage-finish across push and PR runs. Aggregate coverage percentages
  remain owner-waived, not a passing percentage gate.
- This docs-only receipt and state update do not alter source. Tip checks
  are rechecked after pushing the receipt; this hash-bound observation is
  not an assertion about a future unseen commit.

Private-sample and full consumer normalization remain explicitly owner-deferred.
Synthetic metadata/decrypt/output checks pass; actual BUG-013 remains unresolved.
No private reads, byte mutation, mail or deployment performed.

Previous metadata reconciliation remains in the named stash
`metadata-reconciliation-preserved-before-BUG-013`; do not pop it into this PR.
Use explicit `--repo umats/pdfcpu` for forge operations.

Next: retain the open feature PR; check live CI before any future integration.
Merge/publication need separate approval. Resume private **validate-fix** only
when the owner supplies input and the consumer pipeline; no broader linked or
compressed free-list policy is authorized by this change.
