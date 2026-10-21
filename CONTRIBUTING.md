# Contributing to pdfcpu

Thank you for your interest in contributing to pdfcpu.

pdfcpu is an open-source PDF processor written in Go. The project is licensed under the Apache License, Version 2.0.

This document explains how contributions are handled and what is required before a contribution can be accepted.

## License

The pdfcpu project is licensed under the Apache License, Version 2.0.

By contributing to pdfcpu, you agree that your contribution is submitted under the Apache License, Version 2.0, unless explicitly stated otherwise in writing and accepted by the project maintainers.

Contributions may be used, modified, distributed, sublicensed, and otherwise handled as permitted by the Apache License, Version 2.0.

## Developer Certificate of Origin

Before a pull request can be merged, the contributor must accept the [Developer Certificate of Origin 1.1](https://cla-assistant.io/pdfcpu/pdfcpu) when the CLA Assistant check on GitHub asks for it. Pull requests may not be merged until the check confirms acceptance.

By accepting it, you certify that you have the right to submit the contribution under the project's license. The Developer Certificate of Origin grants no rights beyond the Apache License, Version 2.0.

`Signed-off-by` lines in commits are welcome as an additional provenance signal but are not required.

A sign-off line looks like this:

```text
Signed-off-by: Your Name <your.email@example.com>
```

You can add this automatically when committing:

```sh
git commit -s
```

## Contributor Responsibility

By submitting a contribution, you confirm that:

- you wrote the contribution yourself, or otherwise have the right to submit it;
- the contribution can be licensed under the Apache License, Version 2.0;
- the contribution does not knowingly include code copied from projects with incompatible licenses;
- the contribution does not include confidential information, trade secrets, customer documents, private PDFs, credentials, or other material you are not allowed to share;
- if you are contributing on behalf of your employer or another organization, you are authorized to make the contribution.

If you are unsure whether you have the right to contribute something, please do not submit it until the rights are clear.

## Contribution Scope

Good contributions include:

- bug fixes;
- parser, validation, optimization, extraction, encryption, signature, or CLI improvements;
- tests and regression cases;
- documentation improvements;
- examples;
- performance improvements;
- fixes for platform-specific issues;
- improvements that make pdfcpu more robust in production workflows.

Before starting larger changes, please open an issue or discussion first. This helps avoid duplicated work and makes it easier to align the contribution with the project direction.

## Security Issues

Please do not report security vulnerabilities as public GitHub issues.

See the [security policy](SECURITY.md) for private reporting instructions and the coordinated disclosure process.
Security reports follow that policy, including its rules for AI-assisted reports and multiple findings.

## Test Files and PDFs

PDFs used for tests must be shareable.

Do not submit:

- confidential customer PDFs;
- private business documents;
- documents containing personal data;
- documents copied from proprietary test suites;
- documents you do not have permission to redistribute.

If a bug depends on a sensitive PDF, try to create a minimized synthetic test file that reproduces the issue without exposing private content.

## Coding Guidelines

pdfcpu is written in Go.

Please keep contributions consistent with the existing codebase:

- format Go code with `gofmt`;
- keep changes focused;
- avoid unrelated refactoring in bug-fix pull requests;
- add or update tests where practical;
- keep public APIs stable unless the change has been discussed;
- prefer clear error handling over hidden recovery;
- avoid introducing unnecessary dependencies.

## Pull Requests

Before opening a pull request:

1. Create an issue or a discussion.
2. Discussion with maintainers will determine if a PR is appropriate.
3. Make sure your branch is up to date.
4. Keep the pull request focused on one topic.
5. Include tests or a clear explanation if tests are not practical.
6. Include a useful description of the problem and the solution.
7. Accept the Developer Certificate of Origin when the CLA Assistant check asks for it.
8. Do not include unrelated formatting or cleanup changes.

The maintainers may ask for changes before merging. Not every contribution can be accepted.

## Dependencies

New dependencies should be avoided unless they are clearly justified.

Before adding a dependency, consider:

- license compatibility with Apache-2.0;
- long-term maintenance status;
- security history;
- transitive dependencies;
- whether the functionality can reasonably be implemented without it.

Do not add dependencies with GPL-only, AGPL-only, proprietary, unclear, or otherwise incompatible licensing.

## AI-Assisted Contributions

This policy is intended to keep reviews focused, reduce provenance uncertainty, and avoid additional maintainer workload.
It distinguishes between code and reports.

### Code

Pull requests, patches, tests and documentation created substantially by AI tools are not accepted.

Contributors are expected to understand and author the contributions they submit, and to be able to explain every
change in review. Limited help from tools, for example to search, to explain existing code or to check spelling, is
fine. If an AI tool contributed to a commit, disclose it with an `Assisted-by:` or `Co-authored-by:` trailer naming the
tool.

Do not add AI instruction or configuration files, such as `AGENTS.md`, `CLAUDE.md`, Copilot or Cursor rules, to a pull
request.

### Issues and discussions

Write in your own words. AI may assist your investigation, but every report must be understood, manually verified and
independently reproduced by its author. AI-generated, bulk-generated or mechanically reformatted reports will be closed
without investigation.

### Security reports

Security reports are assessed on their merits, whether or not AI tools were used. See the [security policy](SECURITY.md).

## Findings from Reviews and Fuzzing

If you or your organization have many findings, for example from a security review or a fuzzing campaign:

- report security-relevant findings privately as described in the [security policy](SECURITY.md), starting with a
  short list;
- report other problems as individual issues, one per root cause, each with a reproducer, not as a bulk filing;
- proposed patches are welcome as input. The maintainer decides how a problem is fixed and may implement the fix
  independently; reporters are credited, for example with a `Co-authored-by:` trailer or in a security advisory.
  Code incorporated from a contribution requires acceptance of the Developer Certificate of Origin;
- findings are handled in batches at the maintainer's pace. There is no guarantee that every finding is fixed, or fixed
  by a particular date. Reporting a finding creates no entitlement.

## Maintainer Discretion

The maintainers decide whether a contribution is accepted.

A contribution may be declined if it:

- does not fit the project direction;
- introduces unacceptable maintenance burden;
- lacks sufficient tests;
- has unclear licensing or provenance;
- duplicates existing functionality;
- changes public behavior in a problematic way;
- is too broad or difficult to review.

## Questions

For small issues, open a GitHub issue.

For larger design proposals, architectural changes, or contributions made on behalf of a company, please start with an issue or discussion before writing the implementation.
