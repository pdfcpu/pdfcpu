---
layout: default
title: "Split"
---

# Split

Generate a set of PDF files for `inFile` in `outDir` according to given `span` value. Also check out the [extract pages](/extract/extract_pages) command which gives you similar functionality. Have a look at some [examples](#examples).

## Usage

```
pdfcpu split inFile outDir [ span | pageNr... ] [flags]
```

<br>

### Flags

| name       | required | value    | description
|:-----------|:---------|:---------|:-----------
| m(ode)     | no       | span     | Split into PDF files with span pages each (default).
|            |          | bookmark | Split into PDF files representing sections defined by existing bookmarks.
|            |          | page     | Split before specific page number(s).

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required | default
|:-------------|:--------------------|:---------|:-
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outDir       | output directory    | yes
| span         | split span in pages | no       | 1
| pageNr...    | page numbers at which a new output file starts in `page` mode | yes in `page` mode |

In `span` mode, supply at most one positive span.
In `bookmark` mode, do not supply a span or page numbers.
In `page` mode, page numbers must be positive, unique, and sorted in ascending order.

<br>

## Restrictions

The following PDF elements are not carried over into the output files:

* Annotations
* Outlines
* Struct Trees
* Forms

<br>

## Examples

Split a PDF file into single page PDF files in `out`:
```sh
$ pdfcpu split test.pdf out
``` 

<br>

Split a PDF file into individual PDF files for every sheet of paper. Every PDF output file in `out` spans 2 pages of the original:
```sh
$ pdfcpu split test.pdf out 2
```

<br>

Split a PDF file along its bookmarks:
```sh
$ pdfcpu split test.pdf out -m bookmark
```

<br>

Split a PDF file before pages 2,4,10:
```sh
$ pdfcpu split -m page test.pdf out 2 4 10
```

<br>

Split selected pages from a PDF streamed from S3 into a local output directory:

```sh
$ aws s3 cp s3://acme-print/board-pack.pdf - \
   | pdfcpu split -m page - ./board-pack 10 25
```
