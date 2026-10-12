---
bug_id: BUG-2026-10-12-release-validation
status: verified
severity: LOW
scope: release-validation
title: Release validation needs patched Go and full race completion
---

# Release-validation triage — 2026-10-12

## Observed gates

Candidate: PR4 source plus one-line VersionStr change to v0.16.2.
`go test -count=1 ./... && go vet ./... && go build ./...` passed.

`go test -race -count=1 ./...` exited1: pkg/api/test reached the default
10-minute package timeout while TestImagesTextAndPDFWMViaWatermarkMap was
running. The stack was runnable in compression, not a reported race.
All other package results printed PASS, including pkg/cli/test (358 seconds)
and pkg/pdfcpu (187 seconds). No whole-race PASS claim; time-budget exhaustion
is a hypothesis to verify by rerunning unchanged assertions with a larger
explicit package timeout. Do not skip tests or suppress race instrumentation.

`govulncheck ./...` exited3 on local Go1.26.8, reporting nine standard-library
advisories fixed in Go1.26.9: GO-2026-6617,6613,6612,6611,6610,6608,6607,6605,6603.
Canonical output: `.pi-herdsman/free-list-release/govulncheck.log`.
This is a toolchain scan result, not a demonstrated incident in the internal
PDF library. Validate using the patched Go toolchain without raising the
module's Go floor or changing vendor files.

## Required validation

- [x] Full race suite completes with explicit adequate time budget after
      correcting the reproduced process-global test scheduling race.
- [x] Patched Go1.26.9 scan: zero reachable findings; one module advisory not
      called by project code. Ordinary Preflight passed before the full race run.
- [x] Module tidy-diff clean; exact fork-v0.16.1 mechanical module API diff
      empty; CLI version assertion v0.16.2 passes.
- [x] Updated code tip f2072a0e CI passes, including failed-job rerun.
      Documentation-only receipt tip must also pass before merge/publication.

Merge/release remain blocked while these validation checks are unresolved.
Owner has authorized PR4 merge and source-only v0.16.2 release after green gates.
Private sample and consumer normalization remain owner-deferred, not passed.

## Resolution

Patched toolchain scan passed without project dependency/vendor changes.
Module/API/version checks pass. Canonical logs under
`.pi-herdsman/free-list-release/` include govulncheck-patched.log,
module-hygiene.log, apidiff.log and version.txt.

The completed Go1.26.9 race run found a ConfigPath test-harness read/write
race. Focused-race count20 reproduced it. Test commit e9e3ac44 removes only
top-level parallelism from both global-mutating tests; the assertion and100
concurrent writer calls remain. Focused-race count100 and patched ordinary
Preflight pass. Independent static reviewers A/B report no findings,6/6 each.

The full race rerun then passed: exit0,463 seconds, no timeout or race warning.
The executor cleaned generated samples and preserved prior failing logs.
Parent removed only two additional known temporary outputs of its Preflight.
No runtime work was duplicated while the delegated executor was unresolved.

A Windows CI setup assertion crossed a minute boundary during OCSP fixture
key generation. The failure is logged separately in
BUG-2026-10-12-archived-ocsp-fixture-clock, with a stabilization plan. Both
current-tip CI runs pass after failed-job rerun; rerunning is not fixture repair.
See specs/verifications/v0.16.2-release-validation.md for exact gate evidence.
No test skipping, assertion removal or production API/locking workaround.
Merge/publication have not yet occurred.
