---
layout: default
title: "Merge"
---

# Merge

Merge 2 or more PDF files into `outFile`. Have a look at some [examples](#examples).

## Usage

```
pdfcpu merge outFile inFile... [flags]
```

<br>

### Flags

| name       | description                          | default   | required
|:-----------|:-------------------------------------|:----------|:--
| m(ode)     | create, append, zip                  | create    | no
| s(ort)     | sort inFiles if present              | unsorted  | no
| b(ookmarks)| create bookmarks                     | yes       | no
| bookmark-mode | wrap, preserve                    | wrap      | no
| d(ivider)  | insert separator between merged docs | no        | no
| opt        | optimize before writing              | yes       | no
| optimize   | optimize before writing              | yes       | no
| rmsig      | remove signatures                    | no        | no

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| outFile      | PDF output file, use `-` to write to stdout | yes
| inFile...    | at least 2 PDF input files subject to concatenation, use `-` for one stdin input in create mode | yes

<br>

## Restrictions

The following PDF elements are not carried over into the merged document:

* Struct Trees

<br>

## Bookmark Mode

By default `pdfcpu merge` creates bookmarks for the merged result.
The `--bookmark-mode` flag controls how input bookmarks are arranged in the output document.

| mode     | behavior
|:---------|:--------
| wrap     | Create one top-level bookmark per input file and place each input file's existing bookmarks below this filename wrapper.
| preserve | Preserve the input bookmark trees directly, without adding filename wrapper bookmarks.

`wrap` is the default and works well for document packages where each input file should remain visible as a top-level section.
For example, merging `cover.pdf`, `chapter1.pdf`, and `chapter2.pdf` produces top-level bookmarks named after the input files.
Any bookmarks already present in `chapter1.pdf` or `chapter2.pdf` appear below the corresponding file bookmark.

Use `preserve` when the input files already contain a carefully authored outline and the merged document should keep that outline structure without extra filename levels.
This is useful when merging chapters exported from the same source document, or when every input already uses consistent top-level bookmark titles.

`--bookmark-mode` only applies when bookmarks are enabled.
If you disable bookmark creation with `--bookmarks=false`, do not pass `--bookmark-mode`.

<br>

## Examples

pdfcpu respects the order of the provided input files and merges accordingly. Merge three input files into `out.pdf` by concatenating `in3.pdf` to `in2.pdf` and the result to `in1.pdf`:

```sh
$ pdfcpu merge out.pdf in1.pdf in2.pdf in3.pdf
```

<br>

Merge all PDF files in the current directory into `out.pdf` and don't create bookmarks:

```sh
$ pdfcpu merge out.pdf *.pdf -b=f
```

<br>

Merge files and keep the default bookmark wrapping.
The output outline gets one top-level bookmark for each input file:

```sh
$ pdfcpu merge book.pdf cover.pdf chapter1.pdf chapter2.pdf
```

Resulting outline:

```text
cover.pdf
chapter1.pdf
  Introduction
  Getting Started
chapter2.pdf
  Configuration
  Examples
```

<br>

Merge files while preserving their existing bookmark trees directly.
No filename wrapper bookmarks are added:

```sh
$ pdfcpu merge --bookmark-mode preserve book.pdf chapter1.pdf chapter2.pdf
```

Resulting outline:

```text
Introduction
Getting Started
Configuration
Examples
```

<br>

Merge some PDF files into an existing PDF file `out.pdf` and create divider pages between the merged documents:

```sh
$ pdfcpu merge --mode append -divider out.pdf in1.pdf in2.pdf in3.pdf
```

<br>

Zip two files together (eg. like in 1a,1b,2a,2b..):
```sh
$ pdfcpu merge --mode zip out.pdf a.pdf b.pdf
```

<br>

Merge local PDFs and upload the result:

```sh
$ pdfcpu merge - quarterly/*.pdf \
   | aws s3 cp - s3://acme-reports/quarterly/merged.pdf
```

<br>

Merge one PDF streamed from S3 with local files:

```sh
$ aws s3 cp s3://acme-reports/cover.pdf - \
   | pdfcpu merge - - chapter1.pdf chapter2.pdf \
   | aws s3 cp - s3://acme-reports/book.pdf
```
