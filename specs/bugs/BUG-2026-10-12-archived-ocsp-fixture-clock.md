---
bug_id: BUG-2026-10-12-archived-ocsp-fixture-clock
status: open
severity: LOW
scope: tests-only
---

# Archived OCSP fixture can cross a minute boundary

## Evidence

PR4 tip f2072a0e, push run38234437034, Windows Go1.27 test job
114761526563 failed at sign_test.go:3292. The test started at10:11:59.976Z;
its generated OCSP ProducedAt was10:12:00Z, outside the responder certificate's
validity. Other jobs and duplicate pull_request run38234442562 at the same
head passed. Original failure log is preserved in
`.pi-herdsman/free-list-release/ci-windows-failure.log`.

## Cause

TestArchivedOCSPDelegatedResponderValidityRemainsUnknown uses time.Now minus
one nanosecond for responder NotAfter before generating three RSA keys.
Certificate DER stores second precision. The installed x/crypto v0.57.0
ocsp.CreateResponse independently sets ProducedAt to its current minute.
Crossing a minute during key generation makes the setup assertion fail.
The failure occurs before processArchivedOCSPResponses is called. This is
fixture clock coupling, not evidence that signature policy regressed.

## Follow-up plan

Make the test-only signed OCSP timeline controllable and consistent with the
expired responder's certificate interval. Keep the setup checks, expired-now
check, archived-unknown assertion, cryptographic signatures and existing
production validation rules. Test minute-boundary and exact timestamp-precision
cases without sleeps, skips, retries inside tests or assertion removal.

This preexisting test flake is logged, not fixed by rerunning CI. Failed jobs
are being rerun at the same source tip; release requires green current-tip CI.
No production source, signature policy or private-input changes are proposed
in this reader compatibility release. Stabilization remains a separate task.
