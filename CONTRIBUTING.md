# Contributing to pdfcpu

Thank you for your interest in contributing to pdfcpu.

pdfcpu is an open-source PDF processor written in Go. The project is licensed under the Apache License, Version 2.0.

This document explains how contributions are handled and what is required before a contribution can be accepted.

## License

The pdfcpu project is licensed under the Apache License, Version 2.0.

By contributing to pdfcpu, you agree that your contribution is submitted under the Apache License, Version 2.0, unless explicitly stated otherwise in writing and accepted by the project maintainers.

Contributions may be used, modified, distributed, sublicensed, and otherwise handled as permitted by the Apache License, Version 2.0.

## Contributor License Agreement

pdfcpu uses a Contributor License Agreement process for pull requests.

Before a pull request can be accepted, the contributor must complete the CLA process as requested by the CLA Assistant check on GitHub. Pull requests may not be merged until the CLA Assistant check confirms that the required agreement has been completed.

By submitting a pull request and completing the CLA process, you confirm that you have the right to submit the contribution and to grant the rights described in the applicable CLA.

The CLA process is used to document the inbound rights needed by the project maintainers to review, modify, distribute, sublicense, and maintain contributions as part of the Apache-2.0 licensed project.

## Developer Certificate of Origin

pdfcpu may also accept `Signed-off-by` lines in commits as an additional provenance signal, but the CLA Assistant check is the authoritative contribution gate for pull requests unless the maintainers state otherwise.

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

If you believe you have found a security issue, please contact the maintainers privately first. Include enough information to reproduce and assess the problem, but do not include confidential third-party documents unless you have explicit permission to share them.

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
7. Complete the CLA Assistant process when requested.
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

AI-assisted contributions are not accepted.

Please do not submit pull requests, patches, generated code, documentation, tests, or issue comments that were created substantially by AI tools.

This policy is intended to keep reviews focused, reduce provenance uncertainty, and avoid additional maintainer workload.

Contributors are expected to understand and author the contributions they submit.

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
