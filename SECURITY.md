# Security Policy

## Scope and security fixes

This policy covers the pdfcpu Go library, CLI, official release binaries and dependencies included in those artifacts.
Reports concerning older releases are welcome and help establish which versions are affected.

Security fixes are delivered in subsequent releases. The project does not provide maintenance releases or backports.
Users should upgrade to the release identified in the security advisory. An upgrade may require API or configuration
changes; release notes and migration guidance describe those changes. No fixed security support period is promised.

## Reporting a vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Use [GitHub private vulnerability reporting](https://github.com/pdfcpu/pdfcpu/security/advisories/new), or email
[security@pdfcpu.io](mailto:security@pdfcpu.io) if you cannot use GitHub reporting.

Please include what you know:

- affected pdfcpu version or commit, and binary build information where available;
- operating system and architecture;
- command or API entry point and relevant configuration;
- a minimal reproducer or input file, if it can be shared;
- observed impact and any mitigation you have identified;
- whether the issue is public, with links where available; and
- whether you suspect exploitation outside testing, and when it was observed.

Incomplete reports are welcome. If exploitation is suspected, identify that clearly in the subject or opening sentence.
Do not send passwords, private keys or confidential documents. Describe sensitive evidence first so an appropriate
transfer method can be agreed.

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
the fixed release or available mitigations, and upgrade guidance. Urgent protective guidance may be published before
a fix is available. Reporter credit is included with the reporter's consent.

Report details are treated as confidential and shared only as needed for investigation, remediation, coordination and
applicable legal reporting. Regulatory notifications, where required, are handled
separately from public advisory publication and are not delayed to meet a preferred disclosure date.
