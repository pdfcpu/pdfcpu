---
layout: default
title: "Set Page Layout"
---

# Set Page Layout

This command configures the page layout that shall be used when the document is opened.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagelayout set inFile value [ outFile ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------------------------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| value        | page layout mode    | yes
| outFile      | PDF output file, use `-` to write to stdout     | no

<br>

### Page Layouts

| name           | description
|:---------------|:-------------------------------------------------
| SinglePage     | Display one page at a time (default)
| TwoColumnLeft  | Display the pages in two columns, with odd-numbered pages on the left
| TwoColumnRight | Display the pages in two columns, with odd-numbered pages on the right
| TwoPageLeft    | Display the pages two at a time, with odd-numbered pages on the left
| TwoPageRight   | Display the pages two at a time, with odd-numbered pages on the right

<br>

## Examples

Set page layout for `test.pdf` (case agnostic):

```sh
$ pdfcpu pagelayout set test.pdf TwoColumnLeft
$ pdfcpu pagelayout list test.pdf
TwoColumnLeft
```

<br>

Set page layout while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-publishing/ebook.pdf - \
   | pdfcpu pagelayout set - TwoPageLeft - \
   | aws s3 cp - s3://acme-publishing/ebook-spreads.pdf
```
