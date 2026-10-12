# BUG-013 — owner-supplied investigation evidence

This is transferred evidence, not a reproduction performed in this session.

## Observed debugger behavior

An offline public `api.Decrypt` call on the actual `6e60` PDF was compiled
with `-gcflags='all=-N -l'`. Delve inspected scalars only; no PDF, email
content, encryption key, or password was printed.

At `discoverHintStreams` entry:

```text
ctx.Read.RepairedXRef = true
ctx.Read.RepairOffset = 0
```

Discovery immediately returns `nil, nil`. Object 1690 then loads through
ordinary stream decryption and fails with `ciphertext not a multiple of
block size`; candidate discovery and reference/origin audits never run.

The first assignment observed by reader breakpoints was the comparison after
`EnsureValidFreeList` in `readXRefTable` (reported v0.16.1 line 2685):

```text
nr = 0
found = true
before[0] = {offset: 0, generation: 0}
after[0]  = {offset: 0, generation: 65535}
```

This establishes one concrete eligibility rejection. It does not prove that
no other failure exists later. A free-list-head generation change alone
shows no live-object physical-offset mutation; not every free-list change
is harmless.

## Required investigation and safety boundaries

1. Reproduce generation-only object-0 normalization with a synthetic encrypted
   linearized fixture; establish public metadata/decrypt failure on v0.16.1.
2. Determine whether free-list-only normalization can preserve trustworthy
   live-object provenance. Audit object 0 free/in-use state, generation,
   links and all neighboring free entries.
3. Audit every other assignment to `RepairedXRef` before proposing a change.
4. Preserve terminal handling for live-offset repair, rebuilt xrefs, skipped
   entries, duplicate identities, ambiguous parsing and generation mismatches.
5. Retain reference and physical-origin audits for every authorized candidate.
6. Implement a minimal explicit distinction only after safety proof. If unsafe,
   document rejection and request an explicit alternative policy.
7. Once discovery can proceed safely, trace later actual-sample failures.

Never clear the flag in pdfrelay, blindly ignore it here, suppress AES errors,
modify production bytes, or drop arbitrary streams. Hint handling remains a
relaxed-mode compatibility repair, not a normative encryption exemption.

## Regression and acceptance requirements

- Positive generation-only head normalization with unchanged live offsets,
  conditional on established safety proof.
- Negative live-offset repair and rebuilt xrefs remain ineligible.
- Free/in-use identity confusion and malformed free links cannot authorize
  ordinary stream plaintext.
- Existing hint reference, physical-origin, ambiguous syntax, cancellation
  and resource-limit tests remain green.
- Ordinary nonaligned AES streams remain terminal.
- Actual sample must pass offline metadata reading, decrypt, normalization,
  output validation and no-encryption checks before acceptance.
- Committed fixtures must be synthetic. Private input read-only; output in
  memory. Do not print secrets/content, send mail or deploy.

## Reported consumer state

pdfrelay `go test ./...` passes after restoration of its managed Producer
patch. Build and configured lint passed during integration. Two other private
samples normalize successfully using an explicit empty-password candidate;
`6e60` remains unresolved. Dependency update and user changes are uncommitted.
Independent review there is unavailable; this does not waive review here.
No pdfrelay vendor or consumer changes are authorized by this investigation.

Verification requested: baseline and post-fix upstream public-API regression,
security negatives, and repository Preflight.
