---
layout: default
title: "Corpus Validation"
---

# Corpus Validation

Use these commands to assess a directory tree of PDFs. Run them from the corpus root and record the pdfcpu version
along with the results:

```sh
pdfcpu version
```

## Quick assessment

Validate all PDF files below the current directory:

```sh
pdfcpu validate -q "**/*.pdf"
```

The double quotes are important. They pass the recursive pattern to pdfcpu instead of asking the shell to expand it.
This avoids operating-system command-line length limits for large corpora.

## Show the active file

Add progress output when a file may take a long time to process:

```sh
pdfcpu validate -q --progress "**/*.pdf"
```

pdfcpu prints each filename before processing it, so the last progress line identifies the active file if a run stalls.
Validation errors are still reported immediately.

## Save the errors

Write progress and validation errors to a log file:

```sh
pdfcpu validate -q --progress "**/*.pdf" 2> validation.log
```

This redirection works in common Unix shells, PowerShell, and Windows Command Prompt. It replaces an existing log file.

## Choose the validation level

Relaxed validation is the default and is suitable for an initial corpus assessment. Use strict validation for a more
standards-focused pass:

```sh
pdfcpu validate -q --progress -m strict "**/*.pdf"
```

To assess resource optimization as well as validity, add `--opt`. This performs additional work and takes longer:

```sh
pdfcpu validate -q --progress --opt "**/*.pdf"
```

## Investigate one failure

Rerun a suspect file without quiet mode to see the full result:

```sh
pdfcpu validate -m strict path/to/problem.pdf
```

## Read the result

For a corpus run, pdfcpu reports each invalid file and finishes with a summary such as:

```text
validation failed: 66 of 463 files invalid
```

The process exits with status `0` only when every selected PDF validates. Check the status immediately after the
command using the syntax for your shell:

| Shell | Command |
| --- | --- |
| Bash or zsh | `echo $?` |
| PowerShell | `echo $LASTEXITCODE` |
| Windows Command Prompt | `echo %ERRORLEVEL%` |

The file count covers inputs matching `*.pdf`, not every file in the directory tree. Recursive input is processed in
pdfcpu's stable lexical order, which may differ from Finder or another file manager's display order.

## Reporting corpus results

A failed corpus run is only a starting point.<br>
Before opening an issue, reproduce one failure individually with the
latest release or prerelease and isolate one problem with the smallest shareable PDF.

Include the pdfcpu version, operating system, exact command, final summary, and the relevant error message. Attach large
logs as compressed files instead of pasting them, and remove sensitive filenames and paths.

### Automated and AI-generated issues

Corpus output is diagnostic input, not an issue backlog.<br>
Do not use AI, bots, or scripts to create issues in bulk or one issue per validation error.<br>
AI-generated, bulk-generated, or mechanically reformatted corpus reports will be closed immediately without
investigation.

**Repeated bulk submissions may be treated as spam.**

Every submitted report must be understood, manually verified, and independently reproduced by its author.

## Corpus investigations

Public issues are for isolated, reproducible pdfcpu problems.<br>
Reviewing an entire corpus, classifying and
prioritizing large result sets, investigating confidential or non-shareable files, and delivering fixes to an agreed
schedule are substantial engineering work and may require a paid engagement.

Limited corpus investigation and remediation work may be available by arrangement.

See [Validate](/core/validate) for the complete command reference.
