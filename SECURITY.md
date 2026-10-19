# Security Policy

## Scope and security fixes

This policy covers the pdfcpu Go library, CLI, official release binaries and dependencies included in those artifacts.
Reports concerning older releases are welcome and help establish which versions are affected.

Security fixes are delivered in new releases. Only the latest release receives security fixes; the project does not
provide maintenance releases or backports for older versions. Users should upgrade to the release identified in the
security advisory. An upgrade may require API or configuration changes; release notes and migration guidance describe
those changes.

## What counts as a vulnerability

pdfcpu is designed to process untrusted PDF files within configurable resource limits. The following are in scope:

- crashes, hangs or excessive memory, CPU or disk use caused by crafted input when the default resource limits apply;
- input that bypasses configured resource limits, for example decoding more data than the configured maximum;
- outbound network requests that violate the configured network policy;
- results that contradict pdfcpu's documented encryption behaviour, for example content written unencrypted when
  encryption was requested;
- reading or writing files outside the paths a command or API call was given.

The following are usually out of scope:

- resource use within the configured limits, including slow processing of large but valid files;
- behaviour after limits have been disabled or raised beyond their documented defaults;
- issues that require control over pdfcpu's configuration, command-line arguments or execution environment;
- legacy algorithms supported for compatibility with existing documents;
- PDF permission flags, which are honoured by PDF readers rather than enforced by encryption;
- vulnerabilities in dependencies that do not affect pdfcpu. Please report those upstream, and let us know if you
  believe pdfcpu is affected.

If you are unsure, report it. Minor issues may be fixed in a regular release without a separate advisory.

### Signature validation

Signature validation is under active development and is being reworked for v0.17.0. Until then:

- limitations listed in the [signature validation documentation](https://pdfcpu.io/core/sign_validate) are known and
  out of scope; please report other functional problems as regular issues;
- reports where pdfcpu presents a modified document as intact, or a forged signature as cryptographically valid,
  remain in scope. Fixes will normally ship with v0.17.0 rather than in an interim release, unless there is evidence
  of active exploitation.

## Reporting a vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Use [GitHub private vulnerability reporting](https://github.com/pdfcpu/pdfcpu/security/advisories/new), or email
[security@pdfcpu.io](mailto:security@pdfcpu.io) if you cannot use GitHub reporting.

Please include what you know:

- affected pdfcpu version or commit, and binary build information where available;
- operating system and architecture;
- command or API entry point and relevant configuration, including any non-default limits;
- a minimal reproducer or input file, if it can be shared;
- observed impact and any mitigation you have identified;
- whether the issue is public, with links where available; and
- whether you suspect exploitation outside testing, and when it was observed.

Incomplete reports are welcome. If exploitation is suspected, identify that clearly in the subject or opening sentence.
Do not send passwords, private keys or confidential documents. Describe sensitive evidence first so an appropriate
transfer method can be agreed.

## AI-assisted and automated reports

Reports found or written with the help of AI tools, fuzzers or other automated analysis are welcome and are assessed on
their merits. Please say so in the report. Before submitting, reproduce the issue yourself against the latest release
or current `master` and include a reproducer. Unverified tool output and reports that cannot be reproduced may be
closed without detailed assessment.

The restrictions on AI-assisted pull requests in [CONTRIBUTING.md](CONTRIBUTING.md) apply to code contributions, not to
security reports.

## Multiple findings

Please submit one report per root cause. Findings that share a cause belong in one report.

If you have many findings, for example from a fuzzing campaign or a security review, send a short list first, by email
or in a single report, before filing individual reports. Findings are assessed in batches, starting with the highest
security impact. Proposed patches are welcome as input; fixes are implemented by the maintainer, and the reporter is
credited with their consent.

## Response and assessment

The maintainer aims to acknowledge reports within three business days and provide an update at least every seven
calendar days while assessment or remediation is active. Updates may explain that investigation is continuing.
These are communication targets, not guaranteed resolution times. The project currently has one maintainer and no
absence coverage, so responses may be delayed.

Reports are assessed for impact, affected versions and possible mitigations. Suspected active exploitation is prioritised.
The maintainer may request additional information and coordinate with affected dependency maintainers.
If you have not received an acknowledgment within the target period, please follow up by email.

## Coordinated disclosure

The maintainer will coordinate disclosure timing with the reporter and relevant parties, taking account of user risk,
available fixes or mitigations, and any ongoing exploitation. Please allow time for assessment and remediation before
publishing technical details. This is a request for coordination, not a requirement for an indefinite embargo.

Confirmed vulnerabilities are normally published through
[GitHub Security Advisories](https://github.com/pdfcpu/pdfcpu/security/advisories), with affected versions, impact,
the fixed release or available mitigations, and upgrade guidance. A CVE ID is normally requested through GitHub when an
advisory is published. Urgent protective guidance may be published before a fix is available. Reporter credit is
included with the reporter's consent.

Report details are treated as confidential and shared only as needed for investigation, remediation, coordination and
applicable legal reporting. Regulatory notifications, where required, are handled
separately from public advisory publication and are not delayed to meet a preferred disclosure date.

## Testing

In addition to the tests in this repository, fuzzing and most regression testing run in a separate repository.
