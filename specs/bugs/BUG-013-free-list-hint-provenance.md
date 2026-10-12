---
bug_id: BUG-013-free-list-hint-provenance
status: synthetic_verified_private_validation_deferred
severity: MEDIUM
scope: reader-encryption
title: Free-list normalization disqualifies plaintext-hint compatibility
---

# BUG-013: free-list normalization and hint provenance

## Problem

Owner scalar debugger evidence identifies generation-only object-0
normalization as one cause of the private sample's compatibility rejection.
It sets the conservative xref-repair flag before hint discovery; ordinary AES
then rejects a nonaligned stream. Later private-sample eligibility is unknown.

This is related to the prior plaintext-hint fix, not a demonstrated regression
from a previously passing private sample. See `BUG-013-owner-evidence.md`.

Context: an internally used PDF library, not a public-facing service.
No cybersecurity incident or exploit has been demonstrated. The investigation
concerns compatibility and correctness: distinguish harmless normalization
from changes that make object identity unreliable, without accepting ordinary
encrypted streams as plaintext or deleting document content.

## Baseline and authorization

- Target: current pdfcpu checkout, per owner confirmation.
- Branch: `fix/free-list-hint-provenance`.
- Fork release v0.16.1 remote peeled tag:
  `495db4e004756e19bc1a717bbe92f11eb08fa24d`.
- Local v0.16.1 tag points elsewhere (`2f7a89a0`); do not use it as this
  release's baseline and do not rewrite tags.
- Fresh native Preflight passed at the target hash earlier this session.
- Prior metadata edits preserved in stash
  `metadata-reconciliation-preserved-before-BUG-013` on their own branch.
- Owner subsequently authorized commit, feature-branch push and team PR.
- No merge, tags, publication, vendor edits, private mutation, mail or deployment.
- Conditional implementation only after verified RCA and a narrow safety proof.

## Reproduce

Verified at released hash: changing both fixed-width classic object-0 records
from generation65535 to generation0, with no other byte changes, makes public
metadata and Decrypt fail on hint object8 with the AES alignment error.
Canonical input passes. Encrypted-hint controls expose RepairedXRef=true and
RepairOffset=0 with unchanged non-head identities. Fixtures are synthetic
reader-order coverage, not complete Annex F documents. Exact commands/logs:
`.pi-herdsman/free-list-rca/findings.md`.

Private input and the consumer normalization pipeline are not available.
Do not claim actual-sample success from synthetic results.

## Isolate

Audit completed for all repair assignments, pointer mutations and consumers.
The free-list model canonicalizes the head generation without checking its
free/in-use state. It can also reconstruct heads and rewrite malformed links,
cycles and dangling entries. Therefore generation-only deltas alone are not
adequate proof. Prior or later independent repairs must remain terminal.

## Hypothesize

1. Canonical head-generation normalization alone can preserve live provenance.
   Falsify with free/in-use confusion, malformed links, competing identity,
   generation mismatch or any other repair flag assignment.
2. The private sample has additional eligibility defects after this first gate.
   Requires controlled offline tracing after safe candidate discovery exists.

## Verify

RCA verified: the scalar comparison after free-list normalization marks the
head-generation-only case as repaired; candidate discovery rejects it before
reference/origin verification. Only the exact empty-free-list case is accepted
for implementation: existing free, uncompressed head, link0, generation0,
no neighboring free entries, successful canonicalization to generation65535,
and no other state change. Never clear an already-set repair flag.

## TDD fix plan

1. GREEN the public metadata/decrypt RED using only the exact proven empty-list
   head-generation classification. Keep all discovery/finalization gates.
   Update the repair flag's documentation to distinguish unreliable xref
   provenance from this benign canonicalization, without a new public field.
2. Preserve/extend combined malformed-input controls: prior and later repair,
   in-use or compressed head, nil/missing head fields, neighboring free
   entries, invalid links, unexpected head generation, hint-generation
   mismatch, ordinary nonaligned AES, references, limits and cancellation.
3. Run existing hint/provenance tests, fresh full Preflight and focused race.
   Obtain an independent correctness review before synthetic acceptance.

Do not expand to linked free lists without another proof and explicit scope.

## Acceptance criteria

- [x] Public metadata/decrypt RED on the exact released baseline.
- [x] Verified narrow empty-list head-generation classification proof.
- [x] Terminal unsafe xref, identity, generation and ordinary AES cases preserved.
- [x] Existing hint origin/reference, syntax, cancellation and limit tests pass.
- [x] Required repository Preflight passes after the narrow fix.
- [x] Dual independent correctness review passes with no findings.
- [ ] Actual sample passes metadata, decrypt, full normalization, validation and
      no-encryption checks. Owner deferred this check; remains unverified.

## Resolution

**Synthetic fix verified — 2026-10-12.** Private-sample BUG-013 remains unresolved.

The exact empty-free-list generation0 case now permits discovery to proceed,
subject to unchanged reference/origin and all other hint eligibility checks.
No existing true repair flag is cleared. Linked free lists, compressed or
incomplete live identities, altered links and nonzero unexpected generations
are deliberately excluded. No new public field, vendor or consumer change.

Public metadata and Decrypt now pass for the reproduced case; decrypted output
is validated with content/metadata preservation and no encryption. Fresh full
Preflight and focused race pass. Both independent reviewers returned no findings
(A10/10, B8/8), scoring100% for the scoped candidate—not private acceptance.
Reviewed source committed as `d2121f0c` on `fix/free-list-hint-provenance`.
Team PR: https://github.com/umats/pdfcpu/pull/4 against current default `master`.
PR CI pending at this checkpoint; no merge or publication authorized.

Owner explicitly deferred actual-sample and consumer normalization validation.
No full BUG-013 resolution, Annex F, performance, production or release claim.
See `specs/verifications/BUG-013-verification.md` for commands and evidence limits.
